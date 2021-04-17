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
		fmt.Println("小于18")
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
		fmt.Println(s.dbj.M)

		if err = s.Send(conn1, append(juuid, 20, s.WIP2[12], s.WIP2[13], s.WIP2[14], s.WIP2[15]), raddr); e.Errlog(err) {
			return
		}
	} else {
		fmt.Println("进入", step)

		var IP1, Port1 string
		if Port1 = s.dbj.R(string(juuid), "Port1"); Port1 == "" {
			fmt.Println("不能获得1")
			return
		}
		if IP1 = s.dbj.R(string(juuid), "IP1"); IP1 == "" {
			fmt.Println("不能获得1")
			return
		}
		var natAddr1 *net.UDPAddr
		if natAddr1, err = net.ResolveUDPAddr("udp", string(IP1)+":"+string(Port1)); e.Errlog(err) {
			return
		}

		if step == 30 {

			// s.dbj.U(string(juuid), "IP2", raddr.IP.String())
			// s.dbj.U(string(juuid), "Port2", strconv.Itoa(raddr.Port))
			// s.dbj.U(string(juuid), "c2", strconv.Itoa(int(da[18])<<8+int(da[19])))

			fmt.Println("30", s.dbj.M)

			if strconv.Itoa(raddr.Port) == string(Port1) { // 锥形NAT
				if err = s.Send(conn3, append(juuid, 40), natAddr1); e.Errlog(err) {
					return
				}
				if err = s.Send(conn1, append(juuid, 50), natAddr1); e.Errlog(err) {
					return
				}
				s.dbj.U(string(juuid), "step", "50")

			} else {

				if Port1 == s.dbj.R(string(juuid), "c1") && strconv.Itoa(raddr.Port) == Port1 { // 公网IP

					if err = s.Send(conn1, append(juuid, 100), natAddr1); e.Errlog(err) {
						return
					}
					s.dbj.U(string(juuid), "step", "100")

				} else { // 对称NAT

					if raddr.Port-natAddr1.Port <= 5 && 0 < raddr.Port-natAddr1.Port {
						if err = s.Send(conn1, append(juuid, 110), natAddr1); e.Errlog(err) {
							return
						}
						s.dbj.U(string(juuid), "step", "110")

					} else {
						if err = s.Send(conn1, append(juuid, 250), natAddr1); e.Errlog(err) {
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

			if err = s.Send(ip2conn, append(juuid, 70), raddr); e.Errlog(err) {
				return
			}
			if err = s.Send(conn1, append(juuid, 80), raddr); e.Errlog(err) {
				return
			}
			s.dbj.U(string(juuid), "step", "80")

		} else if step == 120 { // 第二IP
			if len(da) != 18 {
				return
			}

			fmt.Println("120", s.dbj.M)
			fmt.Println("120", raddr.IP, raddr.Port)

			if !net.IP.Equal(raddr.IP, natAddr1.IP) {

				s.Send(conn1, append(juuid, 250), natAddr1)
				s.dbj.U(string(juuid), "step", "250")

			} else {
				var bias int = raddr.Port - natAddr1.Port
				if (bias < 10 && bias > 0) || (bias > -10 && bias < 0) {

					s.Send(conn1, append(juuid, 230), natAddr1)
					s.dbj.U(string(juuid), "step", "230")

				} else {

					s.Send(conn1, append(juuid, 240), natAddr1)
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
	//  9 公网IP
	//  a 完全锥形
	//  b IP限制锥形
	//  c 完全锥形或IP限制锥形NAT
	//  d 端口限制锥形
	//  e 顺序对称形NAT
	//  f 无序对称NAT

	var localP1, localP2 int = port, port + 1

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)
	var wip2 net.IP
	var raddr1 *net.UDPAddr = &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s1}
	var raddr2 *net.UDPAddr = &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s2}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: localP1})
	if e.Errlog(err) {
		return -1, err
	}
	defer conn.Close()
	conn2, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: localP2}, raddr1)
	if e.Errlog(err) {
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
				n, err = conn.Read(da)
				if err == nil {
				}
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
			return da[:n], err
		case <-time.After(s.TimeOut):
			flag = false
			return nil, errors.New("timeout")
		}
	}

	/* 开始 */

	// 发 10
	if err = s.Send(conn, append(da, 10, uint8(localP1>>8), uint8(localP1)), raddr1); e.Errlog(err) {
		return -1, err
	}

	// 收 20
	if da, err = R(20); err != nil {
		return distinguish(err)
	}
	fmt.Println("step", da[17])

	if len(da) >= 22 {
		wip2 = net.IPv4(da[18], da[19], da[20], da[21])
	} else {
		fmt.Println(len(da))
		return -1, errors.New("step 20 : Data length less than 22")
	}

	// 第二端口发 30
	if err = s.Send(conn, append(juuid, 30), raddr2); e.Errlog(err) {
		return -1, err
	}
	fmt.Println("回复了30")

	// 收  40,50 ,100,250,90,100,110
	da, err = R(40, 50, 100, 250, 90, 100, 110)
	if err != nil {
		return distinguish(err)
	} else if da[17] == 250 { //无序对称NAT
		return 250, nil
	} else if da[17] == 40 || da[17] == 50 { // 区分端口限制锥形
		if da[17] == 50 {
			if da, err = R(40); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					s.Send(conn, append(juuid, 220), raddr1) // 收到50，收不到40; 端口限制
					return 220, nil
				}
				e.Errlog(err)
				return -1, err
			}
		}
		// 至此，起码已经收到40；为完全锥形或IP限制锥形

		// 发60
		if err = s.Send(conn, append(juuid, 60), raddr1); e.Errlog(err) {
			return 0, err
		}

		// 收 第二IP的包70 或 80
		da, err = R(0xc, 70, 80)
		if e.Errlog(err) {
			return distinguish(err)

		} else if da[17] == 80 {
			if da, err = R(70); err != nil { //收到80，尝试接收70
				if strings.Contains(err.Error(), "timeout") { //收不到70 IP限制锥形
					s.Send(conn, append(juuid, 200), raddr1)
					return 210, nil
				}
			}

			// 至此，已经接收到7 完全锥形NAT
			s.Send(conn, append(juuid, 200), raddr1)
			return 200, nil
		}
	} else if da[17] == 90 || da[17] == 100 { // 区分具有防火墙的公网IP
		if da[17] == 100 {
			if da, err = R(90); err != nil {
				if strings.Contains(err.Error(), "time") { // 收不到90
					// 具有防火墙的公网IP
					s.Send(conn, append(juuid, 190), raddr1)
					return 190, nil
				}
				return -1, err
			}
			// 至此，至少已经收到了90 公网IP
			s.Send(conn, append(juuid, 180), raddr1)
			return 180, nil
		}
	} else if da[17] == 110 { // 请求sever2
		fmt.Println("收到110")

		// 第二IP
		s.Send(conn, append(juuid, 120), &net.UDPAddr{IP: wip2, Port: s.s1})

		// 接收回复 230 240
		if da, err = R(230, 240, 250); err != nil {
			return distinguish(err)
		} else {
			return int(da[17]), nil
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

// 回复函数
func (s *STUN) Send(conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {
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
