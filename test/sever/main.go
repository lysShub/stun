package main

import (
	"fmt"
	"net"

	"github.com/lysShub/e"
	"github.com/lysShub/stun"
)

func main() {

	/* sever */
	fmt.Println("开始l")
	if sconn, err := stun.InitSever(19986, net.ParseIP("192.168.0.50"), net.ParseIP("119.3.166.124")); e.Errlog(err) {
		return
	} else {
		sconn.RunSever()
	}

}
