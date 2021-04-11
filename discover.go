package stun

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/stun/internal/com"
)

// DiscoverSever
// 参数为第一端口和第二端口，接收到的数据，对方的地址
func (s *STUN) discoverSever(conn, conn2 *net.UDPConn, da []byte, raddr *net.UDPAddr) error {

	if len(da) < 18 {
		return nil
	}
	step := int(da[17])
	juuid := da[:17]

	// 处理step
	v := s.dbd.R(string(juuid), "step")
	if v != "" {
		var st int
		if st, err = strconv.Atoi(v); e.Errlog(err) {
			return nil
		}
		if st >= int(step) { // 记录已经存在
			return nil
		} else { // 更新step
			s.dbd.U(string(juuid), "step", strconv.Itoa(int(step)))
		}
	}

	// 回复函数
	var S = func(conn *net.UDPConn, raddr *net.UDPAddr, da []byte) error {
		for i := 0; i < s.Iterate; i++ {
			if _, err = conn.WriteToUDP(da[:18], raddr); err != nil {
				return err
			}
		}
		return nil
	}

	/* 开始 */
	if step == 1 {
		if len(da) != 20 {
			return nil
		}
		var D map[string]string = make(map[string]string)
		D["step"] = "1"
		D["IP1"] = raddr.IP.String()                         // 第一NAT网关IP
		D["Port1"] = strconv.Itoa(raddr.Port)                // 第一NAT网关端口
		D["c1"] = strconv.Itoa(int(da[18])<<8 + int(da[19])) // 第一使用端口
		s.dbd.Ct(string(juuid), D)

		if err = S(conn, raddr, append(juuid, 2)); e.Errlog(err) {
			return err
		}
	} else {

		var IP1, Port1 string
		if Port1 = s.dbd.R(string(juuid), "Port1"); Port1 == "" {
			return nil
		}
		if IP1 = s.dbd.R(string(juuid), "IP1"); IP1 == "" {
			return nil
		}
		var natAddr1 *net.UDPAddr
		if natAddr1, err = net.ResolveUDPAddr("udp", string(IP1)+":"+string(Port1)); e.Errlog(err) {
			return err
		}

		if step == 3 { //3

			if len(da) != 20 {
				return nil
			}

			if strconv.Itoa(raddr.Port) == string(Port1) { //两次请求端口相同、锥形NAT，需进一步判断 回复4和5

				if err = S(conn2, natAddr1, append(juuid, 4)); e.Errlog(err) { //4
					return err
				}

				if err = S(conn, natAddr1, append(juuid, 5)); e.Errlog(err) { //5
					return err
				}

			} else {

				if raddr.Port == int(da[18])<<8+int(da[19]) && Port1 == s.dbd.R(string(juuid), "c1") { // 两次网关与使用端口相同，公网IP 9

					s.dbd.U(string(juuid), "type", "9")
					if err = S(conn, natAddr1, append(juuid, 9)); e.Errlog(err) {
						return err
					}

				} else { // 对称NAT

					if raddr.Port-natAddr1.Port == 1 { // 顺序
						if err = S(conn, natAddr1, append(juuid, 0xe)); e.Errlog(err) {
							return err
						}
						s.dbd.U(string(juuid), "type", "14")
					} else {
						if err = S(conn, natAddr1, append(juuid, 0xf)); e.Errlog(err) {
							return err
						}
						s.dbd.U(string(juuid), "type", "15")
					}

				}
			}

		} else if step == 6 {

			if s.secondIPConn != nil {
				for i := 0; i < s.Iterate; i++ {
					if _, err = s.secondIPConn.WriteToUDP(append(juuid, 7), raddr); e.Errlog(err) {
						return err
					}
				}
				if err = S(conn, raddr, append(juuid, 8)); e.Errlog(err) {
					return err
				}
				s.dbd.U(string(juuid), "type", "8")
			} else { // 不区分，也回复6
				s.dbd.U(string(juuid), "type", "12")
				S(conn, raddr, append(juuid, 0xc))
			}

		} else if step == 0xa || step == 0xb || step == 0xd { //a b d
			s.dbd.U(string(juuid), "type", strconv.Itoa(int(step)))
		}
	}
	return nil
}

// DiscoverClient
// 参数：c1,c2,s1,s2是端口，sever是服务器IP或域名
func (s *STUN) discoverClient(port int) (int, error) {
	// 返回代码:
	// -1 错误
	//  0 无响应
	//  9 公网IP
	//  a 完全锥形
	//  b IP限制锥形
	//  c 完全锥形或IP限制锥形NAT
	//  d 端口限制锥形
	//  e 顺序对称形NAT
	//  f 无序对称NAT

	var c1, c2 int = port, port + 1

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)

	conn, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: c1}, &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s1})
	if e.Errlog(err) {
		return -1, err
	}
	defer conn.Close()
	conn2, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: c2}, &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s1})
	if e.Errlog(err) {
		return -1, err
	}
	defer conn2.Close()
	/* 初始化完成 */

	// 读取函数，收到对应的数据包返回nil
	var R = func(shouleCode ...uint8) error {
		var ch chan error = make(chan error)
		var flag bool = true
		go func() {
			for flag {
				_, err = conn.Read(da)
				if err != nil {
					ch <- err
					return
				} else if bytes.Equal(juuid, da[:17]) {
					for _, v := range shouleCode {
						if v == da[17] {
							ch <- nil
							return
						}
					}
				}
			}
		}()
		select {
		case err := <-ch:
			flag = false
			return err
		case <-time.After(s.TimeOut):
			flag = false
			return errors.New("timeout")
		}
	}
	var S = func(conn *net.UDPConn, d []byte) error {
		for i := 0; i < s.Iterate; i++ {
			if _, err := conn.Write(d); err != nil {
				return err
			}
		}
		return nil
	}

	/* 开始 */

	// 发 1
	da = append(da, 1, uint8(c1>>8), uint8(c1))
	if err = S(conn, da); e.Errlog(err) {
		return -1, err
	}

	// 收 2
	if err = R(2); err != nil {
		return distinguish(err)
	}

	// 第二端口发 3
	if err = S(conn2, append(juuid, 3, uint8(c2>>8), uint8(c2))); e.Errlog(err) {
		return -1, err
	}

	// 收  4、5，9，0xe,0xf
	err = R(4, 5, 9, 0xe, 0xf)
	if err != nil {
		return distinguish(err)

	} else if da[17] == 9 || da[17] == 0xe || da[17] == 0xf { //公网IP、对称NAT
		return int(da[17]), nil

	} else if da[17] == 4 || da[17] == 5 {
		if da[17] == 5 {
			if err = R(4); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					S(conn, append(juuid, 0xd)) // 收到5，收不到4; 端口限制
					return 0xc, nil
				}
				e.Errlog(err)
				return -1, err
			}
		}
		// 至此，起码已经收到4；为完全锥形或IP限制锥形，接下来可能有进一步判断

		// 发6
		if err = S(conn, append(juuid, 6)); e.Errlog(err) {
			return 0, err
		}

		// 收 第二IP的包Juuid:7 或 Juuid:8 或 6和超时(没有区分)
		err = R(0xc, 7, 8)
		if e.Errlog(err) {
			return -1, err

		} else if da[17] == 0xc {
			return 0xc, nil // 不区分

		} else if da[17] == 8 || da[17] == 7 {
			if da[17] == 8 { //收到8，尝试接收7

				if err = R(7); err != nil {
					if strings.Contains(err.Error(), "timeout") { //IP限制形
						S(conn, append(juuid, 0xb))
						return 0xb, nil
					}
					e.Errlog(err)
					return -1, err
				}
			}
			// 至此，已经接收到7 完全锥形NAT
			S(conn, append(juuid, 0xa))
			return 0xa, nil
		}
	}

	return -1, errors.New("Exception") // 异常
}

func distinguish(err error) (int, error) {
	if strings.Contains(err.Error(), "time") {
		return 0, errors.New("sever no reply")
	}
	return -1, err
}
