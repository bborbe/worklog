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
	"runtime"
	"github.com/bborbe/io"
	"os"
	"regexp"
	"strings"
)

var dirPtr = flag.String("dir", "", "git dir")
var authorPtr = flag.String("author", "", "name to match")

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := do(
		context.Background(),
		os.Stdout,
		*dirPtr,
		*authorPtr,
	); err != nil {
		glog.Exit(err)
	}
}

func do(ctx context.Context, out io.Writer, dir string, author string) error {
	glog.V(4).Infof("run worklog started")

	var wg sync.WaitGroup

	commitsChan := make(chan commit, 10)
	commandOutputChan := make(chan []byte, 10)

	// print my commits
	wg.Add(1)
	go func() {
		defer wg.Done()
		for commit := range commitsChan {
			if strings.Index(commit.Author, author) != -1 {
				fmt.Fprintln(out, commit.String())
			}
		}
	}()

	// parse commits from byte
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(commitsChan)
		buf := &bytes.Buffer{}
		for content := range commandOutputChan {
			buf.Write(content)
			if err := consumeCommit(commitsChan, buf); err != nil {
				glog.Exitf("consume commit failed: %v", err)
			}
		}
	}()

	// read git log
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := readGitLog(dir, commandOutputChan); err != nil {
			glog.Exitf("read git commit failed: %v", err)
		}
	}()
	wg.Wait()

	glog.V(4).Infof("run worklog finished")
	return nil
}

func readGitLog(dir string, commandOutputChan chan<- []byte) error {
	defer close(commandOutputChan)
	glog.V(4).Infof("read git %s started", dir)
	cmd := exec.Command("git", "log", "--raw")
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

func consumeCommit(l chan<- commit, buffer *bytes.Buffer) error {
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
}

func (c *commit) String() string {
	return fmt.Sprintf("%s %s", c.Date.Format("2006-01-02T15:04:05"), c.Message)
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
