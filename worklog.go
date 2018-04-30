package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	flag "github.com/bborbe/flagenv"
	"github.com/bborbe/io"
	io_util "github.com/bborbe/io/util"
	"github.com/bborbe/run/errors"
	"github.com/golang/glog"
)

var dirPtr = flag.String("dir", "", "git dir")
var authorPtr = flag.String("author", "", "name to match")
var daysPtr = flag.Int("days", 7, "days to print")

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	dirs := strings.Split(*dirPtr, ",")

	if err := do(
		context.Background(),
		os.Stdout,
		dirs,
		*authorPtr,
		*daysPtr,
	); err != nil {
		glog.Exit(err)
	}
}

func do(ctx context.Context, out io.Writer, dirs []string, author string, days int) error {
	glog.V(1).Infof("run worklog started")

	var wgPrintCommits sync.WaitGroup

	commitsChan := make(chan commit, 10)
	errorChan := make(chan error, 10)

	// print my commits
	wgPrintCommits.Add(1)
	go func() {
		defer wgPrintCommits.Done()
		for commit := range commitsChan {
			if strings.Index(commit.Author, author) != -1 {
				fmt.Fprintln(out, commit.String())
			}
		}
	}()

	var wgDir sync.WaitGroup
	for _, dir := range dirs {
		copyDir := dir
		wgDir.Add(1)
		go func() {
			defer wgDir.Done()
			if err := readCommits(commitsChan, copyDir, days); err != nil {
				glog.Infof("read commit failed: %v", err)
				errorChan <- err
			}
		}()
	}
	wgDir.Wait()

	close(commitsChan)
	wgPrintCommits.Wait()

	glog.V(1).Infof("run worklog finished")
	close(errorChan)
	if len(errorChan) > 0 {
		return errors.NewByChan(errorChan)
	}
	return nil
}

func readCommits(commitsChan chan commit, dir string, days int) error {
	commandOutputChan := make(chan []byte, 10)
	errorChan := make(chan error, 10)

	normalizedDir, err := io_util.NormalizePath(dir)
	if err != nil {
		return fmt.Errorf("normalize path %s failed: %v", dir, err)
	}
	var wgReadGitLog sync.WaitGroup

	// parse commits from byte
	wgReadGitLog.Add(1)
	go func() {
		defer wgReadGitLog.Done()
		buf := &bytes.Buffer{}
		buf.WriteString("\n")
		for content := range commandOutputChan {
			buf.Write(content)
		}
		if err := consumeCommit(commitsChan, buf, normalizedDir); err != nil {
			glog.Infof("consume commit failed: %v", err)
			errorChan <- fmt.Errorf("consume commit of dir %s failed: %v", normalizedDir, err)
		}
	}()

	// read git log
	wgReadGitLog.Add(1)
	go func() {
		defer wgReadGitLog.Done()
		if err := readGitLog(normalizedDir, days, commandOutputChan); err != nil {
			glog.Infof("read git log failed: %v", err)
			errorChan <- fmt.Errorf("read git commits of dir %s failed: %v", normalizedDir, err)
		}
	}()
	wgReadGitLog.Wait()

	close(errorChan)
	if len(errorChan) > 0 {
		return errors.NewByChan(errorChan)
	}
	return nil
}

func readGitLog(dir string, days int, commandOutputChan chan<- []byte) error {
	defer close(commandOutputChan)
	glog.V(4).Infof("read git %s started", dir)
	cmd := exec.Command("git", "log", "--all", "--since", fmt.Sprintf("%d days ago", days), "--raw")
	cmd.Dir = dir
	cmd.Stdout = NewLogParser(commandOutputChan)
	if glog.V(4) {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command failed: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait for command finish failed: %v", err)
	}
	glog.V(4).Infof("read git %s finished", dir)
	return nil
}

var commitRegex = regexp.MustCompile(`(?is)\ncommit\s+.*?\nAuthor:\s+([^\n]+).*?\nDate:\s+([^\n]+).*?\n    ([^\n]+)`)

func consumeCommit(l chan<- commit, buffer *bytes.Buffer, dir string) error {
	glog.V(4).Infof("consume commits started")

	matches := commitRegex.FindAllSubmatch(buffer.Bytes(), -1)
	glog.V(4).Infof("found %d commits", len(matches))
	for _, match := range matches {
		date, err := parseDate(string(match[2]))
		if err != nil {
			glog.Exitf("parse date failed: %v", err)
		}
		c := commit{
			Author:  fmt.Sprintf("%s", string(match[1])),
			Date:    date,
			Message: fmt.Sprintf("%s", string(match[3])),
			Dir:     dir,
		}
		l <- c
		if glog.V(6) {
			glog.Infof("match: %s\n", string(match[0]))
			glog.Infof("commit: %v", l)
		}
	}

	glog.V(4).Infof("consume commits finished")
	return nil
}

func parseDate(dateString string) (time.Time, error) {
	return time.Parse("Mon Jan 2 15:04:05 2006 -0700", dateString)
}

type commit struct {
	Author  string
	Message string
	Date    time.Time
	Dir     string
}

func (c *commit) String() string {
	parts := strings.Split(c.Dir, "/")
	return fmt.Sprintf("%s %s %s", c.Date.Format("2006-01-02T15:04:05"), parts[len(parts)-1], c.Message)
}

func NewLogParser(c chan<- []byte) *logParser {
	l := new(logParser)
	l.c = c
	return l
}

type logParser struct {
	c chan<- []byte
}

func (l *logParser) Write(p []byte) (int, error) {
	b := make([]byte, len(p))
	copy(b, p)
	l.c <- b
	return len(p), nil
}
