package encode

import (
	"bytes"
	"encoding/binary"
	"io"
	"slices"
	"sync"
)

var encodeBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

func Decode(r io.Reader) ([]byte, error) {
	var l int64
	if err := binary.Read(r, binary.BigEndian, &l); err != nil {
		return nil, err
	}

	buf := make([]byte, l)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func Encode(data []byte) []byte {
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		encodeBufferPool.Put(buf)
	}()
	binary.Write(buf, binary.BigEndian, int64(len(data)))
	buf.Write(data)
	result := slices.Clone(buf.Bytes())
	return result
}
