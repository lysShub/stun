package mapdb

import (
	"sync"
	"time"

	"github.com/lysShub/tq"
)

/* 使用map数据结构实现的缓存简单数据库 */

type Db struct {
	// 使用map存储数据，暴露出来是为了可以持久化
	M map[string]map[string]string

	lock       sync.RWMutex // 写入锁
	initFianal sync.WaitGroup
	q          *tq.TQ // 时间任务队列
}

// Init 初始化
func (d *Db) Init() {
	d.initFianal.Add(1)

	d.M = make(map[string]map[string]string)

	d.q = new(tq.TQ)
	d.q.Run()

	var r interface{}
	go func() {
		d.initFianal.Done()
		for {
			r = (<-(d.q.MQ))
			v, ok := r.(string)
			if ok {
				d.lock.RLock()
				delete(d.M, v)
				d.lock.RUnlock()
			}
		}
	}()
	d.initFianal.Wait()
}

// D 删除
func (d *Db) D(id string) {
	d.lock.RLock()
	delete(d.M, id)
	d.lock.RUnlock()
}

// R 读取(没有将会返回空字符串)
func (d *Db) R(id, field string) string {
	d.lock.RLock()
	var r string = d.M[id][field]
	d.lock.RUnlock()
	return r
}

// U 更新值
func (d *Db) U(id, field, value string, ttl ...time.Duration) {

	d.lock.RLock()
	d.M[id][field] = value
	d.lock.RUnlock()
	// ttl
	if len(ttl) != 0 {
		d.q.Add(tq.Ts{
			T: time.Now().Add(ttl[0]),
			P: id,
		})
	}
}

// Ut 更新表
func (d *Db) Ut(id string, t map[string]string, ttl ...time.Duration) {

	if d.M[id] != nil {
		var r map[string]string = make(map[string]string)
		d.lock.RLock()
		for k, v := range d.M[id] {
			r[k] = v
		}
		for k, v := range t {
			r[k] = v
		}
		d.M[id] = r
		d.lock.RUnlock()
	} else {
		d.lock.RLock()
		d.M[id] = t
		d.lock.RUnlock()
	}

	// ttl
	if len(ttl) != 0 {
		d.q.Add(tq.Ts{
			T: time.Now().Add(ttl[0]),
			P: id,
		})
	}

}

// Et 表是否存在(异常情况返回false)
func (d *Db) Et(id string) bool {
	d.lock.RLock()
	if d.M[id] == nil {
		d.lock.RUnlock()
		return false
	}
	d.lock.RUnlock()
	return true
}
