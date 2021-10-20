package store

// boltdb 持久化存储

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

type Store struct {
	// 一个文件只存储一个表

	handle *bolt.DB
}

// OpenDb open
// 	如果数据文件已经存在，则继续写入数据
func OpenDb(dbFilePath string) (*Store, error) {

	var db *bolt.DB
	var err error

	if dbFilePath, err = chechFilePath(dbFilePath); err != nil {
		return nil, err
	}
	db, err = bolt.Open(dbFilePath, 0644, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	} else {
		var b = new(Store)
		b.handle = db
		return b, nil
	}
}

// Close 关闭
func (b *Store) Close() {
	b.handle.Close()
}

// UpdateRow 更新行
func (d *Store) UpdateRow(id string, p map[string]string) error {

	return d.handle.Update(func(tx *bolt.Tx) error {
		if b, err := tx.CreateBucketIfNotExists([]byte(id)); err != nil {
			return err
		} else {
			for k, v := range p {
				if err = b.Put([]byte(k), []byte(v)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (d *Store) DeleteRow(id string) error {
	return d.handle.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(id))
	})
}

func (b *Store) ReadRow(id string) map[string]string {
	var r map[string]string = make(map[string]string)

	b.handle.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(id)); b == nil {
			r = nil // 没有此行
		} else {
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				r[string(k)] = string(v)
			}
		}
		return nil
	})

	return r
}

func chechFilePath(path string) (string, error) {

	if len(path) == 0 {
		return "", errors.New(`invalid filepath`)
	}

	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			return "", errors.New(`the filepath must path rather than floder dir`)
		}
	}

	dir, name := filepath.Dir(path), filepath.Base(path)

	if _, err := os.Stat(dir); err != nil { // 文件夹路径不存在
		return "", err
	}

	if len(name) > 128 {
		return "", errors.New(`the filename too long than 128 Bytes`)
	} else if strings.ContainsAny(name, `<>/\|:*?`) {
		return "", errors.New(`the filename cannot contain any of the following characters: \/:*?"<>|`)
	}
	return filepath.Join(dir, name), nil
}
