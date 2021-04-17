package stun

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/mapdb"
)

//  无论客户端还是服务器都需要两个IP(IP1和IP2)。同一个VPS绑定两张网卡；这两张网卡的私网IP分别是a、b，公网IP分别是x，y。则在客户端IP1、IP2分别配置为x、y，在服务器IP1、IP2分别配置为a、b。

type STUN struct {
	Iterate   int           // 同数据包重复发送次数，确保UDP可靠，默认5
	MatchTime time.Duration // 匹配时长，默认30m
	TimeOut   time.Duration // 超时时间，默认3s
	ExtPorts  int           // 泛端口范围，默认7
	// 服务器端口(同时也要使用SeverPort+1端口)，默认19986
	//   如果设置，服务器和客户端要设置相同
	SeverPort int

	/* 仅客户端配置 */

	// 服务器地址、IP或域名，仅客户端需要设置
	//  其对应的IP应该是服务器第一网卡的公网IP
	Sever string

	/* 仅服务器配置 */

	// 服务器第一网卡的局域网IP，仅服务器需要设置
	//  此网卡的公网IP与客户端设置的Sever应相同
	LIP1 net.IP
	// 服务器第二网卡的局域网IP，仅服务器需要设置
	LIP2 net.IP
	// 服务器第二网卡的公网网IP，仅服务器需要设置
	WIP2 net.IP

	/* 私有 */

	dbj *mapdb.Db // NAT类型判断的数据库
	dbt *mapdb.Db // NAT穿隧数据库
	s1  int       // 服务器第一端口，与SeverPort相同
	s2  int       // 服务器第二端口，与SeverPort+1相同
	s3  int       // 服务器第三端口，与SeverPort+2相同
}

type R struct {
	// 对方网关地址
	Raddr net.Addr
	// 对方NAT类型
	RNat int
	// 己方NAT类型
	LNat int
}

var err error
var errSever error = errors.New("Server no reply")

func (s *STUN) ClientInit(Sever string) error {
	if s.Iterate == 0 {
		s.Iterate = 5
	}
	if s.MatchTime == 0 {
		s.MatchTime = time.Minute * 30
	}
	if s.TimeOut == 0 {
		s.TimeOut = time.Second * 3
	}
	if s.ExtPorts == 0 {
		s.ExtPorts = 7
	}
	if s.SeverPort == 0 {
		s.s1 = 19986
		s.s2 = 19987
		s.s3 = 19988
	} else {
		s.s1 = s.SeverPort
		s.s2 = s.SeverPort + 1
		s.s3 = s.SeverPort + 2
	}

	if s.Sever, err = domainToIP(Sever); err != nil {
		return err
	}
	return nil
}

func (s *STUN) SeverInit(LIP1, LIP2, WIP2 net.IP) error {
	if s.Iterate == 0 {
		s.Iterate = 5
	}
	if s.MatchTime == 0 {
		s.MatchTime = time.Minute * 30
	}
	if s.TimeOut == 0 {
		s.TimeOut = time.Second * 3
	}
	if s.ExtPorts == 0 {
		s.ExtPorts = 7
	}
	if s.SeverPort == 0 {
		s.s1 = 19986
		s.s2 = 19987
		s.s3 = 19988
	} else {
		s.s1 = s.SeverPort
		s.s2 = s.SeverPort + 1
		s.s3 = s.SeverPort + 2
	}

	s.LIP1 = LIP1
	s.LIP2 = LIP2
	s.WIP2 = WIP2

	s.dbj = new(mapdb.Db)
	s.dbj.Init()
	s.dbt = new(mapdb.Db)
	s.dbt.Init()
	return nil
}

func (s *STUN) RunSever() error {

	var conn1, conn2, conn3, ip2conn *net.UDPConn
	if conn1, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.LIP1, Port: s.s1}); e.Errlog(err) {
		return err
	}
	if conn2, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.LIP1, Port: s.s2}); e.Errlog(err) {
		return err
	}
	if conn3, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.LIP1, Port: s.s3}); e.Errlog(err) {
		return err
	}
	if ip2conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.LIP2, Port: s.s1}); e.Errlog(err) {
		return err
	}

	var da []byte = make([]byte, 256)
	var raddr *net.UDPAddr
	var n int
	var cl sync.RWMutex
	// 第二IP接收到的数据
	go func() {
		for {
			if n, raddr, err = ip2conn.ReadFromUDP(da); e.Errlog(err) {
				continue
			}
			if da[0] == 'J' {
				cl.Lock()
				s.judgeSever(conn1, conn3, ip2conn, da[:n], raddr)
				cl.Unlock()
			}
		}
	}()
	// 第二端口接收到数据
	go func() {
		for {
			if n, raddr, err = conn2.ReadFromUDP(da); e.Errlog(err) {
				continue
			}
			if da[0] == 'J' {
				cl.Lock()
				s.judgeSever(conn1, conn3, ip2conn, da[:n], raddr)
				cl.Unlock()
			}
		}
	}()
	// 第一端口接收到的数据
	for {
		if n, raddr, err = conn1.ReadFromUDP(da); e.Errlog(err) {
			continue
		}

		if da[0] == 'J' {
			cl.Lock()
			s.judgeSever(conn1, conn3, ip2conn, da[:n], raddr)
			e.Errlog(err)
			cl.Unlock()

		} else if da[0] == 'T' {
			if err = s.throughSever(conn1, da[:n], raddr); e.Errlog(err) {
				continue
			}
		}
	}
}

// RunClient id双方要相同
func (s *STUN) RunClient(port int, id [16]byte) (R, error) {
	var lnats []int
	for i := 0; i < 1; i++ {
		var tlnat int
		if tlnat, err = s.judgeCliet(RandPort()); err != nil {
			if strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "other") {
				continue // 端口被占用
			} else {
				return R{}, err
			}
		}
		lnats = append(lnats, tlnat)
	}
	lnat := selectMost(lnats)
	return R{}, nil

	// 尝试穿隧
	raddr, rnat, err := s.throughClient(append([]byte("T"), id[:]...), port, lnat)
	if e.Errlog(err) {
		return R{}, nil
	}

	return R{Raddr: raddr, RNat: rnat, LNat: lnat}, nil
}

func RandPort() int {
	b := new(big.Int).SetInt64(int64(52000))
	i, err := rand.Int(rand.Reader, b)
	r := int(i.Int64()) + 100
	if e.Errlog(err) {
		return 52942
	}
	return r
}

func selectMost(l []int) int {
	var m map[int]int = make(map[int]int)
	for _, v := range l {
		m[v] = m[v] + 1
	}
	var c, r int = 0, 0
	for k, v := range m {
		if v > c {
			c = v
			r = k
		}
	}
	return r
}

func domainToIP(sever string) (string, error) {
	if r := net.ParseIP(sever); r == nil { //可能是域名
		var ips []net.IP
		if ips, err = net.LookupIP(sever); err != nil {
			return "", err
		}
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String(), nil
			}
		}
	}
	return sever, nil
}
