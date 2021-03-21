package boltdb

import (
	"os"
	"path/filepath"
	"time"

	"github.com/lysShub/kvdb/com"

	"github.com/boltdb/bolt"
)

// Handle handle
type Handle = *bolt.DB

type Bolt struct {
	DbHandle Handle //句柄
	Path     string //路径
	Root     []byte //key/value的bucket名，默认_root
}

var err error
var b *bolt.Bucket

// OpenDb open
func (d *Bolt) OpenDb() error {
	if d.Path != "" {
		_, err := os.Stat(filepath.Dir(d.Path))
		if err != nil {
			if os.IsNotExist(err) { // 路径不存在
				if err = os.MkdirAll(filepath.Dir(d.Path), 0600); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	} else { // 设置为默认路径
		var path string = com.GetExePath() + `/data.db`
		_, err := os.Stat(filepath.Dir(path))
		if err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(filepath.Dir(path), 0600); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		d.Path = path
	}
	if d.Root == nil {
		d.Root = []byte("_root")
	}

	db, err := bolt.Open(d.Path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	d.DbHandle = db
	return nil
}

// CloseDb close
func (d *Bolt) Close() error {
	return d.DbHandle.Close()
}

// key/value

// SetKey set or updata key/value
func (d *Bolt) SetKey(key string, value []byte) error {
	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		if b, err = tx.CreateBucketIfNotExists(d.Root); err != nil {
			return err
		}
		return b.Put([]byte(key), value)
	})
	return err
}

// DeleteKey delete key
func (d *Bolt) DeleteKey(key string) error {
	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(d.Root)
		if b == nil {
			return nil
		}
		return b.Delete([]byte(key))
	})
	return nil
}

// ReadKey
func (d *Bolt) ReadKey(key string) []byte {
	var r []byte
	err = d.DbHandle.View(func(tx *bolt.Tx) error {
		if b = tx.Bucket(d.Root); b == nil {
			return nil
		}
		_, r = b.Cursor().Seek([]byte(key))
		return nil
	})
	return r
}

// table

// SetTable
func (d *Bolt) SetTable(tableName string, p map[string]map[string][]byte) error {

	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		b, err = tx.CreateBucketIfNotExists([]byte(tableName))
		if err != nil {
			return err
		}

		var sb *bolt.Bucket
		for id, fv := range p {
			if sb, err = b.CreateBucketIfNotExists([]byte(id)); err != nil { //sb: secondary bucket
				return err
			}
			for f, v := range fv {
				err := sb.Put([]byte(f), v)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	return err
}

// SetTableRow
func (d *Bolt) SetTableRow(tableName, id string, fv map[string][]byte) error {
	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		b, err = tx.CreateBucketIfNotExists([]byte(tableName))
		if err != nil {
			return err
		}

		var sb *bolt.Bucket
		if sb, err = b.CreateBucketIfNotExists([]byte(id)); err != nil {
			return err
		}

		for f, v := range fv {
			if err = sb.Put([]byte(f), v); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// SetTableValue
func (d *Bolt) SetTableValue(tableName, id, field string, value []byte) error {

	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		b, err = tx.CreateBucketIfNotExists([]byte(tableName))
		if err != nil {
			return err
		}

		var sb *bolt.Bucket
		if sb, err = b.CreateBucketIfNotExists([]byte(id)); sb == nil {
			return err
		}

		return sb.Put([]byte(field), value)
	})
	return err
}

// DeleteTable
func (d *Bolt) DeleteTable(tableName string) error {
	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(tableName))
	})
	return err
}

// DeleteTableRow
func (d *Bolt) DeleteTableRow(tableName, id string) error {
	err = d.DbHandle.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil { // bucket not exist
			return nil
		}
		return b.DeleteBucket([]byte(id))
	})
	return err
}

// ReadTable
func (d *Bolt) ReadTable(tableName string) map[string]map[string][]byte {
	var r map[string]map[string][]byte = make(map[string]map[string][]byte)

	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil {
			return nil
		}
		var f func(buk *bolt.Bucket) error
		var tmp map[string][]byte = make(map[string][]byte)
		var beforeID string
		f = func(buk *bolt.Bucket) error {
			err := d.DbHandle.View(func(tx *bolt.Tx) error {
				var c *bolt.Cursor
				if buk == nil {
					c = tx.Cursor()
				} else {
					c = buk.Cursor()
				}
				for k, v := c.First(); k != nil; k, v = c.Next() {
					if k != nil {
						if v == nil {
							// id
							if beforeID != "" {
								r[beforeID] = tmp
								tmp = make(map[string][]byte)
							}
							beforeID = string(k)
							var buk2 *bolt.Bucket
							if buk == nil {
								buk2 = tx.Bucket(k)
							} else {
								buk2 = buk.Bucket(k)
							}
							f(buk2)
						} else {
							// field
							tmp[string(k)] = v
						}
					}
				}
				r[beforeID] = tmp
				return nil
			})
			return err
		}
		f(b)

		return nil
	})
	return r
}

// ReadTableExist
func (d *Bolt) ReadTableExist(tableName string) bool {
	var r bool
	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b != nil {
			r = true
		} else {
			r = false
		}
		return nil
	})
	return r
}

// ReadTableRow
func (d *Bolt) ReadTableRow(tableName, id string) map[string][]byte {
	var r map[string][]byte = make(map[string][]byte)
	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil {
			r = nil
			return nil
		}
		sb := b.Bucket([]byte(id))
		if sb == nil {
			r = nil
			return nil
		}
		c := sb.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			r[string(k)] = v
		}
		return nil
	})
	return r
}

// ReadTableRowExist
func (d *Bolt) ReadTableRowExist(tableName, id string) bool {
	var r bool = false
	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil {
			r = false
			return nil
		}
		sb := b.Bucket([]byte(id))
		if sb == nil {
			r = false
			return nil
		}
		c := sb.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			r = true
			break
		}
		return nil
	})
	return r

}

// ReadTableValue
func (d *Bolt) ReadTableValue(tableName, id, field string) []byte {
	var r []byte
	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil {
			return nil
		}
		var sb *bolt.Bucket
		if sb = b.Bucket([]byte(id)); sb == nil {
			return nil
		}
		r = sb.Get([]byte(field))
		return nil
	})
	return r
}

// ReadTableLimits
func (d *Bolt) ReadTableLimits1(tableName, field, exp string, value int) []string {
	var r []string
	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil {
			r = nil
			return nil
		}

		var c, sc *bolt.Cursor = b.Cursor(), nil
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if k != nil && v == nil {
				id := k

				sc = tx.Bucket(k).Cursor()
				for k, v := sc.First(); k != nil; k, v = sc.Next() {
					if string(k) == field {
						fag, err := com.ExpressionCalculate(exp, value, v)
						if err != nil {
							return nil
						}
						if fag {
							r = append(r, string(id))
						}

					}
				}
			}
		}
		return nil
	})

	return r
}

func (d *Bolt) ReadTableLimits(tableName, field, exp string, value int) []string {
	var r []string
	_ = d.DbHandle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		if b == nil {
			return nil
		}

		var f func(buk *bolt.Bucket) error
		var beforeID string
		f = func(buk *bolt.Bucket) error {
			err := d.DbHandle.View(func(tx *bolt.Tx) error {

				var c *bolt.Cursor
				if buk == nil {
					c = tx.Cursor()
				} else {
					c = buk.Cursor()
				}
				for k, v := c.First(); k != nil; k, v = c.Next() {
					if v == nil {
						// id
						beforeID = string(k)
						var buk2 *bolt.Bucket
						if buk == nil {
							buk2 = tx.Bucket(k)
						} else {
							buk2 = buk.Bucket(k)
						}
						f(buk2)
					} else {
						// field
						if string(k) == field {
							fag, err := com.ExpressionCalculate(exp, value, v)
							if err != nil {
								return nil
							}
							if fag {
								r = append(r, string(beforeID))
							}
						}
					}
				}
				return nil
			})
			return err
		}
		f(b)
		return nil
	})

	return r
}
