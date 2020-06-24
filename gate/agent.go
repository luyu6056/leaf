package gate

import (
	"net"
)

type Agent interface {
	WriteMsg(msg []byte)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	UserData() interface{}
	SetUserData(data interface{})
}
