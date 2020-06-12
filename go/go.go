package g

import (
	//"container/list"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
	"runtime"
	"sync"
	"time"
)

// one Go per goroutine (goroutine not safe)
type Go struct {
	ChanCb     chan *func()
	pendingGo  map[*func()]bool
	lock       sync.Mutex
	m_LinearGo sync.Map
}

var pool_LinearGo = sync.Pool{New: func() interface{} { return &LinearGo{} }}

type LinearGo struct {
	f  *func()
	cb *func()
}

type LinearContext struct {
	g       *Go
	go_chan chan *LinearGo
	//linearGo       []*LinearGo
	//mutexLinearGo  sync.Mutex
	//mutexExecution sync.Mutex
	c bool
}

func New(l int) *Go {
	g := new(Go)
	g.ChanCb = make(chan *func(), l)
	g.pendingGo = make(map[*func()]bool)
	return g
}

func (g *Go) Go(f func(), cb func()) {
	g.lock.Lock()
	ptr_cb := &cb
	g.pendingGo[ptr_cb] = true
	g.lock.Unlock()
	go func() {
		defer func() {
			g.ChanCb <- ptr_cb
			if r := recover(); r != nil {
				if conf.LenStackBuf > 0 {
					buf := make([]byte, conf.LenStackBuf)
					l := runtime.Stack(buf, false)
					log.Error("%v: %s", r, buf[:l])
				} else {
					log.Error("%v", r)
				}
			}
		}()

		f()
	}()
}

func (g *Go) Cb(cb *func()) {
	defer func() {
		g.lock.Lock()
		delete(g.pendingGo, cb)
		g.lock.Unlock()
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("%v: %s", r, buf[:l])
			} else {
				log.Error("%v", r)
			}
		}
	}()

	if cb != nil {
		(*cb)()
	}
}

func (g *Go) Close() {
	for len(g.pendingGo) > 0 {
		g.Cb(<-g.ChanCb)
	}
	g.m_LinearGo.Range(func(c, _ interface{}) bool {
		if c != nil {
			c.(*LinearContext).Close()
		}
		return true
	})
	time.Sleep(time.Second) //等待1秒，避免关闭时候重复执行太多次
}

func (g *Go) Idle() bool {
	var n int
	g.m_LinearGo.Range(func(c, _ interface{}) bool {
		n++
		return true
	})
	return len(g.pendingGo) == 0 && n == 0
}

var n int

func (g *Go) NewLinearContext() *LinearContext {

	c := new(LinearContext)
	c.g = g
	//c.linearGo = list.New()
	c.go_chan = make(chan *LinearGo, cap(g.ChanCb))
	g.m_LinearGo.Store(c, true)
	go c.guard_go()
	return c
}

func (c *LinearContext) guard_go() {
	var cb *func()
	defer func() {
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("%v: %s", r, buf[:l])
			} else {
				log.Error("%v", r)
			}
			if cb != nil {
				(*cb)()
			}
		}
		if (c.c && len(c.go_chan) == 0) || c.go_chan == nil { //关闭
			c.g.m_LinearGo.Delete(c)
		} else {
			c.guard_go() //报错后，输出错误，并重新进入守护进程
		}
	}()

	for e := range c.go_chan {
		cb = e.cb
		(*e.f)() //执行
		cb = nil
		(*e.cb)() //回调
		pool_LinearGo.Put(e)
	}
	/*for {
		c.mutexLinearGo.Lock()
		if len(c.linearGo) == 0 { //没有进程
			if c.c { //关闭标识
				c.mutexLinearGo.Unlock()
				break
			}
			c.mutexLinearGo.Unlock() //放开锁
			<-c.go_chan              //等待激活
			c.mutexLinearGo.Lock()
		}
		if len(c.linearGo) == 0 {
			c.mutexLinearGo.Unlock()
			continue
		}
		e := c.linearGo[0]
		cb = e.cb
		c.mutexLinearGo.Unlock()
		e.f()    //执行
		cb = nil //执行失败会在回收执行callback
		e.cb()   //回调
		pool_LinearGo.Put(e)
		c.mutexLinearGo.Lock()
		c.linearGo = c.linearGo[1:] //抛弃队列第一个
		c.mutexLinearGo.Unlock()
	}*/

}

func (c *LinearContext) Close() {
	c.c = true
	defer func() {
		recover()
	}()
	if len(c.go_chan) == 0 && c.go_chan != nil { //循环判断直到队列清空
		close(c.go_chan)
	}
}
func (c *LinearContext) Go(f func(), cb func()) {
	e := pool_LinearGo.Get().(*LinearGo)
	e.f = &f
	e.cb = &cb
	if c.c { //关闭了
		defer func() {
			recover()
		}()
	}
	c.go_chan <- e
}
