package common

import (
	"io"
	"sync"
)

func Forward(w io.Writer, r io.Reader, wg *sync.WaitGroup, flag chan struct{}) {
	defer func() {
		flag <- struct{}{}
		wg.Done()
	}()

	_, err := io.Copy(w, r)
	if err != nil {
		return
	}
}
