package com

import (
	"bytes"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

func Errorlog(errs ...error) bool {
	var flag bool = false
	for _, v := range errs {
		if v != nil {
			// log
			fmt.Println(v)
			flag = true
		}
	}
	return flag
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
