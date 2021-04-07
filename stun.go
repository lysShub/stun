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
	// 客户端或服务器第一端口(主端口)
	Port uint16
	// 客户端或服务器第二端口(仅在NAT发现中被使用)
	SecondPort uint16
	// 服务器地址，IP或域名(客户端中必须设置)
	SeverAddr string
	conn      *net.UDPConn // 客户端与服务器的主端口的‘链接’

	/* NAT类型发现 */
	// 第二网卡的内网IP，可选。如果不设置则不会区分IP与端口限制形NAT
	SecondNetCardIP net.IP
	conn2           *net.UDPConn
	secondIPConn    *net.UDPConn // 第二IP的conn
	dbd             *mapdb.Db    // NAT类型判断的数据库

	/* NAT穿隧 */
	dbt *mapdb.Db // NAT穿隧数据库
}

func (s *STUN) Init(isClient bool) error {

	if s.Port != 0 {
		if isClient { // 客户端
			l, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port)))
			if err != nil {
				return err
			}
			r, err := net.ResolveUDPAddr("udp", s.SeverAddr+":"+strconv.Itoa(int(s.Port)))
			if err != nil {
				return err
			}
			s.conn, err = net.DialUDP("udp", l, r)
			if err != nil {
				s.conn = nil
				return err
			}
		} else { // 服务器
			l, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port)))
			if err != nil {
				return err
			}
			s.conn, err = net.ListenUDP("udp", l)
			if err != nil {
				s.conn = nil
				return nil
			}
		}
	}
	if s.SecondPort != 0 {
		if isClient { //客户端
			l, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.SecondPort)))
			if err != nil {
				return err
			}
			r, err := net.ResolveUDPAddr("udp", s.SeverAddr+":"+strconv.Itoa(int(s.Port)))
			if err != nil {
				return err
			}
			s.conn2, err = net.DialUDP("udp", l, r)
			if err != nil {
				s.conn2 = nil
				return err
			}
		} else { // 服务器
			l2, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.SecondPort)))
			if err != nil {
				return err
			}
			s.conn2, err = net.ListenUDP("udp", l2)
			if err != nil {
				s.conn2 = nil
				return nil
			}
		}
	}
	if s.SecondNetCardIP != nil { // 第二IP
		l2, err := net.ResolveUDPAddr("udp", s.SecondNetCardIP.String()+":"+strconv.Itoa(int(s.Port)))
		if err != nil {
			return nil
		}
		s.secondIPConn, err = net.ListenUDP("udp", l2)
		if err != nil {
			s.secondIPConn = nil
			return err
		}
	}

	if !isClient {
		s.dbd = new(mapdb.Db)
		s.dbd.Init()

		s.dbt = new(mapdb.Db)
		s.dbt.Init()
	}

	return nil
}

var err error
var errSever error = errors.New("Server no reply")

func (s *STUN) Sever() error {
	if err = s.Init(false); err != nil {
		return err
	}

	var da []byte = make([]byte, 256)
	for {
		n, raddr, err := s.conn.ReadFromUDP(da)
		if com.Errorlog(err) {
			continue
		}
		if da[0] == 'J' { //NAT判断
			fmt.Println("接收到数据J")
			if err = s.discoverSever(da[:n], raddr); com.Errorlog(err) {
				continue
			}

		} else if da[0] == 'T' {
			fmt.Println("接收到数据T")
			if err = s.throughSever(da[:n], raddr); com.Errorlog(err) {
				continue
			}
		}
	}
}
