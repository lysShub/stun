package main

import (
	"fmt"
	"time"

	"github.com/lysShub/stun"
)

func main() {
	var s = new(stun.STUN)
	s.Iterate = 5
	s.MatchTime = time.Minute * 20
	s.TimeOut = time.Second * 2

	fmt.Println("开始")

	// fmt.Println(s.Sever(19986, 19987))

	fmt.Println(s.DiscoverClient(9986, 9987, 19986, 19987, "114.116.254.26"))

	// var suuid []byte = []byte("T0123456789abcdef")
	// fmt.Println(s.ThroughClient(suuid))
}
