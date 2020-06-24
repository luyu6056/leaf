package network

type Agent interface {
	OnInit()
	React([]byte)
	OnClose()
}
