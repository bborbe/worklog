package run

import (
	"context"

	"sync"

	"github.com/bborbe/run/errors"
	"github.com/golang/glog"
)

type run func(context.Context) error

func CancelOnFirstFinish(ctx context.Context, runners ...run) error {
	if len(runners) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errors := make(chan error)
	for _, runner := range runners {
		run := runner
		go func() {
			errors <- run(ctx)
		}()
	}
	return <-errors
}

func All(ctx context.Context, runners ...run) error {
	if len(runners) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errorChannel := make(chan error, len(runners))
	var errorWg sync.WaitGroup
	var runWg sync.WaitGroup
	var errs []error
	errorWg.Add(1)
	go func() {
		defer errorWg.Done()
		for err := range errorChannel {
			errs = append(errs, err)
		}
	}()

	for _, runner := range runners {
		run := runner
		runWg.Add(1)
		go func() {
			defer runWg.Done()
			if err := run(ctx); err != nil {
				errorChannel <- err
			}
		}()
	}
	glog.V(4).Infof("wait on runs")
	runWg.Wait()
	close(errorChannel)
	glog.V(4).Infof("wait on error collect")
	errorWg.Wait()
	glog.V(4).Infof("run all finished")
	if len(errs) > 0 {
		glog.V(4).Infof("found %d errors", len(errs))
		return errors.New(errs...)
	}
	glog.V(4).Infof("finished without errors")
	return nil
}
