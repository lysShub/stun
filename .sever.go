package stun

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/mapdb"
	"github.com/lysShub/stun/internal/com"
)

// InitSever 运行端口, 本地第二IP及对应的公网IP
//  第一IP的内网IP可自动获取
func InitSever(port int, lip2, wip2 net.IP) (*sever, error) {
	var s = new(sever)
	s.ResendTimes = 5
	s.MatchTime = time.Second * 30
	s.TimeOut = time.Second * 3
	s.ExtPorts = 5

	if port <= 0 || port >= 65535 {
		port = 19986
	}
	s.s1 = port
	s.s2 = port + 1

	if s.lip2 = lip2; !com.IsLanIP(lip2) {
		return nil, errors.New("invalid parameter 'lip2'")
	}
	if s.wip2 = wip2; com.IsLanIP(wip2) {
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

func (s *sever) RunSever() error {

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

	// 第二IP接收到的数据
	go func() {
		for {
			if n, raddr, err = s.conn3.ReadFromUDP(da); com.Errlog(err) {
				continue
			}
			if da[0] == 'J' {
				cl.Lock()
				s.discoverSever(da[:n], raddr)
				cl.Unlock()
			}
		}
	}()
	// 第二端口接收到数据
	go func() {
		for {
			if n, raddr, err = s.conn2.ReadFromUDP(da); com.Errlog(err) {
				continue
			}
			if da[0] == 'J' {
				cl.Lock()
				s.discoverSever(da[:n], raddr)
				cl.Unlock()
			}
		}
	}()
	// 第一端口接收到的数据
	for {
		if n, raddr, err = s.conn1.ReadFromUDP(da); com.Errlog(err) {
			continue
		}

		if da[0] == 'J' {
			cl.Lock()
			s.discoverSever(da[:n], raddr)
			com.Errlog(err)
			cl.Unlock()

		} else if da[0] == 'T' {
			// if err = s.throughSever(conn1, da[:n], raddr); com.Errlog(err) {
			// 	continue
			// }
		}
	}
}
