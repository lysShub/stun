package main

import (
	"fmt"
	"net"

<<<<<<< HEAD
	"stun/internal/action/sever"
=======
	"stun/action/sever"
>>>>>>> 166bdcbc89bce448fa31b3ef9c364a067cd5758e
)

func main() {

	/* sever */
	fmt.Println("开始l")

	sever.Run(net.ParseIP("192.168.0.50"), net.ParseIP("119.3.166.124"))

}
