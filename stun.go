package stun

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/mapdb"
)

type STUN struct {
	// 服务器第一端口，默认19986
	S1 int
	// 服务器第二端口，仅sever，默认19987
	S2 int
	// 客户端第一端口，默认19986
	C1 int
	// 客户端第二端口，默认19987
	C2 int
	// 第二IP，仅sever，可选
	SIP net.IP
	// 同数据包重复发送次数，由于UDP可靠，默认5
	Iterate int
	// 匹配时长，默认1d
	MatchTime time.Duration
	// 超时时间，默认3s
	TimeOut time.Duration
	// 泛端口范围
	ExtPorts int

	/* 私有 */
	dbd          *mapdb.Db    // NAT类型判断的数据库
	dbt          *mapdb.Db    // NAT穿隧数据库
	secondIPConn *net.UDPConn // 第二IP的UDPconn
}

var err error
var errSever error = errors.New("Server no reply")

// 第一第二端口
func (s *STUN) Sever(s1, s2 int) error {

	s.dbd = new(mapdb.Db)
	s.dbd.Init()
	s.dbt = new(mapdb.Db)
	s.dbt.Init()

	if s.SIP != nil {

		if s.secondIPConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.SIP, Port: s1}); e.Errlog(err) {
			s.secondIPConn = nil
			return err
		}
	}

	var conn *net.UDPConn
	if conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: s1}); e.Errlog(err) {
		return err
	}

	var da []byte = make([]byte, 256)
	for {
		n, raddr, err := conn.ReadFromUDP(da)
		e.Errlog(err)

		if da[0] == 'J' { //NAT判断
			fmt.Println("接收到数据J")
			if err = s.DiscoverSever(conn, 19986, 19987, da[:n], raddr); e.Errlog(err) {
				continue
			}

		} else if da[0] == 'T' {
			fmt.Println("接收到数据T")
			if err = s.throughSever(conn, da[:n], raddr); e.Errlog(err) {
				continue
			}
		}
	}
}
