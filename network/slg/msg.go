package slg

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"

	//"encoding/binary"

	"runtime"
	"runtime/debug"
	"sync"

	"github.com/luyu6056/leaf/log"
)

type BaseProcessor struct {
	call_back func(interface{}, interface{})
	is_aes    bool
}

func NewBaseProcessor(call_back func(interface{}, interface{}), is_aes bool) *BaseProcessor {
	p := new(BaseProcessor)
	p.call_back = call_back
	p.is_aes = is_aes
	return p
}

func (p *BaseProcessor) Route(buf, buf1 *bytes.Buffer, userData interface{}) error {
	//这里捕获下异常吧,万一操作错误,免得玩家断线啥的
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			stack := debug.Stack()
			log.Error(string(stack))
		}
	}()
	var b []byte
	if p.is_aes {
		buf1.Reset()
		b = msg_aes_decrypt(buf, buf1)
	} else {
		b = buf.Bytes()
	}
	p.call_back(b, userData)
	return nil
}

var buf_pool = &sync.Pool{
	New: func() interface{} {
		b := bytes.NewBuffer(nil)
		b.Grow(1024 * 512)
		return b
	},
}

// goroutine safe
func (p *BaseProcessor) Unmarshal(buf *bytes.Buffer) error {
	return nil
}

// goroutine safe
func (p *BaseProcessor) Marshal(msg interface{}) ([][]byte, error) {
	if p.is_aes {
		buf := buf_pool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.Write(msg.([]byte))
		data := msg_aes_encrypt(buf)
		return [][]byte{data}, nil
	}
	return [][]byte{msg.([]byte)}, nil
}

func msg_aes_encrypt(buf *bytes.Buffer) []byte {
	buf.Write(make([]byte, 16-buf.Len()%16))
	buf.Write(buf.Next(4))
	return aesEncrypt(buf)
}

func msg_aes_decrypt(buf *bytes.Buffer, buf1 *bytes.Buffer) []byte {
	ok := aesDecrypt(buf, buf1)
	if !ok {
		return buf.Bytes()
	}
	bin := buf1.Bytes()
	length := len(bin)
	msg := make([]byte, buf.Len())
	copy(msg, bin[length-4:])
	copy(msg[4:], bin[:length-4])
	return msg

}

//var g_aeskey []byte = []byte("jin_tian_ni_chi_le_mei_you?chi_l")
/*var aes_pool = &sync.Pool{
	New: func() interface{} {

		if err != nil {
			return nil
		}
		return &block
	},
}*/
var aes_chan = make(chan *cipher.Block, runtime.NumCPU())

func init() {
	for i := 0; i < runtime.NumCPU(); i++ {
		block, err := aes.NewCipher([]byte{106, 105, 110, 95, 116, 105, 97, 110, 95, 110, 105, 95, 99, 104, 105, 95}) //g_aeskey[:16]
		if err != nil {
			log.Fatal("创建aes池出错")
		}
		aes_chan <- &block
	}
}
func aesEncrypt(origData *bytes.Buffer) []byte {
	block := <-aes_chan
	blockMode := cipher.NewCBCEncrypter(*block, []byte{108, 101, 95, 109, 101, 105, 95, 121, 111, 117, 63, 99, 104, 105, 95, 108}) // g_aeskey[16:]
	crypted := make([]byte, origData.Len())
	blockMode.CryptBlocks(crypted, origData.Bytes())
	buf_pool.Put(origData)

	aes_chan <- block
	return crypted
}

func aesDecrypt(crypted, origData *bytes.Buffer) bool {
	if crypted.Len()%16 > 0 {
		return false
	}
	block := <-aes_chan
	blockMode := cipher.NewCBCDecrypter(*block, []byte{108, 101, 95, 109, 101, 105, 95, 121, 111, 117, 63, 99, 104, 105, 95, 108}) // g_aeskey[16:]
	origData.Write(make([]byte, crypted.Len()))
	blockMode.CryptBlocks(origData.Bytes(), crypted.Bytes())
	aes_chan <- block
	return true
}
