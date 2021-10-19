package main

import (
	"fmt"
	"net"
)

func main() {
	_, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: 19986}, &net.UDPAddr{IP: net.ParseIP("4.4.4.4"), Port: 19986})
	fmt.Println(err)

	if false {
		// windows.Accept(nil)
	}
}
