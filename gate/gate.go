package gate

import (
	"bytes"
	"github.com/name5566/leaf/chanrpc"
	//g "github.com/name5566/leaf/go"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/network"
	"net"
	"reflect"
	//"runtime"
	"runtime/debug"
	"strings"
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
	byte_chan      chan []byte
	byte_chan_look sync.Mutex
	ChanStop       bool
	agent_chan     chan *agent
	wg             sync.WaitGroup
}

func (gate *Gate) Run(closeSig chan bool) {
	var wsServer *network.WSServer
	//g := g.New(gate.PendingWriteNum)
	gate.agent_chan = make(chan *agent, gate.MaxConnNum)
	for i := 0; i < gate.MaxConnNum; i++ {
		gate.agent_chan <- &agent{}
	}

	if gate.WSAddr != "" {
		wsServer = new(network.WSServer)
		wsServer.Addr = gate.WSAddr
		wsServer.MaxConnNum = gate.MaxConnNum
		wsServer.PendingWriteNum = gate.PendingWriteNum
		wsServer.MaxMsgLen = gate.MaxMsgLen
		wsServer.HTTPTimeout = gate.HTTPTimeout
		wsServer.CertFile = gate.CertFile
		wsServer.KeyFile = gate.KeyFile
		wsServer.NewAgent = func(conn *network.WSConn) network.Agent {
			a := <-gate.agent_chan
			a.conn = conn
			a.gate = gate
			a.userData = nil
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			if len(a.byte_chan) == 0 {
				a.byte_chan = make(chan []byte, 1)
				a.byte_chan <- []byte{}
			}
			if len(a.buf_chan) == 0 {
				a.buf_chan = make(chan *bytes.Buffer, 1)
				a.buf_chan <- bytes.NewBuffer(nil)
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
		tcpServer.PendingWriteNum = gate.PendingWriteNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.LenMsgLenInMsg = gate.LenMsgLenInMsg
		tcpServer.ChanStop = gate.ChanStop
		tcpServer.NewAgent = func(conn *network.TCPConn) network.Agent {
			a := <-gate.agent_chan
			a.conn = conn
			a.gate = gate
			a.userData = nil
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			if len(a.byte_chan) == 0 {
				a.byte_chan = make(chan []byte, 2)
				a.byte_chan <- []byte{}
				a.byte_chan <- []byte{}
			}
			if len(a.buf_chan) == 0 {
				a.buf_chan = make(chan *bytes.Buffer, 2)
				a.buf_chan <- bytes.NewBuffer(nil)
				a.buf_chan <- bytes.NewBuffer(nil)
			}
			gate.wg.Add(1)
			return a
		}
	}

	gate.byte_chan = make(chan []byte, gate.MaxConnNum)

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
	conn      network.Conn
	gate      *Gate
	userData  interface{}
	byte_chan chan []byte
	buf_chan  chan *bytes.Buffer
}

func (gate *Gate) get_byte_chan() []byte {
	gate.byte_chan_look.Lock()
	defer gate.byte_chan_look.Unlock()
	if len(gate.byte_chan) == 0 {
		return []byte{}
	} else {
		return <-gate.byte_chan
	}
}
func (gate *Gate) put_byte_chan(b []byte) {
	gate.byte_chan_look.Lock()
	defer gate.byte_chan_look.Unlock()
	if len(gate.byte_chan) == cap(gate.byte_chan) {
		return
	}
	gate.byte_chan <- b
}
func (a *agent) Run() {
	buf := a.gate.get_byte_chan()
	buf1 := a.gate.get_byte_chan()
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			log.Error(string(stack))
		}
		a.gate.put_byte_chan(buf)
		a.gate.put_byte_chan(buf1)
	}()

	for {
		b, err := a.conn.ReadMsg(buf)
		if err != nil {
			if !strings.Contains(err.Error(), "closed") && !strings.Contains(err.Error(), "EOF") {
				log.Error("read message: %v", err)
			}
			break
		}
		if a.gate.Processor != nil {
			b, err := a.gate.Processor.Unmarshal(b)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}
			err = a.gate.Processor.Route(b, buf1, a)
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}
}

func (a *agent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		err := a.gate.AgentChanRPC.Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
	a.gate.agent_chan <- a
	a.gate.wg.Done()
}

func (a *agent) WriteMsg(msg interface{}) {
	if a.gate.Processor != nil {
		b := <-a.byte_chan
		buf := <-a.buf_chan
		defer func() {
			a.byte_chan <- b
			a.buf_chan <- buf
		}()
		data, err := a.gate.Processor.Marshal(msg, b, buf)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		/*go func() {
			r := reflect.ValueOf(a.UserData())
			i := 0
			for r.Kind() == reflect.Ptr && i < 10 { //linux出现跳死可能与此有关
				r = r.Elem()
				i++
			}
			if r.Kind() == reflect.Struct && r.Type().Name() == "PlayerAgentData" {
				f := r.MethodByName("Write_out_log")
				if f.Kind() == reflect.Func {
					params := make([]reflect.Value, 2) //参数
					params[0] = reflect.ValueOf(msg)   //参数设置为20
					params[1] = reflect.ValueOf(a.RemoteAddr().String())
					f.Call(params)
				}
				/*filed := r.FieldByName("Player")
				for filed.Kind() == reflect.Ptr {
					filed = filed.Elem()
				}

				if filed.Kind() != reflect.Invalid {

					filed = filed.MethodByName("Player")
					if filed.Kind() != reflect.Invalid {
						for filed.Kind() == reflect.Ptr {
							filed = filed.Elem()
						}

						log.Debug("用户名:%+v", filed.FieldByName("PlayerName").Interface())
					}
				}

			}
		}()*/
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
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

func (a *agent) Destroy() {
	a.conn.Destroy()
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}
