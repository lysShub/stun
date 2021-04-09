package com

import (
	"bytes"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"

	uuid "github.com/satori/go.uuid"
)

func Errlog(err ...error) bool {
	var haveErr bool = false
	for i, e := range err {
		if e != nil {
			haveErr = true
			_, fp, ln, _ := runtime.Caller(1) //行数

			writers := []io.Writer{
				// errLogHandle, // *os.File
				os.Stdout, //标准输出，最后编译时可以删除
			}
			logger := log.New(io.MultiWriter(writers...), "", log.Ldate|log.Ltime) //|log.Lshortfile
			logger.Println(fp + ":" + strconv.Itoa(ln) + "." + strconv.Itoa(i+1) + "==>" + e.Error())
		}
	}
	return haveErr
}

// CreateUUID 生成id
// 16字节
func CreateUUID() []byte {
	var r []byte
	for {
		r = uuid.Must(uuid.NewV4(), nil).Bytes()
		if !bytes.Contains(r, []byte("`")) {
			return r
		}
	}
}
