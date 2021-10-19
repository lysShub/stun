package main

import (
	"fmt"
	"net"

<<<<<<< HEAD
	"stun/internal/action/client"
=======
	"stun/action/client"
>>>>>>> 166bdcbc89bce448fa31b3ef9c364a067cd5758e
)

func main() {

	/* client */
	fmt.Println("开始l")

	if _, err := client.Run(19986, net.ParseIP("114.116.254.26")); err != nil {
		fmt.Println(err)
	}

}
