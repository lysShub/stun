package main

import (
	"fmt"
	"net"

	"github.com/lysShub/stun/internal/action/client"
)

func main() {

	/* client */
	fmt.Println("开始l")

	if _, err := client.Run(19986, net.ParseIP("114.116.254.26")); err != nil {
		fmt.Println(err)
	}

}
