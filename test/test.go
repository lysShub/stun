package main

import (
	"fmt"

	"github.com/lysShub/stun"
)

func main() {
	var s = new(stun.STUN)
	s.Port = 19986
	s.SeverAddr = "114.116.254.26"
	s.SecondPort = 19987

	fmt.Println("开始")

	fmt.Println(s.Sever())
	//
	// fmt.Println(s.DiscoverClient())
}
