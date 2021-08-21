package stun

// NAT类型判断

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/stun/internal/com"
)

// discoverSever
// 参数为第一端口和第二端口，接收到的数据，对方的地址
func (s *sever) discoverSever(da []byte, raddr *net.UDPAddr) {
	// conn1, conn2, ip2conn *net.UDPConn,

	if len(da) < 18 {
		return
	}
	step := int(da[17])
	var juuid = make([]byte, 17)
	copy(juuid, da)

	v := s.dbj.R(string(juuid), "step")
	if v != "" {
		if len(v) > len(strconv.Itoa(int(step))) || v >= strconv.Itoa(int(step)) {
			return
		}
	}

	/* 开始 */
	if step == 10 {
		if len(da) < 20 {
			return
		}

		var D map[string]string = make(map[string]string)
		D["step"] = "20"                                     // 序号20
		D["IP1"] = raddr.IP.String()                         // 第一NAT网关IP
		D["Port1"] = strconv.Itoa(raddr.Port)                // 第一NAT网关端口
		D["c1"] = strconv.Itoa(int(da[18])<<8 + int(da[19])) // 第一使用端口
		s.dbj.Ut(string(juuid), D)

		if err = s.send(s.conn1, append(juuid, 20, s.wip2[12], s.wip2[13], s.wip2[14], s.wip2[15]), raddr); e.Errlog(err) {
			return
		}

	} else {

		var IP1, Port1 string
		if Port1 = s.dbj.R(string(juuid), "Port1"); Port1 == "" {
			return
		}
		if IP1 = s.dbj.R(string(juuid), "IP1"); IP1 == "" {
			return
		}
		var natAddr1 *net.UDPAddr
		if natAddr1, err = net.ResolveUDPAddr("udp", string(IP1)+":"+string(Port1)); e.Errlog(err) {
			return
		}

		if step == 30 {

			fmt.Println("---30----------------------------------")
			fmt.Println("10", natAddr1)
			fmt.Println("30", raddr)

			s.dbj.U(string(juuid), "IP2", raddr.IP.String())
			s.dbj.U(string(juuid), "Port2", strconv.Itoa(raddr.Port))

			if IP1 == raddr.IP.String() {

				if strconv.Itoa(raddr.Port) == string(Port1) { // 锥形NAT
					if err = s.send(s.conn2, append(juuid, 40), natAddr1); e.Errlog(err) {
						return
					}
					if err = s.send(s.conn1, append(juuid, 50), natAddr1); e.Errlog(err) {
						return
					}
					s.dbj.U(string(juuid), "step", "50")

				} else {

					if Port1 == s.dbj.R(string(juuid), "c1") && strconv.Itoa(raddr.Port) == Port1 { // 公网IP

						if err = s.send(s.conn1, append(juuid, 100), natAddr1); e.Errlog(err) {
							return
						}
						s.dbj.U(string(juuid), "step", "100")

					} else { // 对称NAT

						// 判断端口相连
						if raddr.Port-natAddr1.Port <= s.ExtPorts && 0 < raddr.Port-natAddr1.Port {

							if err = s.send(s.conn1, append(juuid, 110), natAddr1); e.Errlog(err) {
								return
							}
							s.dbj.U(string(juuid), "step", "110")

						} else {
							if err = s.send(s.conn1, append(juuid, 250), natAddr1); e.Errlog(err) {
								return
							}
							s.dbj.U(string(juuid), "step", "250")

						}
					}
				}
			} else {
				if err = s.send(s.conn1, append(juuid, 251), natAddr1); e.Errlog(err) {
					return
				}
				s.dbj.U(string(juuid), "step", "251") // NAT有IP池
			}

		} else if step == 60 {
			if len(da) != 18 {
				return
			}

			if err = s.send(s.conn3, append(juuid, 70), raddr); e.Errlog(err) {
				return
			}
			if err = s.send(s.conn1, append(juuid, 80), raddr); e.Errlog(err) {
				return
			}
			s.dbj.U(string(juuid), "step", "80")

		} else if step == 120 {
			// 区分对称NAT

			fmt.Println("请求：10：", natAddr1.IP, natAddr1.Port)
			fmt.Println("请求：120：", raddr.IP, raddr.Port)
			fmt.Println("120 laddr")
			fmt.Println("--------------------------------------------------")

			if !net.IP.Equal(raddr.IP, natAddr1.IP) {

				s.send(s.conn1, append(juuid, 251), natAddr1)
				s.dbj.U(string(juuid), "step", "251")

			} else {
				if raddr.Port-natAddr1.Port == 0 {
					// IP锥形顺序对称NAT
					s.send(s.conn1, append(juuid, 237), natAddr1)
					s.dbj.U(string(juuid), "step", "237")

				} else if raddr.Port-natAddr1.Port > 0 && raddr.Port-natAddr1.Port <= s.ExtPorts {
					// 完全顺序对称NAT
					s.send(s.conn1, append(juuid, 230), natAddr1)
					s.dbj.U(string(juuid), "step", "230")

				} else {
					// IP限制顺序对称NAT
					s.send(s.conn1, append(juuid, 240), natAddr1)
					s.dbj.U(string(juuid), "step", "240")
				}
			}

		} else if step == 180 || step == 190 || step == 200 || step == 210 || step == 220 {
			s.dbj.U(string(juuid), "step", strconv.Itoa(int(step)))
		}
	}
}

// DiscoverCliet
func (s *client) DiscoverCliet() (int, error) {
	// 返回代码:
	// -1 错误
	//  0 无响应
	//  180 公网IP
	//  190 具有防火墙的公网IP
	//  200 完全锥形nat
	//  210 IP限制形nat
	//  220 端口限制nat
	//  230 完全顺序对称NAT
	//  240 IP限制顺序对称NAT
	//  250 无序对称NAT

	var juuid []byte = []byte{'J'}
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)
	var wip2 net.IP
	var raddr1 *net.UDPAddr = &net.UDPAddr{IP: s.sever, Port: s.cp1}
	// var raddr2 *net.UDPAddr = &net.UDPAddr{IP: s.sever, Port: s.cp2} //服务器第二端口

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1})
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	conn2, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp2}, raddr1)
	if err != nil {
		return -1, err
	}
	defer conn2.Close()
	/* 初始化完成 */

	// 读取函数，收到对应的数据包返回nil
	var R = func(shouleCode ...uint8) ([]byte, error) {
		var ch chan error = make(chan error)
		var flag bool = true
		var n int
		da = make([]byte, 64)
		go func() {
			for flag {
				if n, err = conn.Read(da); err != nil {
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
			return da[:n], err
		case <-time.After(s.TimeOut):
			flag = false
			return nil, errors.New("timeout")
		}
	}

	/* 开始 */

	// 10
	if err = s.send(conn, append(da, 10, uint8(s.cp1>>8), uint8(s.cp1)), raddr1); err != nil {
		return -1, err
	}

	// 20
	if da, err = R(20); err != nil {
		return foo(err)
	}
	if len(da) < 22 {
		return -1, errors.New("step 20 : Data length less than 22")
	}
	wip2 = net.IPv4(da[18], da[19], da[20], da[21]) // sever第二IP
	fmt.Println("sever IP2", wip2)

	// 30
	if err = s.send(conn2, append(juuid, 30), nil); err != nil {
		return -1, err
	}

	// 收  40,50 ,100,250,90,100,110
	da, err = R(40, 50, 100, 250, 90, 100, 110)
	if err != nil {
		return foo(err)
	} else if da[17] == 250 || da[17] == 251 { // 250、251
		return int(da[17]), nil

	} else if da[17] == 40 || da[17] == 50 {
		if da[17] == 50 {
			if da, err = R(40); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					s.send(conn, append(juuid, 220), raddr1)
					return 220, nil
				}

				return -1, err
			}
		}

		if err = s.send(conn, append(juuid, 60), raddr1); err != nil {
			return 0, err
		}

		// da, err = R(0xc, 70, 80) ? 什么东西
		da, err = R(70, 80)
		if err != nil {
			return foo(err)

		} else if da[17] == 80 {

			if da, err = R(70); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					s.send(conn, append(juuid, 210), raddr1)
					return 210, nil
				}
			}

			s.send(conn, append(juuid, 200), raddr1)
			return 200, nil
		}
	} else if da[17] == 90 || da[17] == 100 {
		if da[17] == 100 {
			if da, err = R(90); err != nil {
				if strings.Contains(err.Error(), "timeout") {

					s.send(conn, append(juuid, 190), raddr1)
					return 190, nil
				}
				return -1, err
			}

			s.send(conn, append(juuid, 180), raddr1)
			return 180, nil
		}
	} else if da[17] == 110 {

		if err = s.send(conn, append(juuid, 120), &net.UDPAddr{IP: wip2, Port: s.cp1}); err != nil {
			return -1, err
		}

		// test
		if err = s.send(conn, append(juuid, 120), &net.UDPAddr{IP: net.ParseIP("124.70.28.137"), Port: 19986}); err != nil {
			return -1, err
		}

		if da, err = R(230, 240, 250); err != nil {
			return foo(err)
		} else {
			return int(da[17]), nil
		}
	}

	return -1, errors.New("Exception") // 异常
}

// 回复
func (s *STUN) send(conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {
	for i := 0; i < s.reSendTimes; i++ {
		if raddr != nil {
			if _, err := conn.WriteToUDP(da, raddr); err != nil {
				return err
			}
		} else {
			if _, err := conn.Write(da); err != nil {
				return err
			}
		}
	}
	return nil
}

func foo(err error) (int, error) {
	if strings.Contains(err.Error(), "timeout") {
		return 0, errSever
	}
	return -1, err
}
