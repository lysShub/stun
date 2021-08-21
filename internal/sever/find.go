package sever

// 判断NAT类型服务
import (
	"fmt"
	"net"
	"strconv"

	"github.com/lysShub/stun/config"
	"github.com/lysShub/stun/internal/com"
)

var err error

// findSever
// 	@接收到的数据, 对方的地址
func (s *sever) findSever(da []byte, raddr *net.UDPAddr) {
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
		if s.dbj.Et(string(juuid)) {
			return // juuid重复或冗余发送的数据
		} else if len(da) < 20 {
			return
		}

		var D map[string]string = make(map[string]string)
		D["step"] = "20"                                              // 序号20
		D["NATIP1"] = raddr.IP.String()                               // 第一NAT网关IP
		D["NATPort1"] = strconv.Itoa(raddr.Port)                      // 第一NAT网关端口
		D["clientPort1"] = strconv.Itoa(int(da[18])<<8 + int(da[19])) // 第一使用端口
		s.dbj.Ut(string(juuid), D)

		if err = s.Send(s.conn1, append(juuid, 20, s.wip2[12], s.wip2[13], s.wip2[14], s.wip2[15]), raddr); com.Errlog(err) {
			return
		}

	} else {

		var IP1, Port1 string
		if Port1 = s.dbj.R(string(juuid), "NATPort1"); Port1 == "" {
			return
		}
		if IP1 = s.dbj.R(string(juuid), "NATIP1"); IP1 == "" {
			return
		}
		var natAddr1 *net.UDPAddr
		if natAddr1, err = net.ResolveUDPAddr("udp", string(IP1)+":"+string(Port1)); com.Errlog(err) {
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
					if err = s.Send(s.conn2, append(juuid, 40), natAddr1); com.Errlog(err) {
						return
					}
					if err = s.Send(s.conn1, append(juuid, 50), natAddr1); com.Errlog(err) {
						return
					}
					s.dbj.U(string(juuid), "step", "50")

				} else {

					if Port1 == s.dbj.R(string(juuid), "clientPort1") && strconv.Itoa(raddr.Port) == Port1 { // 公网IP

						if err = s.Send(s.conn1, append(juuid, 100), natAddr1); com.Errlog(err) {
							return
						}
						s.dbj.U(string(juuid), "step", "100")

					} else { // 对称NAT

						// 判断端口相连
						if raddr.Port-natAddr1.Port <= config.ExtPorts && 0 < raddr.Port-natAddr1.Port {

							if err = s.Send(s.conn1, append(juuid, 110), natAddr1); com.Errlog(err) {
								return
							}
							s.dbj.U(string(juuid), "step", "110")

						} else {
							if err = s.Send(s.conn1, append(juuid, 250), natAddr1); com.Errlog(err) {
								return
							}
							s.dbj.U(string(juuid), "step", "250")

						}
					}
				}
			} else {
				if err = s.Send(s.conn1, append(juuid, 251), natAddr1); com.Errlog(err) {
					return
				}
				s.dbj.U(string(juuid), "step", "251") // NAT有IP池
			}

		} else if step == 60 {
			if len(da) != 18 {
				return
			}

			if err = s.Send(s.conn3, append(juuid, 70), raddr); com.Errlog(err) {
				return
			}
			if err = s.Send(s.conn1, append(juuid, 80), raddr); com.Errlog(err) {
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

				s.Send(s.conn1, append(juuid, 251), natAddr1)
				s.dbj.U(string(juuid), "step", "251")

			} else {
				if raddr.Port-natAddr1.Port == 0 {
					// IP锥形顺序对称NAT
					s.Send(s.conn1, append(juuid, 237), natAddr1)
					s.dbj.U(string(juuid), "step", "237")

				} else if raddr.Port-natAddr1.Port > 0 && raddr.Port-natAddr1.Port <= config.ExtPorts {
					// 完全顺序对称NAT
					s.Send(s.conn1, append(juuid, 230), natAddr1)
					s.dbj.U(string(juuid), "step", "230")

				} else {
					// IP限制顺序对称NAT
					s.Send(s.conn1, append(juuid, 240), natAddr1)
					s.dbj.U(string(juuid), "step", "240")
				}
			}

		} else if step == 180 || step == 190 || step == 200 || step == 210 || step == 220 {
			s.dbj.U(string(juuid), "step", strconv.Itoa(int(step)))
		}
	}
}
