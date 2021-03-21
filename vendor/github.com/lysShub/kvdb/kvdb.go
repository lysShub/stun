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
	// have simple key value pair struct store
	// alse have table struct store; badger using prefix and boltdb using bucket nesting
	// in table, you need a id for a "row", similar "PrimaryKey"
	// all "name"(tableName,id,field) are string type, and all "value" are []byte type

	// must set; 0:badgerdb; 1:boltdb
	Type uint8
	// database handle
	DH Handle
	// default local path，badger is floder，boltdb id file
	Path string
	/* only for badgerdb */
	// password，default not have(nil)
	Password [16]byte
	// In memory mod, higher performance，default false
	RAMMode bool
	//delimit string, tableName and id can't contain it ;default ```
	Delimiter string
	/* only for boltdb */
	//key/value store's bucket name, default _root
	Root []byte
}

var errType error = errors.New("kvdb.go: invalid value of KVDB.Type")

// Init init function
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

// key/value operations

// SetKey create/update a value
func (d *KVDB) SetKey(key string, value []byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetKey(key, value)
	} else if d.Type == 1 {
		return d.DH.bt.SetKey(key, value)
	}
	return errType
}

// DeleteKey delete a value
func (d *KVDB) DeleteKey(key string) error {
	if d.Type == 0 {
		return d.DH.bg.DeleteKey(key)
	} else if d.Type == 1 {
		return d.DH.bt.DeleteKey(key)
	}
	return errType
}

// ReadKey read a value
func (d *KVDB) ReadKey(key string) []byte {
	if d.Type == 0 {
		return d.DH.bg.ReadKey(key)
	} else if d.Type == 1 {
		return d.DH.bt.ReadKey(key)
	}
	return nil
}

// table operations

// SetTable create/update a table
func (d *KVDB) SetTable(tableName string, p map[string]map[string][]byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetTable(tableName, p)
	} else if d.Type == 1 {
		return d.DH.bt.SetTable(tableName, p)
	}
	return errType
}

// SetTableRow create/update a record in a table
func (d *KVDB) SetTableRow(tableName, id string, p map[string][]byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetTableRow(tableName, id, p)
	} else if d.Type == 1 {
		return d.DH.bt.SetTableRow(tableName, id, p)
	}
	return errType
}

// SetTableValue create/update some one field's value in a table's some one record
func (d *KVDB) SetTableValue(tableName, id, field string, value []byte, ttl ...time.Duration) error {
	if d.Type == 0 {
		return d.DH.bg.SetTableValue(tableName, id, field, value)
	} else if d.Type == 1 {
		return d.DH.bt.SetTableValue(tableName, id, field, value)
	}
	return errType
}

// DeleteTable deleta a teble
func (d *KVDB) DeleteTable(tableName string) error {
	if d.Type == 0 {
		return d.DH.bg.DeleteTable(tableName)
	} else if d.Type == 1 {
		return d.DH.bt.DeleteTable(tableName)
	}
	return errType
}

// DeleteTableRow delete some one record in a table
func (d *KVDB) DeleteTableRow(tableName, id string) error {
	if d.Type == 0 {
		return d.DH.bg.DeleteTableRow(tableName, id)
	} else if d.Type == 1 {
		return d.DH.bt.DeleteTableRow(tableName, id)
	}
	return errType
}

// ReadTable read all date in a table
func (d *KVDB) ReadTable(tableName string) map[string]map[string][]byte {
	if d.Type == 0 {
		return d.DH.bg.ReadTable(tableName)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTable(tableName)
	}
	return nil
}

// ReadTableExist judge the table is exist
func (d *KVDB) ReadTableExist(tableName string) bool {
	if d.Type == 0 {
		return d.DH.bg.ReadTableExist(tableName)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableExist(tableName)
	}
	return false
}

// ReadTableRow read a record in a table
func (d *KVDB) ReadTableRow(tableName, id string) map[string][]byte {
	if d.Type == 0 {
		return d.DH.bg.ReadTableRow(tableName, id)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableRow(tableName, id)
	}
	return nil
}

// ReadTableRowExist judge a record is exist in a table
func (d *KVDB) ReadTableRowExist(tableName, id string) bool {
	if d.Type == 0 {
		return d.DH.bg.ReadTableRowExist(tableName, id)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableRowExist(tableName, id)
	}
	return false
}

// ReadTableValue read a field's value of some one record in a table
func (d *KVDB) ReadTableValue(tableName, id, field string) []byte {
	if d.Type == 0 {
		return d.DH.bg.ReadTableValue(tableName, id, field)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableValue(tableName, id, field)
	}
	return nil
}

// ReadTableLimits get all id that meeting the conditions
func (d *KVDB) ReadTableLimits(tableName, field, exp string, value int) []string {
	if d.Type == 0 {
		return d.DH.bg.ReadTableLimits(tableName, field, exp, value)
	} else if d.Type == 1 {
		return d.DH.bt.ReadTableLimits(tableName, field, exp, value)
	}
	return nil
}
