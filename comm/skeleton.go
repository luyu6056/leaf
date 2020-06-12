package comm

import (
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/module"
)

func NewSkeleton() * module.Skeleton {
	skeleton := &module.Skeleton{
		GoLen:              10000,
		TimerDispatcherLen: 10000,
		AsynCallLen:        10000,
		ChanRPCServer:      chanrpc.NewServer(10000),
	}
	skeleton.Init()
	return skeleton
}
