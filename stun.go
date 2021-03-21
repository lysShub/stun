package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
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

// DiscoverSever
func (s *STUN) discoverSever(da []byte, raddr *net.UDPAddr) error {

	var step uint16 = 0
	var juuid []byte = nil

	fmt.Println("收到数据", raddr.IP)
	if len(da) != 18 {
		fmt.Println("接收到长度不为18,为", len(da))
		return nil
	}
	step = uint16(da[17])
	juuid = da[:17]

	if step == 1 { //1 开始
		var D map[string][]byte = make(map[string][]byte)
		D["step"] = []byte{1}
		D["rIP"] = []byte(raddr.IP.String())
		D["rPort"] = []byte(strconv.Itoa(raddr.Port))
		err = s.db.SetTableRow(s.dbDiscover, string(juuid), D)
		if err != nil {
			return err
		}
		D = nil

		// 回复
		da[17] = 2 //2
		_, err := s.conn.WriteToUDP(da[:18], raddr)
		if err != nil {
			return err
		}
		fmt.Println("回复了2")

	} else {
		rPort := s.db.ReadTableValue(s.dbDiscover, string(juuid), "rPort")
		if rPort == nil {
			fmt.Println("无法获取到数据库记录")
			return nil
		}
		rIP := s.db.ReadTableValue(s.dbDiscover, string(juuid), "rIP")
		if rIP == nil {
			fmt.Println("无法获取到数据库记录")
			return nil
		}

		if step == 3 { //3
			raddr1, err := net.ResolveUDPAddr("udp", string(rIP)+":"+string(rPort))
			if err != nil {
				return err
			}

			if strconv.Itoa(raddr.Port) == string(rPort) { //两次请求端口不同，需进一步判断 回复4和5
				da[17] = 4 //4
				_, err = s.conn2.WriteToUDP(da[:18], raddr1)
				if err != nil {
					return err
				}

				time.Sleep(time.Millisecond * 300)
				da[17] = 5 //5
				_, err = s.conn.WriteToUDP(da[:18], raddr1)
				if err != nil {
					return err
				}
			} else {
				if raddr.Port == int(s.Port) && string(rPort) == strconv.Itoa(int(s.SecondPort)) { // 两次端口与预定义端口相对，公网IP 9
					err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{9})
					if err != nil {
						return err
					}
					da[17] = 9
					_, err = s.conn.WriteToUDP(da[:18], raddr1)
					if err != nil {
						return err
					}

				} else { // 对称NAT d
					err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{0xd})
					if err != nil {
						return err
					}
					da[17] = 0xd
					_, err = s.conn.WriteToUDP(da[:18], raddr1)
					if err != nil {
						return err
					}
				}
			}

		} else if step == 6 {

			if s.secondIPConn != nil { //回复 7 8
				// 回复 7(确保有效)
				da[17] = 7
				_, err = s.secondIPConn.WriteToUDP(da[:18], raddr)
				if err != nil {
					return err
				}
				// 回复8
				da[17] = 8
				_, err = s.conn.WriteToUDP(da[:38], raddr)
				if err != nil {
					return err
				}

			} else { // 不区分
				err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{6})
				if err != nil {
					return err
				}
			}

		} else if step == 0xa || step == 0xb || step == 0xc { //a b c

			err = s.db.SetTableValue(s.dbDiscover, string(juuid), "type", []byte{uint8(step)})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DiscoverClient
func (s *STUN) DiscoverClient() (int16, error) {

	// return code:
	// -1 error
	//  6 Full Cone or Restricted Cone
	//  9 No NAT(Public IP)
	//  a Full Cone
	//  b Restricted Cone
	//  c Port Restricted Cone
	//  d Symmetric Cone
	//  e Sever no response
	//  f Exceptions

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)

	fmt.Println("开始")
	/*
	* 操作
	 */

	// 发Juuid:1
	da := []byte(juuid)
	da = append(da, 1)
	_, err = s.conn.Write(da)
	if err != nil {
		return -1, err
	}
	fmt.Println("发送1")
	a := time.Now().UnixNano()
	// 收Juuid:2
	err = s.conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	if err != nil {
		return -1, err
	}
	_, err = s.conn.Read(da)
	b := time.Now().UnixNano()
	fmt.Println("接收2等待", (b-a)/1e6)
	if err != nil { //超时 服务器没有回复
		return 0xe, errSever

	} else if !bytes.Equal(da[:17], juuid) || da[17] != 2 { // 异常
		fmt.Println("step", da[17])
		return -1, errors.New("Exceptions: need Juuid2, instead " + string(da[:18]))
	}
	fmt.Println("收到2")

	// 第二端口发Juuid:3
	da[17] = 3
	_, err = s.conn2.Write(da[:18])
	if err != nil {
		return -1, err
	}
	a = time.Now().UnixNano()
	// 收Juuid:9 或 Juuid:d 或 Juuid:5(收不到4) 或 Juuid:4(接下来应收到5)
	err = s.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return -1, err
	}
	_, err = s.conn.Read(da)
	b = time.Now().UnixNano()
	fmt.Println("发送3后等待", (b-a)/1e6)

	if err != nil {
		return -1, err
	}

	fmt.Println("收到", string(da[17]))

	if bytes.Equal(da[:17], juuid) && da[17] == 9 { //公网IP
		return 9, nil

	} else if bytes.Equal(da[:17], juuid) && da[17] == 0xd { //对称NAT
		return 0xd, nil

	} else if bytes.Equal(da[:17], juuid) && da[17] == 5 { //收到5且收不到4 端口限制nat

		// 收不到4
		// 回复
		da[17] = 0xc
		_, _ = s.conn.Write(da[:38])
		return 0xc, nil

	} else if bytes.Equal(da[:17], juuid) && da[17] == 4 { //收到4
		// 收 5
		err = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		if err != nil {
			return -1, err
		}
		_, err = s.conn.Read(da)
		if err != nil {
			return 0xe, errSever
		}
		if bytes.Equal(da[:17], juuid) && da[17] == 5 { // 完全或IP限制锥形NAT
			// 收 第二IP的包Juuid:7 或 Juuid:8 或 超时(没有区分)

			err = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
			if err != nil {
				return 0, err
			}
			_, err = s.conn.Read(da)
			if err != nil {
				if strings.Contains(err.Error(), "time") { //超时 不区分
					return 6, nil
				}
				return 0, err

			} else if bytes.Equal(da[:17], juuid) && da[17] == 8 { // 收到8，

				// 收7(由于UDP不保证数据包到达顺序)
				err = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
				if err != nil {
					return 0, err
				}
				_, err = s.conn.Read(da)
				if err == nil && da[17] == 7 {
					da[17] = 0xa
					s.conn.Write(da[:38])
					return 0xa, nil //完全限制锥形
				}

				// 回复
				da[17] = 0xb
				s.conn.Write(da[:38])
				return 0xb, nil // IP限制锥形

			} else if bytes.Equal(da[:17], juuid) && da[17] == 7 { //收到7
				// 不用再接收8，已经收到7，确定为完全锥形
				da[17] = 0xa
				s.conn.Write(da[:38])
				return 0xa, nil //完全锥形
			}
		} else { // 收到4却收不到5 异常
			return 0xf, nil
		}
	}
	return 0xf, nil
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

	laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port)))
	if err != nil {
		return err
	}
	var conns [5]*net.UDPConn
	for i := 0; i < 5; i++ {
		raddr, err := net.ResolveUDPAddr("udp", rip.String()+":"+strconv.Itoa(rport))
		if err != nil {
			return err
		}
		conn, err := net.DialUDP("udp", laddr, raddr)
		if err != nil {
			return err
		}
		conns[i] = conn
	}

	// 繁杂操作

	return nil
}

/* other function */

func visibleSlice(b []byte) string {
	var r string
	for _, v := range b {
		r = r + strconv.Itoa(v) + ""
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
