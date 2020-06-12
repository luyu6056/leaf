package dyb

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	//"github.com/name5566/leaf/log"
	//"runtime"
	//"runtime/debug"
	"sync"
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

func (p *BaseProcessor) Route(buf, buf1 []byte, userData interface{}) error {
	//这里捕获下异常吧,万一操作错误,免得玩家断线啥的

	var b []byte
	if p.is_aes {
		b = msg_aes_decrypt(buf, buf1)
	} else {
		b = buf
	}
	p.call_back(b, userData)
	return nil
}

// goroutine safe
func (p *BaseProcessor) Unmarshal(data []byte) ([]byte, error) {
	return data, nil
}

// goroutine safe
func (p *BaseProcessor) Marshal(msg interface{}, b []byte, buf *bytes.Buffer) ([][]byte, error) {
	if p.is_aes {
		buf.Reset()
		buf.Write(msg.([]byte))
		data := msg_aes_encrypt(b, buf)
		return [][]byte{data}, nil
	}
	return [][]byte{msg.([]byte)}, nil
}

func msg_aes_encrypt(b []byte, buf *bytes.Buffer) []byte {
	buf.Write(make([]byte, 16-buf.Len()%16))
	buf.Write(buf.Next(4))
	return aesEncrypt(b, buf.Bytes())
}

func msg_aes_decrypt(buf, buf1 []byte) []byte {
	bin, ok := aesDecrypt(buf, buf1)
	if !ok {
		return nil
	}
	length := len(bin)
	//msg := make([]byte, len(buf))
	copy(buf, bin[length-4:])
	copy(buf[4:], bin[:length-4])
	return buf

}

//var g_aeskey []byte = []byte("jin_tian_ni_chi_le_mei_you?chi_l")
var aes_pool = &sync.Pool{
	New: func() interface{} {
		block, _ := aes.NewCipher(*aeskey1) //g_aeskey[:16]
		return &block
	},
}

var aeskey1 = &[]byte{106, 105, 110, 95, 116, 105, 97, 110, 95, 110, 105, 95, 99, 104, 105, 95}
var aeskey2 = &[]byte{108, 101, 95, 109, 101, 105, 95, 121, 111, 117, 63, 99, 104, 105, 95, 108}

func aesEncrypt(b []byte, origData []byte) []byte {
	//block := <-aes_chan
	block := aes_pool.Get().(*cipher.Block)
	blockMode := cipher.NewCBCEncrypter(*block, *aeskey2) // g_aeskey[16:]
	if len(b) < len(origData) {
		b = make([]byte, len(origData))
	}
	crypted := b[:len(origData)]
	blockMode.CryptBlocks(crypted, origData)
	//aes_chan <- block
	aes_pool.Put(block)
	return crypted
}

func aesDecrypt(crypted, origData []byte) ([]byte, bool) {
	if len(crypted)%16 > 0 {
		return nil, false
	}
	if len(origData) < len(crypted) {
		origData = make([]byte, len(crypted))
	}
	//block := <-aes_chan
	block := aes_pool.Get().(*cipher.Block)
	blockMode := cipher.NewCBCDecrypter(*block, *aeskey2) // g_aeskey[16:]
	msg := origData[:len(crypted)]
	blockMode.CryptBlocks(msg, crypted)
	//aes_chan <- block
	aes_pool.Put(block)
	return msg, true
}
