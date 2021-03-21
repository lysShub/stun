# Golang key/Value database api



inherited KV database:

- [badgerdb](https://github.com/dgraph-io/badger/v2)[boltdb](https://github.com/boltdb/bolt)

- [boltdb](https://github.com/boltdb/bolt)

Unified api make it easier to using and suppoer "table" struct data.

### Start

**GO111MODULE=on**

```
go get github.com/lysShub/kvdb
```

in your Golang project

```go
// main.go
package main

import (
	"fmt"
	"time"

	"github.com/lysShub/kvdb"
)

var pp map[string]map[string][]byte = map[string]map[string][]byte{
	"id1": {
		"field1": []byte("1111111111"),
		"field2": []byte("222222222222"),
	},
	"id2": {
		"field1": []byte("aaaaaaaaaaaaa"),
		"field2": []byte("19986"),
	},
	"id3": {
		"field1": []byte("aaaaaaaaaaaaaaa"),
		"field2": []byte("@@@@@@@@@@@@@@@@"),
	},
}

func main() {
	var err error
	var db = new(kvdb.KVDB)

	db.Type = 0
	db.RAMMode = true

	if err = db.Init(); err != nil {
		fmt.Println(0, err)
		return
	}
	defer db.Close()

	a := time.Now()
	if err = db.SetTable("test", pp); err != nil {
		fmt.Println(1, err)
		return
	}
	r := db.ReadTable("test")
	b := time.Since(a)

	fmt.Println(r)
	fmt.Println("use time：", b)
	return
}
```

```shell
go mod vendor
```

```shell
go build -o test test.go
```

```
./test
```

