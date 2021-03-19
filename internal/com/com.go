package com

import (
	uuid "github.com/satori/go.uuid"
)

func Errorlog(errs ...error) error {
	for _, v := range errs {
		if v != nil {
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
