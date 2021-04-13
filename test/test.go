package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/lysShub/e"
	"github.com/lysShub/stun"
)

func main() {
	if runtime.GOOS == "android" {
		fh, err := os.OpenFile(`/mnt/sdcard/a/err.log`, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			e.L(os.Stdout, os.Stderr)
		} else {
			e.L(os.Stdout, os.Stderr, fh)
		}
	}

	var s = new(stun.STUN)
	s.Sever = "114.116.254.26"
	s.SeverPort = 19986

	/* sever */
	fmt.Println("开始")
	if err := s.Init(false); e.Errlog(err) {
		return
	}
	s.RunSever()

	/* client */
	fmt.Println("开始")
	if err := s.Init(true); e.Errlog(err) {
		return
	}
	fmt.Println(s.RunClient(8080, [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 'a', 'b', 'c', 'd', 'e', 'f'}))

}
