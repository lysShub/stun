package client

import (
	"errors"
	"fmt"
	"net"

	"stun/config"
	"stun/internal/com"
)

// client Client Conn
type client struct {
	sever net.IP // 服务器IP

	// cp1, cp2 int // 本地(客户端)使用端口, client port

}

var err error

// Run
// 	@port 准备穿透的端口, 可能最终实现穿透的是其他端口
// 	@sever 服务器地址
func Run(port int, sever net.IP) (int, error) {

	var c, err = initClient(sever)
	if err != nil {
		return -1, err
	}

	var natType int
	if natType, err = c.findClient(com.RandPort()); err != nil {
		return -1, err
	}
	fmt.Println("natType", natType)
	return 0, nil

	// 尝试穿隧
	// raddr, rnat, err := s.throughClient(append([]byte("T"), id[:]...), port, natType)
	// if com.Errlog(err) {
	// 	return nil
	// }
	// fmt.Println(raddr, rnat)
	// return R{Raddr: raddr, RNat: rnat, LNat: lnat}, nil
	// return nil

}

func initClient(sever net.IP) (*client, error) {
	var s = new(client)

	if s.sever = sever; com.IsLanIP(sever) {
		return nil, errors.New("invalid parameter 'sever'")
	}
	return s, nil
}

func (c *client) Send(conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {
	for i := 0; i < config.ResendTimes; i++ {
		if raddr != nil {
			if _, err := conn.WriteToUDP(da, raddr); err != nil {
				return err
			}
		} else {
			if _, err := conn.Write(da); err != nil {
				return err
			}
		}
	}
	return nil
}
