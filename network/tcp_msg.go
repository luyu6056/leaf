package network

import (
	//"bytes"

	"encoding/binary"
	"errors"
	"io"
	"math"
	"strconv"
	"sync"

	"github.com/luyu6056/gnet"
	"github.com/luyu6056/tls"
)

// --------------
// | len | data |
// --------------
type MsgParser struct {
	lenMsgLen      int
	minMsgLen      uint32
	maxMsgLen      uint32
	littleEndian   bool
	lenMsgLenInMsg bool
	msg_buf        []byte
}

func NewMsgParser() *MsgParser {
	p := new(MsgParser)
	p.lenMsgLen = 2
	p.minMsgLen = 1
	p.maxMsgLen = 4096
	p.littleEndian = false
	p.msg_buf = make([]byte, 4)
	return p
}

// It's dangerous to call the method on reading or writing
func (p *MsgParser) SetMsgLen(lenMsgLen int, minMsgLen uint32, maxMsgLen uint32) {
	if lenMsgLen == 1 || lenMsgLen == 2 || lenMsgLen == 4 {
		p.lenMsgLen = lenMsgLen
	}
	if minMsgLen != 0 {
		p.minMsgLen = minMsgLen
	}
	if maxMsgLen != 0 {
		p.maxMsgLen = maxMsgLen
	}

	var max uint32
	switch p.lenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}
	if p.minMsgLen > max {
		p.minMsgLen = max
	}
	if p.maxMsgLen > max {
		p.maxMsgLen = max
	}
	p.msg_buf = p.msg_buf[:p.lenMsgLen]
}

// It's dangerous to call the method on reading or writing
func (p *MsgParser) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

func (p *MsgParser) SetLenMsgLenInMsg(lenMsgLenInMsg bool) {
	p.lenMsgLenInMsg = lenMsgLenInMsg
}

//基于gnet接口的Decode
func (p *MsgParser) Decode(c gnet.Conn) (data []byte, err error) {
	if c.BufferLength() > 0 {
		data = c.Read()
		// parse len
		var msgLen uint32
		switch p.lenMsgLen {
		case 1:
			msgLen = uint32(data[0])
		case 2:
			if len(data) < 2 {
				return nil, nil
			}
			if p.littleEndian {
				msgLen = uint32(binary.LittleEndian.Uint16(data))
			} else {
				msgLen = uint32(binary.BigEndian.Uint16(data))
			}
		case 4:
			if len(data) < 4 {
				return nil, nil
			}
			if p.littleEndian {
				msgLen = binary.LittleEndian.Uint32(data)
			} else {
				msgLen = binary.BigEndian.Uint32(data)
			}
		}

		if p.lenMsgLenInMsg {
			msgLen -= uint32(p.lenMsgLen)
		}

		// check len
		if msgLen > p.maxMsgLen {
			return nil, errors.New("message too long msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
		} else if msgLen < p.minMsgLen {
			return nil, errors.New("message too short msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
		}
		//消息长度不够，meglen not enough
		if len(data[p.lenMsgLen:]) < int(msgLen) {
			return nil, nil
		}
		c.ShiftN(p.lenMsgLen + int(msgLen))
		return data[p.lenMsgLen : p.lenMsgLen+int(msgLen)], nil
	}
	return nil, nil
}
func (p *MsgParser) Encode(c gnet.Conn, data []byte) ([]byte, error) {
	msglen := uint32(len(data))
	if p.lenMsgLenInMsg {
		msglen += uint32(p.lenMsgLen)
	}
	if msglen > p.maxMsgLen {
		return nil, errors.New("message too long msgLen:" + strconv.Itoa(int(msglen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	} else if msglen < p.minMsgLen {
		return nil, errors.New("message too short msgLen:" + strconv.Itoa(int(msglen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	}
	buf := msgbufpool.Get().(*tls.MsgBuffer)
	buf.Reset()
	defer msgbufpool.Put(buf)
	switch p.lenMsgLen {
	case 1:
		buf.WriteByte(byte(msglen))
	case 2:
		b := buf.Make(2)
		if p.littleEndian {
			b[0] = byte(msglen)
			b[1] = byte(msglen >> 8)
		} else {
			b[1] = byte(msglen)
			b[0] = byte(msglen >> 8)
		}
	case 4:
		b := buf.Make(4)
		if p.littleEndian {
			b[0] = byte(msglen)
			b[1] = byte(msglen >> 8)
			b[2] = byte(msglen >> 16)
			b[3] = byte(msglen >> 24)
		} else {
			b[3] = byte(msglen)
			b[2] = byte(msglen >> 8)
			b[1] = byte(msglen >> 16)
			b[0] = byte(msglen >> 24)
		}
	}
	buf.Write(data)
	c.WriteNoCodec(buf.Bytes())
	return nil, nil
}

var msgbufpool = sync.Pool{New: func() interface{} {
	return &tls.MsgBuffer{}
}}

// goroutine safe
func (p *MsgParser) Write(conn *TCPConn, in []byte) error {
	// get len
	var msgLen = uint32(len(in))
	// check len
	if msgLen > p.maxMsgLen { //修改为一个包最大发送10M
		return errors.New("message too long msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	} else if msgLen < p.minMsgLen {
		return errors.New("message too short msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	}
	buffer := msgbufpool.Get().(*tls.MsgBuffer)
	buffer.Reset()
	defer msgbufpool.Put(buffer)
	msg := buffer.Make(p.lenMsgLen + int(msgLen))

	if p.lenMsgLenInMsg {
		msgLen += uint32(p.lenMsgLen)
	}

	// write len
	switch p.lenMsgLen {
	case 1:
		msg[0] = byte(msgLen)
	case 2:
		if p.littleEndian {
			binary.LittleEndian.PutUint16(msg, uint16(msgLen))
		} else {
			binary.BigEndian.PutUint16(msg, uint16(msgLen))
		}
	case 4:
		if p.littleEndian {
			binary.LittleEndian.PutUint32(msg, msgLen)
		} else {
			binary.BigEndian.PutUint32(msg, msgLen)
		}
	}

	// write data
	copy(msg[p.lenMsgLen:], in)
	_, err := conn.conn.Write(msg)
	return err
}

func (p *MsgParser) Read(conn *TCPConn, buf []byte) ([]byte, error) {
	if cap(buf) < p.lenMsgLen {
		buf = make([]byte, p.lenMsgLen)
	}
	bufMsgLen := buf[:p.lenMsgLen]
	// read len

	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}
	// parse len
	var msgLen uint32
	switch p.lenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLen[0])
	case 2:
		if p.littleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen))
		}
	case 4:
		if p.littleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen)
		}
	}

	if p.lenMsgLenInMsg {
		msgLen -= uint32(p.lenMsgLen)
	}

	// check len
	if msgLen > p.maxMsgLen {
		return nil, errors.New("message too long msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	} else if msgLen < p.minMsgLen {
		return nil, errors.New("message too short msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	}
	if cap(buf) < int(msgLen) {
		buf = make([]byte, msgLen)
	}
	// data
	msg := buf[:msgLen]
	if _, err := io.ReadFull(conn, msg); err != nil {
		return nil, err
	}

	return msg, nil
}
