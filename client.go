package stun

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/lysShub/stun/internal/com"
)

func InitClient(port int, sever net.IP) (*cconn, error) {
	var s = new(cconn)
	s.Iterate = 5
	s.MatchTime = time.Second * 30
	s.TimeOut = time.Second * 3
	s.ExtPorts = 5
	if port <= 0 || port >= 65535 {
		port = 19986
	}
	s.cp1 = port
	s.cp2 = port + 1

	if s.sever = sever; com.IsLanIP(sever) {
		return nil, errors.New("invalid parameter 'sever'")
	}
	return s, nil
}

// RunClient
//  确保id随机
func (s *cconn) RunClient(port int, id [16]byte) error {
	var natType int
	if natType, err = s.DiscoverCliet(); err != nil {
		return err
	}
	fmt.Println("natType", natType)
	return nil

	// 尝试穿隧
	// raddr, rnat, err := s.throughClient(append([]byte("T"), id[:]...), port, natType)
	// if e.Errlog(err) {
	// 	return nil
	// }
	// fmt.Println(raddr, rnat)
	// return R{Raddr: raddr, RNat: rnat, LNat: lnat}, nil
	return nil
}
