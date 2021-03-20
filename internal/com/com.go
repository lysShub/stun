package com

import (
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
	return uuid.Must(uuid.NewV4(), nil).Bytes()
}
