package common

import (
	"endpoint/pkg/zlog"
	"fmt"
	"io"
	"sync"

	"golang.org/x/net/quic"
)

// bufferPool 用于重用缓冲区，减少 GC 压力
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024) // 32KB 缓冲区
	},
}

// Copy 双向复制数据，使用缓冲区池优化内存分配
func Copy(dst io.ReadWriteCloser, src io.ReadWriteCloser, eventCh chan string, addr string) {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	f := func(i, o io.ReadWriteCloser) {
		defer wg.Done()
		defer i.Close()
		defer o.Close()

		buf := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf)

		_, err := io.CopyBuffer(i, o, buf)
		if err != nil && !zlog.Ignore(err) {
			select {
			case errCh <- err:
			default:
			}
		}
	}

	wg.Add(2)
	go f(dst, src)
	go f(src, dst)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		zlog.Error(err.Error())
	}

	if eventCh != nil {
		select {
		case eventCh <- fmt.Sprintf("%s disconnected", addr):
		default:
		}
	}
}

// CopySingle 单向复制数据，使用缓冲区池
func CopySingle(dst io.Writer, src io.Reader) (int64, error) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	return io.CopyBuffer(dst, src, buf)
}

type Pipe struct {
	Stream *quic.Stream
}

func NewPipe(stream *quic.Stream) *Pipe {
	return &Pipe{
		Stream: stream,
	}
}

func (p *Pipe) Read(b []byte) (n int, err error) {
	return p.Stream.Read(b)
}

func (p *Pipe) Write(b []byte) (n int, err error) {
	return p.Stream.Write(b)
}

func (p *Pipe) Close() error {
	return p.Stream.Close()
}
