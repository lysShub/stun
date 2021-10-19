package sever

import (
	"errors"
	"net"
	"sync"

	"stun/config"
	"stun/internal/com"

	"github.com/lysShub/mapdb"
)

// sever Sever Conn
type sever struct {
	s1, s2     int          // 端口
	lip1, lip2 net.IP       // 内网第一IP, 内网第二IP
	wip2       net.IP       // 内网第二IP(第二网卡)对应的公网IP
	conn1      *net.UDPConn // 第一IP: 第一端口
	conn2      *net.UDPConn // 第一IP: 第二端口
	conn3      *net.UDPConn // 第二IP: 第一端口

	dbj *mapdb.Db // NAT类型判断的数据库
	dbt *mapdb.Db // NAT穿隧数据库
}

// Run
// 	@第二IP的内网地址和外网地址; 第一IP为默认上网IP
func Run(lip2, wip2 net.IP) error {
	var s *sever
	if s, err = initSever(lip2, wip2); err != nil {
		return err
	}

	if s.conn1, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.lip1, Port: s.s1}); err != nil {
		panic(err)
	}
	if s.conn2, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.lip1, Port: s.s2}); err != nil {
		panic(err)
	}
	if s.conn3, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.lip2, Port: s.s1}); err != nil {
		panic(err)
	}

	var da []byte = make([]byte, 256)
	var raddr *net.UDPAddr
	var n int
	var cl sync.Mutex

	// 第二IP接收到的数据s
	go func() {
		for {
			if n, raddr, err = s.conn3.ReadFromUDP(da); com.Errlog(err) {
				continue
			}
			if da[0] == 'J' {
				cl.Lock()
				s.findSever(da[:n], raddr)
				cl.Unlock()
			}
		}
	}()

	// 第一IP第二端口接收到数据
	go func() {
		for {
			if n, raddr, err = s.conn2.ReadFromUDP(da); com.Errlog(err) {
				continue
			}
			if da[0] == 'J' {
				cl.Lock()
				s.findSever(da[:n], raddr)
				cl.Unlock()
			}
		}
	}()

	// 第一IP第一端口接收到的数据
	for {
		if n, raddr, err = s.conn1.ReadFromUDP(da); com.Errlog(err) {
			continue
		}

		if da[0] == 'J' {
			cl.Lock()
			s.findSever(da[:n], raddr)
			com.Errlog(err)
			cl.Unlock()

		} else if da[0] == 'T' {
			// if err = s.throughSever(conn1, da[:n], raddr); com.Errlog(err) {
			// 	continue
			// }
		}
	}
}

func initSever(lip2, wip2 net.IP) (*sever, error) {
	var s = new(sever)

	s.s1 = 19986
	s.s2 = 19987

	if s.lip2 = lip2.To16(); !s.lip2.IsPrivate() {
		return nil, errors.New("invalid parameter 'lip2'")
	}
	if s.wip2 = wip2.To16(); s.wip2.IsPrivate() {
		return nil, errors.New("invalid parameter 'wip2'")
	}
	if s.lip1, err = com.GetLocalIP(); err != nil {
		return nil, err
	}

	s.dbj = new(mapdb.Db)
	s.dbj.Init()
	s.dbt = new(mapdb.Db)
	s.dbt.Init()

	return s, nil
}

func (s *sever) Send(conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {
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
