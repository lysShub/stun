package ioer

/* 为Client提供连接平滑迁移的能力
1. 平滑迁移本地地址
2. 平滑迁移对端地址
3. 两个地址同时迁移
*/

import (
	"net"
)

// 缓冲长度
const buflen int = 2000

type Conns struct {
	ch chan *[buflen]byte

	conn *conn
}

type conn struct {
	pbuf [buflen]byte // 前凉两个字节表示payload长度
	conn net.UDPConn

	next *conn // 闭合链表
}

func Dial(laddr, raddr net.UDPAddr) (*Conns, error) {

	return nil, nil
}

func (c *Conns) Add(laddr, raddr net.UDPAddr) error {

	return nil
}

func (c *Conns) Delete(laddr, raddr net.UDPAddr) error {
	return nil
}

func (c *Conns) Read(b []byte) (int, error) {

	return 0, nil
}

func (c *Conns) Write(b []byte) (int, error) {
	return 0, nil
}
