package main

import (
	"net"
)

func main() {

	net.DialUDP("udp", nil, nil)
	a := net.ParseIP("")
	a.String()
}
