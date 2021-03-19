package kvdb

import (
	"errors"
	"time"

	"github.com/lysShub/kvdb/badgerdb"
	"github.com/lysShub/kvdb/boltdb"
)

// Handle
type Handle struct {
	bg *badgerdb.Badger
	bt *boltdb.Bolt
}

// key/value database
type KVDB struct {
	// 有表结构存储；badger是使用前缀实现的，boltdb是使用桶嵌套实现的
	// 也有简单的key/value键值对储存；badger是没有前缀存储，boltdb是存储在特定的一个桶中
	// 在表中，每一条记录都有一此表中唯一id将各项字段联系起来，类似主键

	// 必须 0:badgerdb; 1:boltdb
	Type uint8
	// 只设置Type，Init将以默认参数打开
	DH Handle
	// 默认当前路径，badger是文件夹，boltdb是文件
	Path string
	/* 仅badgerdb */
	// 密码，默认无
	Password [16]byte
	// 内存模式、性能高很多，默认false
	RAMMode bool
	//分割符，默认`
	Delimiter string
	/* 仅boltdb */
	// 键值储存bucket名，默认_root
	Root []byte
}

// 将操作与预期不符将返回错误；如删除表中某一条记录，但此表不存在，将不会有错误信息返回
// 所有字段名使用string，所有值使用[]byte

var errType error = errors.New("kvdb.go: invalid value of KVDB.Type")

// 初始化函数
func (d *KVDB) Init() error {
	if d.Type == 0 { //badgerdb
		var b = new(badgerdb.Badger)

		b.Path = d.Path
		b.Password = d.Password
		b.RAM = d.RAMMode
		b.Delimiter = "`"
		if err := b.OpenDb(); err != nil {
			return err
		}
		d.DH.bg = b
		return nil
	} else if d.Type == 1 { //blotdb
		var b = new(boltdb.Bolt)
		b.Path = d.Path
		b.Root = d.Root
		if err := b.OpenDb(); err != nil {
			return err
		}
		d.DH.bt = b
		return nil
	}
	return errType
}

func (d *KVDB) Close() {
	if d.Type == 0 { //badgerdb
		d.DH.bg.Close()
	} else if d.Type == 1 { //blotdb
		d.DH.bt.Close()
	}
	return
}

// key/value 操作

// SetKey 设置/修改一个值
func (d *KVDB) SetKey(key string, value []byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetKey(key, value)
	} else if d.Type == 1 {
		return d.DH.bt.SetKey(key, value)
	}
	return errType
}

// DeleteKey 删除一个值
func (d *KVDB) DeleteKey(key string) error {
	if d.Type == 0 {
		return d.DH.bg.DeleteKey(key)
	} else if d.Type == 1 {
		return d.DH.bt.DeleteKey(key)
	}
	return errType
}

// ReadKey 读取一个值
func (d *KVDB) ReadKey(key string) []byte {
	if d.Type == 0 {
		return d.DH.bg.ReadKey(key)
	} else if d.Type == 1 {
		return d.DH.bt.ReadKey(key)
	}
	return nil
}

// 表操作

// SetTable 设置/修改一个表
func (d *KVDB) SetTable(tableName string, p map[string]map[string][]byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetTable(tableName, p)
	} else if d.Type == 1 {
		return d.DH.bt.SetTable(tableName, p)
	}
	return errType
}

// SetTableRow 设置/修改一个表中的一条记录
func (d *KVDB) SetTableRow(tableName, id string, p map[string][]byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetTableRow(tableName, id, p)
	} else if d.Type == 1 {
		return d.DH.bt.SetTableRow(tableName, id, p)
	}
	return errType
}

// SetTableValue 设置/修改一个表中的一条记录的某个字段的值
func (d *KVDB) SetTableValue(tableName, id, field string, value []byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetTableValue(tableName, id, field, value)
	} else if d.Type == 1 {
		return d.DH.bt.SetTableValue(tableName, id, field, value)
	}
	return errType
}

// DeleteTable 删除一个表
func (d *KVDB) DeleteTable(tableName string) error {
	if d.Type == 0 {
		return d.DH.bg.DeleteTable(tableName)
	} else if d.Type == 1 {
		return d.DH.bt.DeleteTable(tableName)
	}
	return errType
}

// DeleteTableRow 删除表中的某条记录
func (d *KVDB) DeleteTableRow(tableName, id string) error {
	if d.Type == 0 {
		return d.DH.bg.DeleteTableRow(tableName, id)
	} else if d.Type == 1 {
		return d.DH.bt.DeleteTableRow(tableName, id)
	}
	return errType
}

// ReadTable 读取一个表中所有数据
func (d *KVDB) ReadTable(tableName string) map[string]map[string][]byte {
	if d.Type == 0 {
		return d.DH.bg.ReadTable(tableName)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTable(tableName)
	}
	return nil
}

// ReadTableExist 表是否存在
func (d *KVDB) ReadTableExist(tableName string) bool {
	if d.Type == 0 {
		return d.DH.bg.ReadTableExist(tableName)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableExist(tableName)
	}
	return false
}

// ReadTableRow 读取表中一条记录
func (d *KVDB) ReadTableRow(tableName, id string) map[string][]byte {
	if d.Type == 0 {
		return d.DH.bg.ReadTableRow(tableName, id)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableRow(tableName, id)
	}
	return nil
}

// ReadTableValue 读取表中一条记录的某个字段值
func (d *KVDB) ReadTableValue(tableName, id, field string) []byte {
	if d.Type == 0 {
		return d.DH.bg.ReadTableValue(tableName, id, field)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableValue(tableName, id, field)
	}
	return nil
}

// ReadTableLimits 获取表中满足条件的所有id
func (d *KVDB) ReadTableLimits(tableName, field, exp string, value int) []string {
	if d.Type == 0 {
		return d.DH.bg.ReadTableLimits(tableName, field, exp, value)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableLimits(tableName, field, exp, value)
	}
	return nil
}
