package tq

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

// 确保初始化完成
var InitEnd sync.WaitGroup

func init() {
	InitEnd.Add(1)
}

type TQ struct {
	// 使用UTC时间，及不要有time.Now().Local()的写法；除非你知道将发生什么

	// 将按照预定时间返回消息；请及时读取，否则会阻塞以致影响后续任务
	MQ chan interface{}

	/* 内部 */
	chans map[int64](chan Ts) // 储存任务
	ends  map[int64]time.Time // 记录对应管道的最后一次任务的时间
	imr   chan Ts             //
	dcl   int                 // 默认任务管道容量
	cid   chan int64          // 传递id，表示新建了管道
	wc    sync.Mutex          // 读写锁
}

// Ts 表示一个任务
type Ts struct {
	T time.Time
	P interface{}
}

// Run 运行任务，阻塞函数，请使用协程运行。
func (t *TQ) Run() {

	t.imr = make(chan Ts, 64)
	t.cid = make(chan int64, 16)
	t.MQ = make(chan interface{}, 64)
	t.chans = make(map[int64](chan Ts))
	t.ends = make(map[int64]time.Time)
	t.dcl = 64

	// 执行任务
	go func() {
		for { // 新建了管道

			select {
			case i := <-t.cid:
				go t.exec(i)
			case <-time.After(time.Minute):
				// nothing
			}
			i := <-t.cid
			go t.exec(i)
		}
	}()

	InitEnd.Done() // 初始化完成

	// 分发任务
	for {
		var r Ts

		select {
		case r = <-t.imr:

			if len(t.ends) == 0 { // 第一次

				var sc chan Ts = make(chan Ts, t.dcl)
				var id = t.randId()

				t.chans[id] = sc
				t.ends[id] = r.T
				t.chans[id] <- r
				t.cid <- id
			} else {

				var flag bool = false
				for id, v := range t.ends {

					if r.T.After(v) && len(t.chans[id]) < t.dcl { //不需要新建管道

						t.chans[id] <- r
						t.ends[id] = r.T
						flag = true
						break
					}
				}
				// 需要新建管道
				if !flag {
					var sc chan Ts = make(chan Ts, t.dcl)
					var id = t.randId()

					t.chans[id] = sc
					t.ends[id] = r.T
					t.chans[id] <- r
					t.cid <- id
				}
			}

		case <-time.After(time.Minute):
			// nothing
		}

	}
}

// Add 增加任务
func (t *TQ) Add(r Ts) {
	t.imr <- r
}

// exec 执行任务
func (t *TQ) exec(id int64) {
	var ts Ts

	for {

		t.wc.Lock()
		// 执行完任务后应该退出
		if len(t.chans[id]) == 0 {

			delete(t.ends, id)  // 删除ends中记录
			close(t.chans[id])  // 关闭管道
			delete(t.chans, id) // 删除chans中记录

			t.wc.Unlock()
			return
		}
		t.wc.Unlock()

		ts = <-t.chans[id]
		time.Sleep(ts.T.Sub(time.Now())) //延时等待

		t.MQ <- ts.P
	}
}

// randId 随机数
func (t *TQ) randId() int64 {
	b := new(big.Int).SetInt64(int64(9999))
	i, err := rand.Int(rand.Reader, b)
	if err != nil {
		return 63
	}
	r := i.Int64() + time.Now().UnixNano()
	return r
}
