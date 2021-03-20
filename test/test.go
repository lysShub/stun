package test

import (
	"fmt"

	"github.com/lysShub/stun/internal/discover"
)

func Test() {
	var d = new(discover.Discover)
	d.FirstPort = 19986
	d.SecondPort = 19987

	fmt.Println(d.Sever())

}
