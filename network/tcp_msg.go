package network

import (
	//"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strconv"
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

// goroutine safe
func (p *MsgParser) Read(conn *TCPConn, buf []byte) ([]byte, error) {
	if len(buf) < p.lenMsgLen {
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
	if len(buf) < int(msgLen) {
		buf = make([]byte, msgLen)
	}
	// data
	msg := buf[:msgLen]
	if _, err := io.ReadFull(conn, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// goroutine safe
func (p *MsgParser) Write(conn *TCPConn, args ...[]byte) error {
	// get len
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	// check len
	if msgLen > /*p.maxMsgLen*/ 10*1024*1024 { //修改为一个包最大发送10M
		return errors.New("message too long msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	} else if msgLen < p.minMsgLen {
		return errors.New("message too short msgLen:" + strconv.Itoa(int(msgLen)) + " maxMsgLen:" + strconv.Itoa(int(p.maxMsgLen)))
	}

	msg := make([]byte, uint32(p.lenMsgLen)+msgLen)

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
	l := p.lenMsgLen
	for i := 0; i < len(args); i++ {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	conn.Write(msg)

	return nil
}
