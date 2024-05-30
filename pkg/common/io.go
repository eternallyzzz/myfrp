package common

import (
	"endpoint/pkg/zlog"
	"fmt"
	"io"
	"sync"
)

func Copy(dst io.ReadWriteCloser, src io.ReadWriteCloser, eventCh chan string, addr string) {
	var errs []error
	var wg sync.WaitGroup
	f := func(i, o io.ReadWriteCloser) {
		defer wg.Done()
		defer i.Close()
		defer o.Close()

		_, err := io.Copy(i, o)
		if err != nil {
			errs = append(errs, err)
		}
	}

	wg.Add(2)
	go f(dst, src)
	go f(src, dst)
	wg.Wait()

	for _, err := range errs {
		zlog.Error(err.Error())
	}

	if eventCh != nil {
		eventCh <- fmt.Sprintf("%s=%d*", addr, 1)
	}
}
