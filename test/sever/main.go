package main

import (
	"fmt"
	"net"

	"github.com/lysShub/stun/internal/sever"
)

func main() {

	/* sever */
	fmt.Println("开始l")

	sever.Run(net.ParseIP("192.168.0.50"), net.ParseIP("119.3.166.124"))

}
