package main

import (
	"fmt"
	"time"

	"github.com/lysShub/stun/internal/db"
)

func main() {

	dbHandl := new(db.Db)
	dbHandl.TTL = time.Hour
	go dbHandl.Init()

	time.Sleep(time.Millisecond * 10) //等待初始化完成

	var t map[string]string = map[string]string{
		"sdfsd":    "dsadfas",
		"dsfasdas": "啊啊啊啊",
	}
	dbHandl.Ct("1", t, true)
	dbHandl.Ct("2", t)

	fmt.Println(dbHandl.R("1", "sdfsd"))
	time.Sleep(time.Millisecond * 1001)
	fmt.Println(dbHandl.R("1", "sdfsd"))
	fmt.Println(dbHandl.R("2", "啊啊啊啊"))

}
