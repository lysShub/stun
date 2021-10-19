//go:build windows
// +build windows

package ioer

import "net"

func dial() {
	net.DialUDP("udp", nil, nil)
}
