package cluster

import (
	"math"
	"time"

	"github.com/luyu6056/leaf/conf"
	"github.com/luyu6056/leaf/network"
)

var (
	server  *network.TCPServer
	clients []*network.TCPClient
)

func Init() {
	if conf.ListenAddr != "" {
		server = new(network.TCPServer)
		server.Addr = conf.ListenAddr
		server.MaxConnNum = int(math.MaxInt32)
		server.LenMsgLen = 4
		server.MaxMsgLen = math.MaxUint32
		server.NewAgent = newAgent

		server.Start()
	}

	for _, addr := range conf.ConnAddrs {
		client := new(network.TCPClient)
		client.Addr = addr
		client.ConnNum = 1
		client.ConnectInterval = 3 * time.Second
		client.PendingWriteNum = conf.PendingWriteNum
		client.LenMsgLen = 4
		client.MaxMsgLen = math.MaxUint32
		client.NewAgent = newAgent

		client.Start()
		clients = append(clients, client)
	}
}

func Destroy() {
	if server != nil {
		server.Close()
	}

	for _, client := range clients {
		client.Close()
	}
}

type Agent struct {
	conn network.Conn
}

func newAgent(conn network.Conn) network.Agent {
	a := new(Agent)
	a.conn = conn
	return a
}

func (a *Agent) OnInit() {}

func (a *Agent) OnClose()       {}
func (a *Agent) React(b []byte) {}
