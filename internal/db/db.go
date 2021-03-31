package db

import (
	"sync"
	"time"
)

/* 使用map数据结构实现的缓存简单数据库 */

// type Handle map[string]map[string]string

type ttlStruce struct {
	T time.Time
	I string
}

type Db struct {
	// 使用map存储数据，仅单线程安全
	h map[string]map[string]string

	// 允许同时存在的ttl任务数，超出后设置的ttl将会失效
	TTLCap int64
	// 生存时间
	TTL time.Duration

	flag  bool           // 初始标志
	queue chan ttlStruce // 用于ttl
	lock  sync.RWMutex   // 写入锁
}

// 请用协程启动此程序以初始化
func (d *Db) Init() error {
	d.h = make(map[string]map[string]string)
	if d.TTLCap == 0 {
		d.TTLCap = 1 << 16
	}
	d.queue = make(chan ttlStruce, d.TTLCap)

	d.flag = true

	var r ttlStruce
	for {
		r = (<-(d.queue))

		time.Sleep(r.T.Sub(time.Now()))
		d.lock.RLock()
		delete(d.h, r.I)
		d.lock.RUnlock()
	}
}

// D 删除
func (d *Db) D(id string) {
	if d.flag {
		d.lock.RLock()
		delete(d.h, id)
		d.lock.RUnlock()
	}
}

// R 读取(没有将会返回空字符串)
func (d *Db) R(id, field string) string {
	if d.flag {
		return d.h[id][field]
	}
	return ""
}

// U 更新值(表不存在将不会记录)
func (d *Db) U(id, field, value string) {
	if d.flag {
		var t map[string]string = make(map[string]string)
		t = d.h[id]
		if t == nil {
			return
		}
		t[field] = value
		d.lock.RLock()
		d.h[id] = t
		d.lock.RUnlock()
	}
}

// Ut 更新表(表不存在将不会记录)
func (d *Db) Ut(id string, t map[string]string) {
	if d.flag {
		if d.h[id] == nil {
			return
		}
		d.lock.RLock()
		d.h[id] = t
		d.lock.RUnlock()
	}
}

// Ct 创造表(ttl及表的生存时间，请使用UTC时间)
func (d *Db) Ct(id string, t map[string]string, ttl ...bool) {
	if d.flag {
		var ct time.Time
		d.lock.RLock()
		if len(ttl) != 0 && ttl[0] {
			ct = time.Now().Add(d.TTL)
		}
		d.h[id] = t
		d.lock.RUnlock()

		if len(ttl) != 0 && ttl[0] { //设置ttl
			r := ttlStruce{
				ct,
				id,
			}
			d.queue <- r
		}
	}
}

// Et 表是否存在(异常情况返回false)
func (d *Db) Et(id string) bool {
	if d.flag {
		if d.h[id] == nil {
			return false
		}
		return true
	}
	return false
}

// 返回当前ttl任务数
func (d *Db) LenTTL() int64 {
	return int64(len(d.queue))
}
