package network

type Processor interface {
	// must goroutine safe
	Route(buf []byte, userData interface{}) error
	// must goroutine safe
	Unmarshal(buf []byte) ([]byte, error)
	// must goroutine safe
	Marshal(msg []byte) ([]byte, error)
}
