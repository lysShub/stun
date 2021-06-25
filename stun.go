package stun

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/mapdb"
	"github.com/lysShub/stun/internal/com"
)

//  无论客户端还是服务器都需要两个IP(IP1和IP2)。同一个VPS绑定两张网卡；这两张网卡的私网IP分别是a、b，公网IP分别是x，y。则在客户端IP1、IP2分别配置为x、y，在服务器IP1、IP2分别配置为a、b。

type STUN struct {
	Iterate   int           // 同数据包重复发送次数，确保UDP可靠，默认5
	MatchTime time.Duration // 匹配时长
	TimeOut   time.Duration // 超时时间
	ExtPorts  int           // 泛端口范围，默认7

	Port int // 端口，使用多个端口则依次递增
}

var err error
var errSever error = errors.New("Server no reply")

type sconn struct {
	STUN

	s1, s2     int          // 端口
	lip1, lip2 net.IP       // 内网第一IP, 内网第二IP
	wip2       net.IP       // 内网第二IP(第二网卡)对应的公网IP
	conn1      *net.UDPConn // 第一IP: 第一端口
	conn2      *net.UDPConn // 第一IP: 第二端口
	conn3      *net.UDPConn // 第二IP: 第一端口

	dbj *mapdb.Db // NAT类型判断的数据库
	dbt *mapdb.Db // NAT穿隧数据库
}

// InitSever 运行端口, 本地第二IP及对应的公网IP
func InitSever(port int, lip2, wip2 net.IP) (*sconn, error) {
	var s = new(sconn)
	s.Iterate = 5
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

func (s *sconn) RunSever() error {

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
			if n, raddr, err = s.conn3.ReadFromUDP(da); e.Errlog(err) {
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
			if n, raddr, err = s.conn2.ReadFromUDP(da); e.Errlog(err) {
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
		if n, raddr, err = s.conn1.ReadFromUDP(da); e.Errlog(err) {
			continue
		}

		if da[0] == 'J' {
			cl.Lock()
			s.discoverSever(da[:n], raddr)
			e.Errlog(err)
			cl.Unlock()

		} else if da[0] == 'T' {
			// if err = s.throughSever(conn1, da[:n], raddr); e.Errlog(err) {
			// 	continue
			// }
		}
	}
}

type cconn struct {
	STUN
	sever  net.IP // 服务器
	c1, c2 int    // 本地使用端口
}

func InitClient(port int, sever net.IP) (*cconn, error) {
	var s = new(cconn)
	s.Iterate = 5
	s.MatchTime = time.Second * 30
	s.TimeOut = time.Second * 3
	s.ExtPorts = 5
	if port <= 0 || port >= 65535 {
		port = 19986
	}
	s.c1 = port
	s.c2 = port + 1

	if s.sever = sever; com.IsLanIP(sever) {
		return nil, errors.New("invalid parameter 'sever'")
	}
	return s, nil
}

// RunClient
func (s *cconn) RunClient(port int, id [16]byte) error {
	var natType int
	if natType, err = s.discoverCliet(); err != nil {
		return err
	}
	fmt.Println("natType", natType)
	return nil

	// 尝试穿隧
	raddr, rnat, err := s.throughClient(append([]byte("T"), id[:]...), port, natType)
	if e.Errlog(err) {
		return nil
	}
	fmt.Println(raddr, rnat)
	// return R{Raddr: raddr, RNat: rnat, LNat: lnat}, nil
	return nil
}
