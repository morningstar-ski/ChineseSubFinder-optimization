package downloader

import (
	"context"
	"fmt"
	"runtime/debug"
)

func runDownloaderErrorStep(ctx context.Context, fn func() error) (err error, panicked interface{}, canceled bool) {
	done := make(chan error, 1)
	panicChan := make(chan interface{}, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- fmt.Sprintf("%v\n%s", p, debug.Stack())
			}
		}()
		done <- fn()
	}()

	select {
	case err = <-done:
		return err, nil, false
	case panicked = <-panicChan:
		return nil, panicked, false
	case <-ctx.Done():
		return nil, nil, true
	}
}
