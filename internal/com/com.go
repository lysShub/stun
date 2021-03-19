package com

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
)

func Errorlog(errs ...error) error {
	for _, v := range errs {
		if v != nil {
			fmt.Println(v)
			return v
		}
	}
	return nil
}

// CreateUUID 生成id
// 16字节
func CreateUUID() []byte {
	return uuid.Must(uuid.NewV4(), nil).Bytes()
}
