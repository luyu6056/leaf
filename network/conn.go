package network

import (
	//"bytes"
	"net"
)

type Conn interface {
	ReadMsg(buf []byte) ([]byte, error)
	WriteMsg(args ...[]byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()
}
