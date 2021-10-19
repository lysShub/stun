package main

import (
	"net"
)

func main() {
	_, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: 19986}, &net.UDPAddr{IP: net.ParseIP("4.4.4.4"), Port: 19986})
	fmt.Println(err)

<<<<<<< HEAD
	net.DialUDP("udp", nil, nil)
	a := net.ParseIP("")
	a.String()
=======
	if false {
		// windows.Accept(nil)
	}
>>>>>>> 166bdcbc89bce448fa31b3ef9c364a067cd5758e
}
