package network

import (
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/name5566/leaf/log"
)

type TCPClient struct {
	sync.Mutex
	Addr            string
	ConnNum         int
	ConnectInterval time.Duration
	PendingWriteNum int
	AutoReconnect   bool
	NewAgent        func(*TCPConn) Agent
	conns           ConnSet
	wg              sync.WaitGroup
	closeFlag       bool

	// msg parser
	LenMsgLen      int
	MinMsgLen      uint32
	MaxMsgLen      uint32
	LittleEndian   bool
	LenMsgLenInMsg bool // if the msg len contain in "header" len
	msgParser      *MsgParser
	Client_log     bool
	ChanStop       bool
}

func (client *TCPClient) Start() {
	client.init()

	for i := 0; i < client.ConnNum; i++ {
		client.wg.Add(1)
		go client.connect()
	}
}

func (client *TCPClient) init() {
	client.Lock()
	defer client.Unlock()

	if client.ConnNum <= 0 {
		client.ConnNum = 1
		log.Release("invalid ConnNum, reset to %v", client.ConnNum)
	}
	if client.ConnectInterval <= 0 {
		client.ConnectInterval = 3 * time.Second
		log.Release("invalid ConnectInterval, reset to %v", client.ConnectInterval)
	}
	if client.PendingWriteNum <= 0 {
		client.PendingWriteNum = 100
		log.Release("invalid PendingWriteNum, reset to %v", client.PendingWriteNum)
	}
	if client.NewAgent == nil {
		log.Fatal("NewAgent must not be nil")
	}
	if client.conns != nil {
		log.Fatal("client is running")
	}

	client.conns = make(ConnSet)
	client.closeFlag = false

	// msg parser
	msgParser := NewMsgParser()
	msgParser.SetMsgLen(client.LenMsgLen, client.MinMsgLen, client.MaxMsgLen)
	msgParser.SetByteOrder(client.LittleEndian)
	msgParser.SetLenMsgLenInMsg(client.LenMsgLenInMsg)
	client.msgParser = msgParser
}

func (client *TCPClient) dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", client.Addr)
		if err == nil || client.closeFlag {
			return conn
		}
		if client.Client_log {
			log.Error("connect to %v error: %v", client.Addr, err)
		}
		time.Sleep(client.ConnectInterval)
		continue
	}
}

func (client *TCPClient) connect() {
	defer client.wg.Done()

reconnect:
	try(func() {
		conn := client.dial()
		if conn == nil {
			return
		}

		client.Lock()
		if client.closeFlag {
			client.Unlock()
			conn.Close()
			return
		}
		client.conns[conn] = struct{}{}
		client.Unlock()

		tcpConn := newTCPConn(conn, client.PendingWriteNum, client.msgParser, client.ChanStop)
		agent := client.NewAgent(tcpConn)

		defer func() {
			// cleanup
			tcpConn.Close()
			client.Lock()
			delete(client.conns, conn)
			client.Unlock()
			agent.OnClose()

		}()

		agent.Run()
	})

	if client.AutoReconnect {
		time.Sleep(client.ConnectInterval)
		goto reconnect
	}
}

func (client *TCPClient) Close() {
	client.Lock()
	client.closeFlag = true
	for conn := range client.conns {
		conn.Close()
	}
	client.conns = nil
	client.Unlock()

	//client.wg.Wait()
}

func try(f func()) {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			log.Error(string(stack))
		}
	}()
	f()
}
