package gate

import (
	"github.com/luyu6056/leaf/chanrpc"

	//g "github.com/luyu6056/leaf/go"
	"net"

	"github.com/luyu6056/leaf/log"
	"github.com/luyu6056/leaf/network"

	//"runtime"

	"sync"
	"time"
)

type Gate struct {
	MaxConnNum      int
	PendingWriteNum int
	MaxMsgLen       uint32
	Processor       network.Processor
	AgentChanRPC    *chanrpc.Server

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	// tcp
	TCPAddr        string
	LenMsgLen      int
	LittleEndian   bool
	LenMsgLenInMsg bool
	ChanStop       bool

	wg sync.WaitGroup
}

func (gate *Gate) Run(closeSig chan bool) {
	var wsServer *network.WSServer

	if gate.WSAddr != "" {
		wsServer = new(network.WSServer)
		wsServer.Addr = gate.WSAddr
		wsServer.MaxConnNum = gate.MaxConnNum
		wsServer.MaxMsgLen = gate.MaxMsgLen
		wsServer.HTTPTimeout = gate.HTTPTimeout
		wsServer.CertFile = gate.CertFile
		wsServer.KeyFile = gate.KeyFile
		wsServer.NewAgent = func(conn network.Conn) network.Agent {
			a := &agent{}
			a.conn = conn
			a.gate = gate
			a.userData = nil
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			gate.wg.Add(1)
			return a
		}
	}

	var tcpServer *network.TCPServer
	if gate.TCPAddr != "" {
		tcpServer = new(network.TCPServer)
		tcpServer.Addr = gate.TCPAddr
		tcpServer.MaxConnNum = gate.MaxConnNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.LenMsgLenInMsg = gate.LenMsgLenInMsg
		tcpServer.ChanStop = gate.ChanStop
		tcpServer.NewAgent = func(conn network.Conn) network.Agent {
			a := &agent{}
			a.conn = conn
			a.gate = gate
			a.userData = nil
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			gate.wg.Add(1)
			return a
		}
	}

	if wsServer != nil {
		wsServer.Start()
	}
	if tcpServer != nil {
		tcpServer.Start()
	}
	<-closeSig
	if wsServer != nil {
		wsServer.Close()
	}
	if tcpServer != nil {
		tcpServer.Close()
	}
	gate.wg.Wait()
}

func (gate *Gate) OnDestroy() {

}

type agent struct {
	//g         *g.Go
	//l         *g.LinearContext
	conn     network.Conn
	gate     *Gate
	userData interface{}
}

func (a *agent) React(b []byte) {

	if a.gate.Processor != nil {
		b, err := a.gate.Processor.Unmarshal(b)
		if err != nil {
			log.Debug("unmarshal message error: %v", err)
			return
		}
		err = a.gate.Processor.Route(b, a)
		if err != nil {
			log.Debug("route message error: %v", err)
			return
		}
	} else {
		log.Debug("agent not have a Processor")
		a.conn.Close()
	}

}

func (a *agent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		err := a.gate.AgentChanRPC.Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
	a.gate.wg.Done()
}

func (a *agent) WriteMsg(msg []byte) {
	if a.gate.Processor != nil {
		b, err := a.gate.Processor.Marshal(msg)
		if err != nil {
			log.Error("marshal message  error: %v", err)
		}
		a.conn.WriteMsg(b)
	} else {
		a.conn.WriteMsg(msg)
	}
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}
func (a *agent) OnInit() {

}
