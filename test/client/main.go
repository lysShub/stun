package main

import (
	"fmt"
	"net"

	"github.com/lysShub/e"
	"github.com/lysShub/stun"
)

func main() {

	/* client */
	fmt.Println("开始l")
	if cconn, err := stun.InitClient(19986, net.ParseIP("114.116.254.26")); e.Errlog(err) {
		return
	} else {
		fmt.Println(cconn.RunClient(19986, [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 'a', 'b', 'c', 'd', 'e', 'f'}))
	}

}
