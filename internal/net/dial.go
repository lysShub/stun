package net

import "net"

type sysDialer struct {
	net.Dialer
	network, address string
}
