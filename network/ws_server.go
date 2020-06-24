package network

import (
	"bytes"
	"crypto/x509"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/luyu6056/tls"

	"github.com/luyu6056/gnet"
	"github.com/luyu6056/gnet/examples/codec"
	"github.com/luyu6056/leaf/log"
)

type WSServer struct {
	Addr        string
	MaxConnNum  int
	MaxMsgLen   uint32
	CertFile    string
	KeyFile     string
	CaFile      string
	HTTPTimeout time.Duration
	NewAgent    func(Conn) Agent
	wg          sync.WaitGroup
	ConnNum     int32
	gnet.EventHandler
	Close func()
}

func (server *WSServer) Start() {

	if server.MaxConnNum <= 0 {
		server.MaxConnNum = 100
		log.Release("invalid MaxConnNum, reset to %v", server.MaxConnNum)
	}

	if server.MaxMsgLen <= 0 {
		server.MaxMsgLen = 4096
		log.Release("invalid MaxMsgLen, reset to %v", server.MaxMsgLen)
	}
	if server.HTTPTimeout <= 0 {
		server.HTTPTimeout = 10 * time.Second
		log.Release("invalid HTTPTimeout, reset to %v", server.HTTPTimeout)
	}
	if server.NewAgent == nil {
		log.Fatal("NewAgent must not be nil")
	}

	var tlsconfig *tls.Config
	if server.CertFile != "" || server.KeyFile != "" {
		tlsconfig = &tls.Config{
			NextProtos:               []string{"h2", "http/1.1"},
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			},
		}
		server_cert, err := ioutil.ReadFile(server.CertFile)
		if err != nil {
			log.Fatal("读取服务器证书cert错误 err: %v", err)
		}
		server_key, err := ioutil.ReadFile(server.KeyFile)
		if err != nil {
			log.Fatal("读取服务器证书key错误 err: %v", err)
		}
		if ca, err := ioutil.ReadFile(server.CaFile); err == nil {
			ca = bytes.TrimLeft(ca, "\n")
			server_cert = bytes.Replace(server_cert, ca, nil, 1)
			server_cert = append(server_cert, ca...)
			certPool := x509.NewCertPool()
			if ok := certPool.AppendCertsFromPEM(ca); ok {
				tlsconfig.ClientCAs = certPool
			} else {
				log.Debug("ca_cert加载失败", err)
			}
		}

		cert, err := tls.X509KeyPair(server_cert, server_key)
		if err != nil {
			log.Fatal("tls.LoadX509KeyPair err: %v", err)
		}
		tlsconfig.Certificates = []tls.Certificate{cert}
	}

	option := []gnet.Option{gnet.WithTCPKeepAlive(time.Second * 600), gnet.WithCodec(&codec.Tlscodec{}), gnet.WithReusePort(true), gnet.WithOutbuf(1024), gnet.WithTls(tlsconfig), gnet.WithMultiOut(true)}
	go gnet.Serve(server, "tcp://"+server.Addr, option...)
}
func (server *WSServer) OnInitComplete(svr gnet.Server) gnet.Action {
	server.Close = svr.Close
	log.Release("leaf run websocks on " + server.Addr)
	return gnet.None
}
func (server *WSServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	num := int(atomic.AddInt32(&server.ConnNum, 1))
	if num >= server.MaxConnNum {
		log.Debug("too many connections")
		return nil, gnet.Close
	}

	return
}
func (server *WSServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	atomic.AddInt32(&server.ConnNum, -1)
	switch svr := c.Context().(type) {
	case *codec.WSconn:
		switch agent := svr.Ctx.(type) {
		case Agent:
			agent.OnClose()
		}
	}
	c.SetContext(nil)
	return
}
func (server *WSServer) React(data []byte, c gnet.Conn) (action gnet.Action) {
	switch svr := c.Context().(type) {
	case *codec.Httpserver:

		//tmp_ctx.Request.Body.Reset()
		//tmp_ctx.Request.Body.Write(data[len(data)-8 : len(data)])
		err := svr.Upgradews(c)
		if err != nil {
			action = gnet.Close
			return
		}
		if ws, ok := c.Context().(*codec.WSconn); ok {
			agent := server.NewAgent(gnetConn{c})
			agent.OnInit()
			ws.Ctx = agent
		} else {
			action = gnet.Close
			return
		}
		return
	case *codec.WSconn:
		if agent, ok := svr.Ctx.(Agent); ok {
			agent.React(data)
		}
	}
	return
}
