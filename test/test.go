package main

import (
	"fmt"

	"github.com/lysShub/e"
	"github.com/lysShub/stun"
)

func main() {
	var s = new(stun.STUN)
	s.Sever = "114.116.254.26"
	s.SeverPort = 19986

	/* client */
	fmt.Println("开始")
	if err := s.Init(true); e.Errlog(err) {
		return
	}
	s.RunClient(8089)

	/* sever */
	// fmt.Println("开始")
	// if err := s.Init(false); e.Errlog(err) {
	// 	return
	// }
	// s.RunSever()
}
