package main

import (
	"fmt"

	"github.com/lysShub/stun"
)

func main() {
	var s = new(stun.STUN)

	fmt.Println("开始")

	fmt.Println(s.Sever(19986, 19987))
	//
	// fmt.Println(s.DiscoverClient(19986, 19987, 19986, 19987, "114.116.254.26"))

	// var suuid []byte = []byte("T0123456789abcdef")
	// fmt.Println(s.ThroughClient(suuid))
}
