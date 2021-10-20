package mapdb

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lysShub/mapdb/store"
)

/* 使用map数据结构实现的缓存简单数据库 */

type Db struct {
	// 一个实例存储一个表
	// 支持行的TTL

	Name string // 名称, 必须参数
	Log  bool   //在数据TTL删除之前尽可能进行持久化, 使用的boltdb记录

	m map[string]map[string]string //

	ch    chan string //
	press bool
	lock  sync.Mutex   // 锁
	s     *store.Store // 持久化删除日志

}

// NewMapDb
func NewMapDb(config func(*Db)) (*Db, error) {
	var d = new(Db)
	config(d)
	if err := d.init(); err != nil {
		return nil, err
	}
	return d, nil
}

// init 初始化
func (d *Db) init() error {

	if d.Log {
		path := filepath.Join(getExePath(), d.Name)
		fmt.Println(path)
		var err error
		if d.s, err = store.OpenDb(path); err != nil {
			return err
		}
	}
	d.m = make(map[string]map[string]string)
	d.ch = make(chan string, 1<<15)
	go func() {
		var id string
		for id = range d.ch {
			if d.press && len(d.ch) < 17 {
				d.press = false
			} else if !d.press && len(d.ch) > 7 {
				d.press = true
			}
			if len(d.ch) > 1<<15-10 {
				panic(len(d.ch))
			}

			if !d.press && d.Log { // 非压力中，记录持久化日志
				d.s.UpdateRow(id, d.m[id])
			}
			d.lock.Lock()
			delete(d.m, id)
			d.lock.Unlock()
		}
	}()
	return nil
}

// R 查，没有将会返回空字符串
func (d *Db) R(id, field string) string {
	return d.m[id][field]
}

// U 更新值
func (d *Db) U(id, field, value string) {
	d.lock.Lock()
	if d.m[id] == nil {
		d.m[id] = map[string]string{}
		d.m[id][field] = value
	} else {
		d.m[id][field] = value
	}
	d.lock.Unlock()
}

func (d *Db) ReadRow(id string) map[string]string {
	return d.m[id]
}

// UpdateRow 更新一行
func (d *Db) UpdateRow(id string, t map[string]string) {
	d.lock.Lock()
	if d.m[id] != nil {
		for k, v := range t {
			d.m[id][k] = v
		}
	} else {
		d.m[id] = t
	}
	d.lock.Unlock()
}

// DeleteRow 删除一行
// 	实际删除操作可能会延后；并可能持久化到日志中
func (d *Db) DeleteRow(id string) {
	d.ch <- id
}

// ExitRow 行是否存在
func (d *Db) ExitRow(id string) bool {
	return d.m[id] != nil
}

// Drop 销毁
// 	如果设置Log, 数据将会持久化到日志中
func (d *Db) Drop() {

	d.lock.Lock() //
	defer d.lock.Unlock()
	if d.Log {
		for k, v := range d.m {
			if err := d.s.UpdateRow(k, v); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		d.s.Close()
	}
	d.m = nil
	d = nil
}

// getExePath 方法可执行文件(不包括文件名)所在路径
func getExePath() string {
	ex, err := os.Executable()
	if err != nil {
		exReal, err := filepath.EvalSymlinks(ex)
		if err != nil {
			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				return "./"
			}
			return dir
		}
		return filepath.Dir(exReal)
	}
	return filepath.Dir(ex)
}
