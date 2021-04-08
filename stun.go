package stun

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/lysShub/mapdb"
	"github.com/lysShub/stun/internal/com"
)

type STUN struct {
	// 第二IP
	SIP net.IP

	/* 私有 */
	dbd          *mapdb.Db    // NAT类型判断的数据库
	dbt          *mapdb.Db    // NAT穿隧数据库
	secondIPConn *net.UDPConn // 第二IP的UDPconn
}

var err error
var errSever error = errors.New("Server no reply")

// 第一第二端口
func (s *STUN) Sever(s1, s2 int) error {
	laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(s1))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	var da []byte = make([]byte, 256)
	for {
		n, raddr, err := conn.ReadFromUDP(da)
		if com.Errorlog(err) {
			continue
		}
		if da[0] == 'J' { //NAT判断
			fmt.Println("接收到数据J")
			if err = s.DiscoverSever(conn, 19986, 19987, da[:n], raddr); com.Errorlog(err) {
				continue
			}

		} else if da[0] == 'T' {
			fmt.Println("接收到数据T")
			if err = s.throughSever(conn, da[:n], raddr); com.Errorlog(err) {
				continue
			}
		}
	}
}
