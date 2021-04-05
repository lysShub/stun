package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

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

	s.dbd = new(mapdb.Db)
	go s.dbd.Init()

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
	var t int = 5 // 回复重复包数
	var step uint16 = 0
	var juuid []byte
	var rIP, rPort string

	if len(da) != 18 {
		return nil
	}
	step = uint16(da[17])
	juuid = da[:17]

	v := s.dbd.R(string(juuid), "step")
	if v != "" {
		var s int
		if s, err = strconv.Atoi(v); err != nil {
			return nil
		}
		if s >= int(step) { // 记录已经存在
			fmt.Println("拦截", juuid, step)
			return nil
		}
	}

	// 回复函数
	var mS = func(raddr *net.UDPAddr, da []byte) error {
		for i := 0; i < t; i++ {
			if _, err = s.conn.WriteToUDP(da[:18], raddr); err != nil {
				return err
			}
		}
		return nil
	}

	/* 开始 */
	if step == 1 {
		var D map[string]string = make(map[string]string)
		D["step"] = "1"
		D["rIP"] = raddr.IP.String()
		D["rPort"] = strconv.Itoa(raddr.Port)
		s.dbd.Ct(string(juuid), D)

		da[17] = 2
		if err = mS(raddr, da); err != nil {
			return err
		}
	} else {

		if rPort = s.dbd.R(string(juuid), "rPort"); rPort == "" {
			fmt.Println("没有获取到数据")
			return nil
		}
		if rIP = s.dbd.R(string(juuid), "rIP"); rIP == "" {
			fmt.Println("没有获取到数据")
			return nil
		}

		if step == 3 { //3

			var raddr1 *net.UDPAddr
			if raddr1, err = net.ResolveUDPAddr("udp", string(rIP)+":"+string(rPort)); err != nil {
				return err
			}

			if strconv.Itoa(raddr.Port) == string(rPort) { //两次请求端口相同，需进一步判断 回复4和5

				da[17] = 4 //4
				if err = mS(raddr1, da); err != nil {
					return err
				}

				da[17] = 5 //5
				if err = mS(raddr1, da); err != nil {
					return err
				}

			} else {
				if raddr.Port == int(s.Port) && string(rPort) == strconv.Itoa(int(s.SecondPort)) { // 两次端口与预定义端口相对，公网IP 9

					s.dbd.U(string(juuid), "type", "9")
					da[17] = 9
					if err = mS(raddr1, da); err != nil {
						return err
					}

				} else {
					s.dbd.U(string(juuid), "type", "13")

					da[17] = 0xd
					if err = mS(raddr1, da); err != nil {
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
				if err = mS(raddr, da); err != nil {
					return err
				}

			} else { // 不区分，没有回复
				s.dbd.U(string(juuid), "type", "6")
			}

		} else if step == 0xa || step == 0xb || step == 0xc { //a b c
			s.dbd.U(string(juuid), "type", strconv.Itoa(int(step)))
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

	var t int = 5 // 回复重复包数
	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)
	da = append(da, 1)

	//读取函数，收到对应的数据包返回nil
	var R = func(shouleCode ...uint8) error {
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

	var S = func(conn *net.UDPConn, d []byte) error {
		for i := 0; i < t; i++ {
			if _, err := conn.Write(d); err != nil {
				return err
			}
		}
		return nil
	}

	/* start */

	// 发 1
	if err = S(s.conn, da); err != nil {
		return 0, err
	}

	// 收 2
	if err = R(2); err != nil { //timeout sever offline
		return 0xe, errSever
	}

	// 第二端口发 3
	da[17] = 3
	if err = S(s.conn2, da[:18]); err != nil {
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
					// s.conn.Write(da[:18])
					S(s.conn, da[:18])
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
						S(s.conn, da[:18])
						return 0xb, nil
					}
					return 0, err
				}
			}

			// 至此，已经接收到7 完全锥形NAT
			da[17] = 0xa
			S(s.conn, da[:18])

			return 0xa, nil
		}
	}

	return 0xf, nil // 异常
}
