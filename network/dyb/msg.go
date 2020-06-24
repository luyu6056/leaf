package dyb

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"sync"

	"github.com/luyu6056/tls"
)

func READ_int8(bin []byte) (*int8, []byte) {
	data := int8(bin[0])
	return &data, bin[1:]
}
func READ_int16(bin []byte) (*int16, []byte) {
	data := int16(binary.LittleEndian.Uint16(bin[:2]))
	return &data, bin[2:]
}
func READ_int32(bin []byte) (*int32, []byte) {
	data := int32(binary.LittleEndian.Uint32(bin[:4]))
	return &data, bin[4:]
}
func READ_int64(bin []byte) (*int64, []byte) {
	data := int64(binary.LittleEndian.Uint64(bin[:8]))
	return &data, bin[8:]
}
func READ_string(bin []byte) (*string, []byte) {
	length, bin := READ_int16(bin)
	data := ""
	if *length > 0 {
		data = string(bin[:*length])
	}
	return &data, bin[*length:]
}
func WRITE_int8(data *int8, buf *bytes.Buffer) {
	buf.WriteByte(byte(*data))
}
func WRITE_int16(data *int16, buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, uint16(*data))
}
func WRITE_int32(data *int32, buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, uint32(*data))
}
func WRITE_int64(data *int64, buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, uint64(*data))
}
func WRITE_string(data *string, buf *bytes.Buffer) {
	length := int16(len(*data))
	WRITE_int16(&length, buf)
	buf.WriteString(*data)

}

var msgbufpool = sync.Pool{New: func() interface{} {
	return &tls.MsgBuffer{}
}}

type BaseProcessor struct {
	call_back func(interface{}, interface{})
	is_aes    bool
	buf       *bytes.Buffer
}

func NewBaseProcessor(call_back func(interface{}, interface{}), is_aes bool) *BaseProcessor {
	p := new(BaseProcessor)
	p.call_back = call_back
	p.is_aes = is_aes
	p.buf = bytes.NewBuffer(nil)
	return p
}

func (p *BaseProcessor) Route(in []byte, userData interface{}) error {
	//这里捕获下异常吧,万一操作错误,免得玩家断线啥的

	var b []byte
	if p.is_aes {
		b = msg_aes_decrypt(in)
	} else {
		b = in
	}
	p.call_back(b, userData)
	return nil
}

// goroutine safe
func (p *BaseProcessor) Unmarshal(data []byte) ([]byte, error) {
	return data, nil
}

// goroutine safe
func (p *BaseProcessor) Marshal(msg []byte) ([]byte, error) {
	if p.is_aes {
		return msg_aes_encrypt(msg), nil
	}
	return msg, nil
}

func msg_aes_encrypt(b []byte) []byte {
	if len(b)%16 > 0 {
		b = append(b, make([]byte, 16-len(b)%16)...)
	}
	buf := msgbufpool.Get().(*tls.MsgBuffer)
	defer msgbufpool.Put(buf)
	buf.Reset()
	buf.Write(b)
	buf.Write(buf.Next(4))
	return aesEncrypt(buf.Bytes(), b)
}

func msg_aes_decrypt(in []byte) []byte {
	buf, ok := aesDecrypt(in)
	defer msgbufpool.Put(buf)
	if !ok {
		return nil
	}
	length := buf.Len()
	//msg := make([]byte, len(buf))
	copy(in, buf.Bytes()[length-4:])
	copy(in[4:], buf.Bytes()[:length-4])
	return in

}

//var g_aeskey []byte = []byte("jin_tian_ni_chi_le_mei_you?chi_l")

var block, _ = aes.NewCipher(aeskey1) //g_aeskey[:16]

var aeskey1 = []byte{106, 105, 110, 95, 116, 105, 97, 110, 95, 110, 105, 95, 99, 104, 105, 95}
var aeskey2 = []byte{108, 101, 95, 109, 101, 105, 95, 121, 111, 117, 63, 99, 104, 105, 95, 108}

func aesEncrypt(in, out []byte) []byte {
	blockMode := cipher.NewCBCEncrypter(block, aeskey2) // g_aeskey[16:]
	blockMode.CryptBlocks(out, in)
	return out
}

func aesDecrypt(crypted []byte) (*tls.MsgBuffer, bool) {
	if len(crypted)%16 > 0 {
		return nil, false
	}
	out := msgbufpool.Get().(*tls.MsgBuffer)
	out.Reset()
	origData := out.Make(len(crypted))
	blockMode := cipher.NewCBCDecrypter(block, aeskey2) // g_aeskey[16:]
	blockMode.CryptBlocks(origData, crypted)
	return out, true
}
