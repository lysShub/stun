package main

import (
	"fmt"
	"net"
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

	/* sever */
	fmt.Println("开始l")
	if err := s.SeverInit(net.ParseIP("192.168.0.40"), net.ParseIP("192.168.0.50"), net.ParseIP("119.3.166.124")); e.Errlog(err) {
		return
	}
	s.RunSever()

	/* client */
	fmt.Println("开始l")
	if err := s.ClientInit("114.116.254.26"); e.Errlog(err) { //119.3.166.124
		return
	}
	fmt.Println(s.RunClient(15683, [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 'a', 'b', 'c', 'd', 'e', 'f'}))

}
