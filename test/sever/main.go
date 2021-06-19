package main

import (
	"fmt"
	"net"

	"github.com/lysShub/e"
	"github.com/lysShub/stun"
)

func main() {

	var s = new(stun.STUN)

	/* sever */
	fmt.Println("开始l")
	if err := s.SeverInit(net.ParseIP("192.168.0.40"), net.ParseIP("192.168.0.50"), net.ParseIP("119.3.166.124")); e.Errlog(err) {
		return
	}
	s.RunSever()

}
