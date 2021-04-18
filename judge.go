package stun

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

// DiscoverSever
// 参数为第一端口和第二端口，接收到的数据，对方的地址
func (s *STUN) judgeSever(conn1, conn3, ip2conn *net.UDPConn, da []byte, raddr *net.UDPAddr) {

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

		if err = s.send(conn1, append(juuid, 20, s.WIP2[12], s.WIP2[13], s.WIP2[14], s.WIP2[15]), raddr); e.Errlog(err) {
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

			s.dbj.U(string(juuid), "IP2", raddr.IP.String())
			s.dbj.U(string(juuid), "Port2", strconv.Itoa(raddr.Port))

			if strconv.Itoa(raddr.Port) == string(Port1) { // 锥形NAT
				if err = s.send(conn3, append(juuid, 40), natAddr1); e.Errlog(err) {
					return
				}
				if err = s.send(conn1, append(juuid, 50), natAddr1); e.Errlog(err) {
					return
				}
				s.dbj.U(string(juuid), "step", "50")

			} else {

				if Port1 == s.dbj.R(string(juuid), "c1") && strconv.Itoa(raddr.Port) == Port1 { // 公网IP

					if err = s.send(conn1, append(juuid, 100), natAddr1); e.Errlog(err) {
						return
					}
					s.dbj.U(string(juuid), "step", "100")

				} else { // 对称NAT

					if raddr.Port-natAddr1.Port <= 5 && 0 < raddr.Port-natAddr1.Port {

						if err = s.send(conn1, append(juuid, 110), natAddr1); e.Errlog(err) {
							return
						}
						s.dbj.U(string(juuid), "step", "110")

					} else {
						if err = s.send(conn1, append(juuid, 250), natAddr1); e.Errlog(err) {
							return
						}
						s.dbj.U(string(juuid), "step", "250")
					}
				}
			}

		} else if step == 60 {
			if len(da) != 18 {
				return
			}

			if err = s.send(ip2conn, append(juuid, 70), raddr); e.Errlog(err) {
				return
			}
			if err = s.send(conn1, append(juuid, 80), raddr); e.Errlog(err) {
				return
			}
			s.dbj.U(string(juuid), "step", "80")

		} else if step == 120 {
			// 进一步区分对称NAT

			if !net.IP.Equal(raddr.IP, natAddr1.IP) {

				s.send(conn1, append(juuid, 250), natAddr1)
				s.dbj.U(string(juuid), "step", "250")

			} else {
				fmt.Println("第一次请求：", natAddr1.IP, natAddr1.Port)
				fmt.Println("第三次请求：", raddr.IP, raddr.Port)

				if raddr.Port-natAddr1.Port > 0 && raddr.Port-natAddr1.Port <= 10 {

					s.send(conn1, append(juuid, 230), natAddr1)
					s.dbj.U(string(juuid), "step", "230")

				} else {

					s.send(conn1, append(juuid, 240), natAddr1)
					s.dbj.U(string(juuid), "step", "240")
				}
			}

		} else if step == 180 || step == 190 || step == 200 || step == 210 || step == 220 {
			s.dbj.U(string(juuid), "step", strconv.Itoa(int(step)))
		}
	}
}

// DiscoverClient
func (s *STUN) judgeCliet(port int) (int, error) {
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

	var localP1, localP2 int = port, port + 1

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)
	var wip2 net.IP
	var raddr1 *net.UDPAddr = &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s1}
	var raddr2 *net.UDPAddr = &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s2}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: localP1})
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	conn2, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: localP2}, raddr1)
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
	if err = s.send(conn, append(da, 10, uint8(localP1>>8), uint8(localP1)), raddr1); err != nil {
		return -1, err
	}

	// 20
	if da, err = R(20); err != nil {
		return foo(err)
	}
	if len(da) < 22 {
		return -1, errors.New("step 20 : Data length less than 22")
	}
	wip2 = net.IPv4(da[18], da[19], da[20], da[21])

	if err = s.send(conn, append(juuid, 30), raddr2); err != nil {
		return -1, err
	}

	// 收  40,50 ,100,250,90,100,110
	da, err = R(40, 50, 100, 250, 90, 100, 110)
	if err != nil {
		return foo(err)
	} else if da[17] == 250 {
		return 250, nil
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

		da, err = R(0xc, 70, 80)
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

		if err = s.send(conn, append(juuid, 120), &net.UDPAddr{IP: wip2, Port: s.s1}); err != nil {
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
	for i := 0; i < s.Iterate; i++ {
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
		return 0, errors.New("sever no reply or network timeout")
	}
	return -1, err
}
