package network

import (
	"bytes"
)

type Processor interface {
	// must goroutine safe
	Route(buf, buf1 []byte, userData interface{}) error
	// must goroutine safe
	Unmarshal(buf []byte) ([]byte, error)
	// must goroutine safe
	Marshal(msg interface{}, b []byte, buf *bytes.Buffer) ([][]byte, error)
}
