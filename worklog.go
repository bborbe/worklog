package main

import (
	"github.com/golang/glog"
	"context"
	"os/exec"
	"fmt"
	"time"
	"bytes"
	"sync"
)

func main() {
	err := do(context.Background())
	if err != nil {
		glog.Exit(err)
	}
}

func do(ctx context.Context) error {

	c := make(chan []byte, 10)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("git", "log", "--raw")
		cmd.Stdout = NewLogParser(c)
		if err := cmd.Start(); err != nil {
			fmt.Errorf("start command failed: %v", err)
		}
		if err := cmd.Wait(); err != nil {
			fmt.Errorf("wait for command finish failed: %v", err)
		}
	}()

	wg.Wait()

	return nil
}

type log struct {
	Author  string
	Message string
	Date    time.Time
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
