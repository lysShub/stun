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
}

func (s *STUN) Init(isClient bool) error {

	s.dbDiscover = "discover"
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
			s.conn, err = net.DialUDP("udp", l, r)
			if err != nil {
				s.conn = nil
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

	/*



	 */

	fmt.Println("收到数据", raddr.IP)
	if len(da) != 18 {
		fmt.Println("接收到长度不为18，为", len(da))
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

	} else if step == 2 { // 其他
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
					da[37] = 0xd
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

// Client client
func (s *STUN) discoverClient() (int16, error) {
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

	// 收Juuid:2
	err = s.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return -1, err
	}
	_, err = s.conn.Read(da)
	if err != nil { //超时 服务器没有回复
		return 0xe, errSever

	} else if !bytes.Equal(da[:18], juuid) || da[18] != 2 { // 异常
		return -1, errors.New("Exceptions: need Juuid2, instead " + string(da[:19]))
	}
	fmt.Println("收到2")

	// 第二端口发Juuid:3
	da[18] = 3
	_, err = s.conn2.Write(da[:19])
	if err != nil {
		return -1, err
	}

	// 收Juuid:9 或 Juuid:d 或 Juuid:5(收不到4) 或 Juuid:4(接下来应收到5)
	err = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
	if err != nil {
		return -1, err
	}
	_, err = s.conn.Read(da)
	if err != nil {
		return -1, err
	}

	fmt.Println("收到", string(da[18]))

	if bytes.Equal(da[:18], juuid) && da[18] == 9 { //公网IP
		return 9, nil

	} else if bytes.Equal(da[:18], juuid) && da[18] == 0xd { //对称NAT
		return 0xd, nil

	} else if bytes.Equal(da[:18], juuid) && da[18] == 5 { //收到5且收不到4 端口限制nat

		// 收不到4
		// 回复
		da[18] = 0xc
		_, _ = s.conn.Write(da[:38])
		return 0xc, nil

	} else if bytes.Equal(da[:18], juuid) && da[18] == 4 { //收到4
		// 收 5
		err = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		if err != nil {
			return -1, err
		}
		_, err = s.conn.Read(da)
		if err != nil {
			return 0xe, errSever
		}
		if bytes.Equal(da[:18], juuid) && da[18] == 5 { // 完全或IP限制锥形NAT
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

			} else if bytes.Equal(da[:18], juuid) && da[18] == 8 { // 收到8，

				// 收7(由于UDP不保证数据包到达顺序)
				err = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
				if err != nil {
					return 0, err
				}
				_, err = s.conn.Read(da)
				if err == nil && da[18] == 7 {
					da[18] = 0xa
					s.conn.Write(da[:38])
					return 0xa, nil //完全限制锥形
				}

				// 回复
				da[18] = 0xb
				s.conn.Write(da[:38])
				return 0xb, nil // IP限制锥形

			} else if bytes.Equal(da[:18], juuid) && da[18] == 7 { //收到7
				// 不用再接收8，已经收到7，确定为完全锥形
				da[18] = 0xa
				s.conn.Write(da[:38])
				return 0xa, nil //完全锥形
			}
		} else { // 收到4却收不到5 异常
			return 0xf, nil
		}
	}
	return 0xf, nil
}
