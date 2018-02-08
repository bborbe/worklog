package main

import (
	"github.com/golang/glog"
	"context"
	"os/exec"
	"fmt"
	"time"
	"bytes"
	"sync"
	flag "github.com/bborbe/flagenv"
	io_util "github.com/bborbe/io/util"
	"runtime"
	"github.com/bborbe/io"
	"os"
	"regexp"
	"strings"
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
	glog.V(4).Infof("run worklog started")

	var wgPrintCommits sync.WaitGroup
	var wgReadGitLog sync.WaitGroup

	commitsChan := make(chan commit, 10)

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

	for _, dir := range dirs {
		commandOutputChan := make(chan []byte, 10)

		normalizedDir, err := io_util.NormalizePath(dir)
		if err != nil {
			glog.Exitf("normalize path %s failed: %v", dir, err)
		}

		// parse commits from byte
		wgReadGitLog.Add(1)
		go func() {
			defer wgReadGitLog.Done()
			buf := &bytes.Buffer{}
			for content := range commandOutputChan {
				buf.Write(content)
				if err := consumeCommit(commitsChan, buf, normalizedDir); err != nil {
					glog.Exitf("consume commit of dir %s failed: %v", normalizedDir, err)
				}
			}
		}()

		// read git log
		wgReadGitLog.Add(1)
		go func() {
			defer wgReadGitLog.Done()
			if err := readGitLog(normalizedDir, days, commandOutputChan); err != nil {
				glog.Exitf("read git commits of dir %s failed: %v", normalizedDir, err)
			}
		}()
	}

	wgReadGitLog.Wait()
	close(commitsChan)
	wgPrintCommits.Wait()

	glog.V(4).Infof("run worklog finished")
	return nil
}

func readGitLog(dir string, days int, commandOutputChan chan<- []byte) error {
	defer close(commandOutputChan)
	glog.V(4).Infof("read git %s started", dir)
	cmd := exec.Command("git", "log", "--since", fmt.Sprintf("%d days ago", days), "--raw")
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

//var commitRegex = regexp.MustCompile(`(?is)\ncommit [^\n]\n.*?Author: ([^\n].*)\n.*?Date: ([^\n].*)\n.*?\n\n`)
var commitRegex = regexp.MustCompile(`(?is)commit .*?\nAuthor:\s+([^\n]+).*\nDate:\s+([^\n]+).*?\n    ([^\n]+)\n`)

func consumeCommit(l chan<- commit, buffer *bytes.Buffer, dir string) error {
	glog.V(4).Infof("consume commits started")

	content := buffer.Bytes()
	matches := commitRegex.FindAllSubmatch(content, -1)
	glog.V(4).Infof("found %d commits", len(matches))
	replaces := 0
	for _, match := range matches {
		replaces += len(match[0])
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
	}
	glog.V(4).Infof("truncate %d bytes from content", replaces)
	buffer.Truncate(replaces)

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
