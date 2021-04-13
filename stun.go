package stun

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/mapdb"
)

type STUN struct {
	// 服务器地址，IP或域名，仅client
	Sever string
	// 服务器端口
	SeverPort int
	// 同数据包重复发送次数，确保UDP可靠，默认5
	Iterate int
	// 匹配时长，默认30m
	MatchTime time.Duration
	// 超时时间，默认3s
	TimeOut time.Duration
	// 泛端口范围，默认7
	ExtPorts int
	// 第二IP，可选，仅sever
	SIP net.IP

	/* 私有 */
	dbd          *mapdb.Db    // NAT类型判断的数据库
	dbt          *mapdb.Db    // NAT穿隧数据库
	secondIPConn *net.UDPConn // 第二IP的UDPconn
	s1           int          // 服务器第一端口，与SeverPort相同
	s2           int          // 服务器第二端口
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

// Init ic==true meaning initialize for client
func (s *STUN) Init(ic bool) error {

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
		return errors.New("Please set field SeverPort. ")
	} else {
		s.s1 = s.SeverPort
		s.s2 = s.SeverPort + 1
	}
	if ic { //client
		if s.Sever == "" {
			return errors.New("Please set field Sever. ")
		} else {
			if s.Sever, err = domainToIP(s.Sever); e.Errlog(err) { //可能是域名
				return errors.New("invlid field Sever.")
			}
		}
	} else { //sever
		s.dbd = new(mapdb.Db)
		s.dbd.Init()
		s.dbt = new(mapdb.Db)
		s.dbt.Init()

		if s.SIP != nil {
			if s.secondIPConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: s.SIP, Port: s.SeverPort}); e.Errlog(err) {
				return errors.New("invlid field SIP. ")
			}
		}
	}
	return nil
}

func (s *STUN) RunSever() error {

	var conn *net.UDPConn
	if conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: s.s1}); e.Errlog(err) {
		return err
	}
	var conn2 *net.UDPConn
	if conn2, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: s.s2}); e.Errlog(err) {
		return err
	}

	var da []byte = make([]byte, 256)
	var raddr *net.UDPAddr
	var n int
	for {

		if n, raddr, err = conn.ReadFromUDP(da); e.Errlog(err) {
			continue
		}

		if da[0] == 'J' { //NAT判断
			if err = s.discoverSever(conn, conn2, da[:n], raddr); e.Errlog(err) {
				continue
			}

		} else if da[0] == 'T' {
			if err = s.throughSever(conn, da[:n], raddr); e.Errlog(err) {
				continue
			}
		}
	}
}

// RunClient id双方要相同
func (s *STUN) RunClient(port int, id [16]byte) (R, error) {
	var lnats []int
	for i := 0; i < 3; i++ {
		var tlnat int
		if tlnat, err = s.discoverClient(RandPort()); e.Errlog(err) {
			if strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "other") {
				continue // 端口被占用
			} else {
				return R{}, err
			}
		}
		lnats = append(lnats, tlnat)
	}
	lnat := selectMost(lnats)
	fmt.Println("NAT类型:", lnat)

	// 尝试穿隧
	raddr, rnat, err := s.throughClient(append([]byte("T"), id[:]...), port, lnat)
	if e.Errlog(err) {
		fmt.Println("对方nat", rnat)
		return R{}, nil
	}

	return R{Raddr: raddr, RNat: rnat, LNat: lnat}, nil
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
