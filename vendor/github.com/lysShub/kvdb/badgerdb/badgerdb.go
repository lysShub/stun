package badgerdb

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/lysShub/kvdb/com"

	badger "github.com/dgraph-io/badger/v2"
)

// Handle db handle
type Handle = *badger.DB

// Badger badgerdb
// badgerdb中没有表的概念，使用前缀实现，使用Delimiter区分前缀和字段
type Badger struct {
	DbHandle  Handle   //必须，数据库句柄
	Path      string   //储存路径，默认路径文当前路径db文件夹
	Password  [16]byte //密码，默认无密码
	RAM       bool     //内存模式，默认false
	Delimiter string   //分割符，默认为字符`
}

var errStr error = errors.New("can not include delimiter character")
var err error

// OpenDb open db
func (d *Badger) OpenDb() error {
	if d.Path != "" { // 设置路径
		fi, err := os.Stat(d.Path)
		if err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(d.Path, 0666); err != nil {
					return err
				}
			} else {
				return err
			}
		} else if !fi.IsDir() { //存在同名文件
			if err = os.MkdirAll(d.Path, 0600); err != nil {
				return err
			}
		}
	} else { // 设置为默认路径
		var path string = com.GetExePath() + `\db\`
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				if err = os.Mkdir(path, 0600); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		d.Path = path
	}

	var opts badger.Options
	if d.RAM {
		opts = badger.DefaultOptions("").WithInMemory(true)
	} else {
		opts = badger.DefaultOptions(d.Path)
	}
	opts = opts.WithLoggingLevel(badger.ERROR)

	if d.Password[:] != nil {
		opts.EncryptionKey = d.Password[:]
	}
	if d.Delimiter == "" {
		d.Delimiter = "`"
	}
	opts.ValueLogFileSize = 1 << 29 //512MB

	db, err := badger.Open(opts)
	d.DbHandle = db
	return err
}

// CloseDb close
func (d *Badger) Close() error {
	return d.DbHandle.Close()
}

// checkkey
func (d *Badger) checkkey(ks ...string) bool {
	for _, k := range ks {
		if strings.Contains(k, d.Delimiter) {
			return false
		}
		// for _, v := range k {
		// 	if string(v) == d.Delimiter {
		// return false
		// 	}
		// }
	}
	return true
}

// key/value

// SetKey
func (d *Badger) SetKey(key string, value []byte, ttl ...time.Duration) error {
	if !d.checkkey(key) {
		return errStr
	}
	txn := d.DbHandle.NewTransaction(true)
	defer txn.Discard()

	if len(ttl) == 0 {
		if err = txn.Set([]byte(key), value); err != nil {
			return err
		}
	} else {
		if err = txn.SetEntry(
			badger.NewEntry([]byte(key), value).WithTTL(ttl[0])); err != nil {
			return err
		}
	}
	return txn.Commit()
}

// DeleteKey
func (d *Badger) DeleteKey(key string) error {
	if !d.checkkey(key) {
		return errStr
	}
	txn := d.DbHandle.NewTransaction(true)
	defer txn.Discard()

	if err = txn.Delete([]byte(key)); err != nil {
		return err
	}
	return txn.Commit()
}

// ReadKey
func (d *Badger) ReadKey(key string) []byte {
	if !d.checkkey(key) {
		return nil
	}
	txn := d.DbHandle.NewTransaction(false)
	defer txn.Discard()

	var item *badger.Item
	if item, err = txn.Get([]byte(key)); err != nil {
		return nil
	}

	if err = txn.Commit(); err != nil {
		return nil
	}

	var valCopy []byte
	if valCopy, err = item.ValueCopy(nil); err != nil {
		return nil
	}
	return valCopy
}

// table

// SetTable
func (d *Badger) SetTable(tableName string, t map[string]map[string][]byte, ttl ...time.Duration) error {
	if !d.checkkey(tableName) {
		return errStr
	}

	txn := d.DbHandle.NewTransaction(true)
	defer txn.Discard()

	for id, kv := range t {
		if !d.checkkey(id) {
			return errStr
		}
		if !d.checkkey(id) {
			return errStr
		}
		for k, v := range kv {
			if !d.checkkey(k) {
				return errStr
			}
			if len(ttl) == 0 {
				if err = txn.Set([]byte(tableName+d.Delimiter+id+d.Delimiter+k), v); err != nil {
					return err
				}
			} else {
				if err = txn.SetEntry(badger.NewEntry([]byte(tableName+d.Delimiter+id+d.Delimiter+k), v).WithTTL(ttl[0])); err != nil {
					return err
				}
			}
		}
	}
	return txn.Commit()
}

// SetTableRow
func (d *Badger) SetTableRow(tableName, id string, kv map[string][]byte, ttl ...time.Duration) error {
	if !d.checkkey(tableName, id) {
		return errStr
	}
	txn := d.DbHandle.NewTransaction(true)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	for k, v := range kv {
		if !d.checkkey(k) {
			return errStr
		}
		if len(ttl) == 0 {
			if err = txn.Set([]byte(tableName+d.Delimiter+id+d.Delimiter+k), []byte(v)); err != nil {
				return err
			}
		} else {
			if err = txn.SetEntry(badger.NewEntry([]byte(tableName+d.Delimiter+id+d.Delimiter+k), []byte(v)).WithTTL(ttl[0])); err != nil {
				return err
			}
		}
	}
	it.Close()
	return txn.Commit()
}

// SetTableValue
func (d *Badger) SetTableValue(tableName, id, field string, value []byte, ttl ...time.Duration) error {
	if !d.checkkey(tableName, id, field) {
		return errStr
	}
	txn := d.DbHandle.NewTransaction(true)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()
	if len(ttl) == 0 {
		if err = txn.Set([]byte(tableName+d.Delimiter+id+d.Delimiter+field), []byte(value)); err != nil {
			return err
		}
	} else {
		if err = txn.SetEntry(badger.NewEntry([]byte(tableName+d.Delimiter+id+d.Delimiter+field), []byte(value)).WithTTL(ttl[0])); err != nil {
			return err
		}
	}
	it.Close()
	return txn.Commit()
}

// DeleteTable
func (d *Badger) DeleteTable(tableName string) error {
	if !d.checkkey(tableName) {
		return errStr
	}
	txn := d.DbHandle.NewTransaction(true)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	prefix := []byte(tableName + d.Delimiter)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		if err = txn.Delete(it.Item().Key()); err != nil {
			return err
		}
	}
	it.Close()
	return txn.Commit()
}

// DeleteTableRow
func (d *Badger) DeleteTableRow(tableName, id string) error {
	if !d.checkkey(tableName, id) {
		return errStr
	}
	txn := d.DbHandle.NewTransaction(true)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	prefix := []byte(tableName + d.Delimiter + id + d.Delimiter)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		if err = txn.Delete(it.Item().Key()); err != nil {
			return err
		}
	}
	it.Close()
	return txn.Commit()
}

// ReadTable
func (d *Badger) ReadTable(tableName string) map[string]map[string][]byte {
	if !d.checkkey(tableName) {
		return nil
	}
	var r map[string]map[string][]byte = make(map[string]map[string][]byte)
	var sr map[string][]byte = make(map[string][]byte)
	var tmpID string = ""

	txn := d.DbHandle.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	var deByte, v []byte = []byte(d.Delimiter), nil
	prefix := []byte(tableName + d.Delimiter)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		rk := bytes.SplitN(it.Item().Key(), deByte, 3)
		if len(rk) != 3 {
			return nil
		}
		if string(rk[1]) != tmpID {
			if tmpID != "" {
				r[tmpID] = sr
			}
			sr = make(map[string][]byte)
			tmpID = string(rk[1])
		}
		if v, err = it.Item().ValueCopy(nil); err != nil {
			return nil
		}
		sr[string(rk[2])] = v
	}
	r[tmpID] = sr
	it.Close()

	return r
}

// ReadTableExist
func (d *Badger) ReadTableExist(tableName string) bool {
	if !d.checkkey(tableName) {
		return false
	}
	txn := d.DbHandle.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	prefix := []byte(tableName + d.Delimiter)

	R := false
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		R = true
	}
	it.Close()

	return R
}

// ReadTableRow
func (d *Badger) ReadTableRow(tableName, id string) map[string][]byte {
	if !d.checkkey(tableName, id) {
		return nil
	}
	var r map[string][]byte = make(map[string][]byte)
	txn := d.DbHandle.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	prefix := []byte(tableName + d.Delimiter + id + d.Delimiter)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		rk := bytes.SplitN(it.Item().KeyCopy(nil), []byte(d.Delimiter), 3)
		if len(rk) == 3 {
			r[string(rk[2])], err = it.Item().ValueCopy(nil)
			if err != nil {
				return nil
			}
		} else {
			return nil
		}

	}
	it.Close()

	return r
}

// ReadTableValue
func (d *Badger) ReadTableValue(tableName, id, field string) []byte {
	if !d.checkkey(tableName, id, field) {
		return nil
	}
	var r []byte
	txn := d.DbHandle.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	prefix := []byte(tableName + d.Delimiter + id + d.Delimiter + field)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {

		if r, err = it.Item().ValueCopy(nil); err != nil {
			return nil
		}
	}
	it.Close()

	return r
}

// ReadTableLimits
func (d *Badger) ReadTableLimits(tableName, field, exp string, value int) []string {
	if !d.checkkey(tableName, field) {
		return nil
	}
	var r []string
	txn := d.DbHandle.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer txn.Discard()

	prefix := []byte(tableName + d.Delimiter)

	var deByte, v []byte = []byte(d.Delimiter), nil
	var rs [][]byte
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		rs = bytes.SplitN(it.Item().Key(), deByte, 3)
		if len(rs) != 3 {
			return nil
		}
		if string(rs[2]) == field {
			if v, err = it.Item().ValueCopy(nil); err != nil {
				return nil
			}
			fag, err := com.ExpressionCalculate(exp, value, v)
			if err != nil {
				return nil
			}
			if fag {
				r = append(r, string(rs[1]))
			}
		}
	}

	it.Close()

	return r
}
