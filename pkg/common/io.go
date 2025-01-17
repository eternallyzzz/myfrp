package common

import (
	"endpoint/pkg/zlog"
	"fmt"
	"golang.org/x/net/quic"
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
		if err != nil && !zlog.Ignore(err) {
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
		eventCh <- fmt.Sprintf("%s disconnected", addr)
	}
}

type Pipe struct {
	Stream *quic.Stream
}

func (p *Pipe) Read(b []byte) (n int, err error) {
	return p.Stream.Read(b)
}

func (p *Pipe) Write(b []byte) (n int, err error) {
	n, err = p.Stream.Write(b)
	p.Stream.Flush()
	return
}

func (p *Pipe) Close() error {
	return p.Stream.Close()
}
