package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lysShub/kvdb"
	"github.com/lysShub/stun/internal/com"
)

type STUN struct {
	// 第一端口，客户端和服务器设置要相同
	Port uint16
	conn *net.UDPConn // 第一端口
	// NAT判断和穿透的服务器地址，IP或域名
	SeverAddr string
	db        *kvdb.KVDB

	/* NAT类型判断*/
	// NAT判断第二端口，客户端和服务器设置要相同
	SecondPort uint16
	conn2      *net.UDPConn // 第二端口
	// 第二网卡的局域网IP，可选。如果不设置则不会区分IP与端口限制形NAT
	SecondNetCardIP net.IP
	secondIPConn    *net.UDPConn // 第二IP的第一端口的conn
	dbDiscover      string       // NAT类型判断的数据表名

	/* NAT穿隧 */
	dbThrough string // NAT穿透数据表名
}

func (s *STUN) Init(isClient bool) error {
	s.dbDiscover = "discover"
	s.dbThrough = "through"
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
		} else {
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
		} else {
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
	if s.SecondNetCardIP != nil {
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
	if s.db == nil {
		var db = new(kvdb.KVDB)
		db.Type = 0
		db.RAMMode = true
		if err = db.Init(); err != nil {
			return err
		}
		s.db = db
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
			fmt.Println("接收到数据")
			if err = s.discoverSever(da[:n], raddr); com.Errorlog(err) {
				continue
			}

		} else if da[0] == '?' {

		}
	}
}

// DiscoverSever NAT type discover
func (s *STUN) discoverSever(da []byte, raddr *net.UDPAddr) error {
	var t int = 4
	var step uint16 = 0
	var juuid, rIP, rPort []byte
	var mS = func(raddr *net.UDPAddr) error {
		for i := 0; i < 5; i++ {
			if _, err = s.conn.WriteToUDP(da[:18], raddr); err != nil {
				return err
			}
		}
		return nil
	}

	if len(da) != 18 {
		return nil
	}
	step = uint16(da[17])
	juuid = da[:17]

	if step == 1 {
		var D map[string][]byte = make(map[string][]byte)
		D["step"] = []byte{1}
		D["rIP"] = []byte(raddr.IP.String())
		D["rPort"] = []byte(strconv.Itoa(raddr.Port))
		if err = s.db.SetTableRow(s.dbDiscover, string(juuid), D); err != nil {
			return err
		}

		da[17] = 2
		if err = mS(raddr); err != nil {
			return err
		}
	} else {

		if rPort = s.db.ReadTableValue(s.dbDiscover, string(juuid), "rPort"); rPort == nil {
			return nil
		}
		if rIP = s.db.ReadTableValue(s.dbDiscover, string(juuid), "rIP"); rIP == nil {
			return nil
		}

		if step == 3 { //3

			var raddr1 *net.UDPAddr
			if raddr1, err = net.ResolveUDPAddr("udp", string(rIP)+":"+string(rPort)); err != nil {
				return err
			}

			if strconv.Itoa(raddr.Port) == string(rPort) { //两次请求端口相同，需进一步判断 回复4和5

				da[17] = 4 //4
				if err = mS(raddr1); err != nil {
					return err
				}

				da[17] = 5 //5
				if err = mS(raddr1); err != nil {
					return err
				}

			} else {
				if raddr.Port == int(s.Port) && string(rPort) == strconv.Itoa(int(s.SecondPort)) { // 两次端口与预定义端口相对，公网IP 9

					if err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{9}); err != nil {
						return err
					}
					da[17] = 9
					if err = mS(raddr1); err != nil {
						return err
					}

				} else {
					if err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{0xd}); err != nil {
						return err
					}
					da[17] = 0xd
					if err = mS(raddr1); err != nil {
						return err
					}
				}
			}

		} else if step == 6 {

			if s.secondIPConn != nil {
				da[17] = 7
				for i := 0; i < t; i++ {
					if _, err = s.secondIPConn.WriteToUDP(da[:18], raddr); err != nil {
						return err
					}
				}
				da[17] = 8
				if err = mS(raddr); err != nil {
					return err
				}

			} else { // 不区分，没有回复
				if err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{6}); err != nil {
					return err
				}
			}

		} else if step == 0xa || step == 0xb || step == 0xc { //a b c
			if err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{uint8(step)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// DiscoverClient NAT type discover
func (s *STUN) DiscoverClient() (int16, error) {
	// 返回代码:
	//  0 error
	//  6 Full Cone or Restricted Cone
	//  9 No NAT(Public IP)
	//  a Full Cone
	//  b Restricted Cone
	//  c Port Restricted Cone
	//  d Symmetric Cone
	//  e Sever no response
	//  f Exceptions

	// 临时
	if err = s.Init(true); err != nil {
		return 0, err
	}

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)
	da = append(da, 1)
	var R = func(shouleCode ...uint8) error { //读取函数，收到对应的数据包返回nil
		for { // 由于Sever对相同的包会回复多次，所以对读取到的不是期望的包应该丢弃
			err = s.conn.SetReadDeadline(time.Now().Add(time.Second * 1))
			if err != nil {
				return err
			}
			_, err = s.conn.Read(da)
			if err != nil {
				return err
			} else if bytes.Equal(juuid, da[:17]) {
				var flag bool
				for _, v := range shouleCode {
					if v == da[17] {
						flag = true
					}
				}
				if flag {
					return nil
				}
			}
		}
	}
	/* start */

	// 发 1
	_, err = s.conn.Write(da)
	if err != nil {
		return 0, err
	}

	// 收 2
	err = R(2)
	if err != nil { //timeout sever offline
		return 0xe, errSever
	}

	// 第二端口发 3
	da[17] = 3
	_, err = s.conn2.Write(da[:18])
	if err != nil {
		return 0, err
	}

	// 收  9,d,4,5
	err = R(9, 0xd, 4, 5)
	if err != nil {
		return 0, err
	} else if da[17] == 9 { //公网IP
		return 9, nil

	} else if da[17] == 0xd { //对称NAT
		return 0xd, nil

	} else if da[17] == 4 || da[17] == 5 {
		if da[17] == 5 {
			err = R(4)
			if err != nil {
				if strings.Contains(err.Error(), "time") { //timeout 不区分

					da[17] = 0xc
					s.conn.Write(da[:18])
					return 0xc, nil
				}
				return 0, err
			}
		}
		// 至此，起码4已经收到；为完全锥形或IP限制锥形，接下来可能有进一步判断

		// 收 第二IP的包Juuid:7 或 Juuid:8 或 超时(没有区分)
		err = R(7, 8)
		if err != nil {
			if strings.Contains(err.Error(), "time") { //超时 不区分
				return 6, nil
			}
			return 0, err

		} else if da[17] == 8 || da[17] == 7 {
			if da[17] == 8 { //收到8，尝试接收7

				err = R(7)
				if err != nil {
					if strings.Contains(err.Error(), "time") { //超时
						da[17] = 0xb
						s.conn.Write(da[:38])
						return 0xb, nil
					}
					return 0, err
				}
			}

			// 至此，已经接收到7 完全锥形NAT
			da[17] = 0xa
			s.conn.Write(da[:38])

			return 0xa, nil
		}
	}

	return 0xf, nil // 异常
}

// throughSever
func (s *STUN) throughSever(da []byte, raddr *net.UDPAddr) error {

	if len(da) != 18 {
		fmt.Println("长度不为18")
		return nil
	}
	if da[17] == 1 {
		tuuid := da[:17]
		if s.db.ReadTableRowExist(s.dbThrough, string(tuuid)) { //已存在第一条记录,记录第二条
			err = s.db.SetTableValue(s.dbThrough, string(tuuid), "ip2", []byte{
				raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15],
			})
			com.Errorlog(err)
			err = s.db.SetTableValue(s.dbThrough, string(tuuid), "port2", []byte(strconv.Itoa(raddr.Port)))
			com.Errorlog(err)

			// 回复
			var ip1 []byte = s.db.ReadTableValue(s.dbThrough, string(tuuid), "ip1")
			if ip1 == nil {
				com.Errorlog(errors.New("can't read ip1, tuuid is:" + visibleSlice(tuuid)))
				return nil
			}
			var port1 []byte = s.db.ReadTableValue(s.dbThrough, string(tuuid), "port1")
			// 回复当前
			bn := append(tuuid, 2, raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port), ip1[0], ip1[2], ip1[3], ip1[4], port1[0], port1[1])
			s.conn.WriteToUDP(bn, raddr)
			// 回复之前
			bb := append(tuuid, 2, ip1[0], ip1[2], ip1[3], ip1[4], port1[0], port1[1], raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port))
			raddr2, err := net.ResolveUDPAddr("udp", string(ip1)+":"+string(string(port1)))
			if com.Errorlog(err) {
				return nil
			}
			laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port)))
			if com.Errorlog(err) {
				return nil
			}
			conn, err := net.DialUDP("udp", laddr, raddr2)
			if com.Errorlog(err) {
				return nil
			}
			defer conn.Close()
			_, err = conn.Write(bb)
			if com.Errorlog(err) {
				return nil
			}

		} else {
			err = s.db.SetTableValue(s.dbThrough, string(tuuid), "ip1", []byte{
				raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15],
			})
			com.Errorlog(err)
			err = s.db.SetTableValue(s.dbThrough, string(tuuid), "port1", []byte(strconv.Itoa(raddr.Port)))
			com.Errorlog(err)
		}
	} else {
		// 不可能

	}

	return nil
}

func (s *STUN) throughClient(tuuid []byte) error {

	_, err = s.conn.Write(tuuid)
	if err != nil {
		return err
	}

	var b []byte = make([]byte, 512)
	for i := 0; i <= 6; i++ { // 等待5s
		err = s.conn.SetReadDeadline(time.Now().Add(time.Second * 1))
		if err != nil {
			return err
		}
		_, err = s.conn.Read(b)
		if err != nil {
			return err
		}
		if bytes.Equal(b[:17], tuuid) {
			break
		} else if i == 6 {
			return errors.New("sever no reply")
		}
	}

	// lip := net.ParseIP(string(b[18:23]))
	// lport := int(b[23])<<8 + int(b[24])
	rip := net.ParseIP(string(b[24:29]))
	rport := int(b[29])<<8 + int(b[30])
	fmt.Println("对方IP", rip.String(), rport)

	laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port)))
	if err != nil {
		return err
	}
	var conns []*net.UDPConn
	for i := 0; i < 5; i++ {
		raddr, err := net.ResolveUDPAddr("udp", rip.String()+":"+strconv.Itoa(rport))
		if err != nil {
			return err
		}
		conn, err := net.DialUDP("udp", laddr, raddr)
		defer conn.Close()
		if err != nil {
			if i == 0 {
				return err
			}
			i--
			continue
		}
		conns = append(conns, conn)
	}

	// 繁杂操作
	// 收
	var ch chan int = make(chan int, 1)
	var da []byte = make([]byte, 64)
	var flag bool = false
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		for i, v := range conns {
			index := i
			conn := v
			go func() {
				for !flag {
					conn.Read(da)
					if bytes.Equal(da[:17], tuuid) && da[17] == 3 {
						wg.Done()
						ch <- index
					}
				}
			}()
		}
		wg.Wait() // 阻塞
	}()
	// 发
	_, err = conns[0].Write(append(tuuid, 3))
	if err != nil {
		return err
	}

	go func() {
		for !flag {
			for _, v := range conns {
				v.Write(append(tuuid, 3))
			}
			time.Sleep(time.Millisecond * 200)
		}
	}()

	var wh int
	select { //阻塞 5s
	case wh = <-ch:
	case <-time.After(time.Second * 5):
		return errors.New("超时无法完成穿隧")
	}

	// 成功一半
	for i, v := range conns {
		if i != wh {
			v.Close()
		}
	}
	conns[wh].Write(append(tuuid, 3))
	conns[wh].Write(append(tuuid, 4))
	for i := 0; i < 20; i++ {
		conns[wh].SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		_, err = conns[wh].Read(da)
		if err != nil {
			if i >= 15 {
				break //返回
			} else {
				i--
			}
		} else if bytes.Equal(da[:17], tuuid) && da[17] == 4 {
			conns[wh].Write(append(tuuid, 4))
		}
	}

	//

	return nil
}

/* other function */

func visibleSlice(b []byte) string {
	var r string
	for _, v := range b {
		r = r + strconv.Itoa(int(v)) + ""
	}
	return r
}
func byteToint(b []byte) int {
	var r, l int = 0, len(b)
	for i, v := range b {
		r = r + int(v)<<(8*(l-i-1))
	}
	return r
}
