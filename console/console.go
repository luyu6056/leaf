package console

import (
	"bufio"
	"bytes"
	"math"
	"strconv"
	"strings"

	"github.com/luyu6056/leaf/conf"
	"github.com/luyu6056/leaf/network"
)

var server *network.TCPServer

func Init() {
	if conf.ConsolePort == 0 {
		return
	}

	server = new(network.TCPServer)
	server.Addr = "localhost:" + strconv.Itoa(conf.ConsolePort)
	server.MaxConnNum = int(math.MaxInt32)
	server.NewAgent = newAgent

	server.Start()
}

func Destroy() {
	if server != nil {
		server.Close()
	}
}

type Agent struct {
	conn   network.Conn
	reader *bufio.Reader
}

func newAgent(conn network.Conn) network.Agent {
	a := new(Agent)
	a.conn = conn

	return a
}

func (a *Agent) OnInit() {

}

func (a *Agent) OnClose() {}
func (a *Agent) React(b []byte) {
	var n int
	for i := bytes.IndexByte(b[n:], '\n'); i > -1; i = bytes.IndexByte(b[n:], '\n') {
		line := string(b[n : n+i])
		line = strings.TrimSuffix(line[:len(line)-1], "\r")

		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		if args[0] == "quit" {
			break
		}
		var c Command
		for _, _c := range commands {
			if _c.name() == args[0] {
				c = _c
				break
			}
		}
		if c == nil {
			a.conn.WriteMsg([]byte("command not found, try `help` for help\r\n"))
			continue
		}
		output := c.run(args[1:])
		if output != "" {
			a.conn.WriteMsg([]byte(output + "\r\n"))
		}
	}
}
