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
func (s *STUN) judgeSever(conn, conn2, ip2conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {

	if len(da) < 18 {
		return nil
	}
	step := int(da[17])
	juuid := da[:17]

	// 处理step
	v := s.dbd.R(string(juuid), "step")
	if v != "" {
		if v >= strconv.Itoa(int(int(step))) {
			return nil // 记录已经存在 , 过滤
		}
	}
	fmt.Println("放行", da[17])

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
	if step == 10 {
		if len(da) != 20 {
			return nil
		}
		var D map[string]string = make(map[string]string)
		D["step"] = "20"
		D["IP1"] = raddr.IP.String()                         // 第一NAT网关IP
		D["Port1"] = strconv.Itoa(raddr.Port)                // 第一NAT网关端口
		D["c1"] = strconv.Itoa(int(da[18])<<8 + int(da[19])) // 第一使用端口
		s.dbd.Ut(string(juuid), D)

		if err = S(conn, raddr, append(juuid, 20, s.WIP2[12], s.WIP2[13], s.WIP2[14], s.WIP2[15])); e.Errlog(err) {
			return err
		}
		fmt.Println("回复了20", append(juuid, 20, s.WIP2[12], s.WIP2[13], s.WIP2[14], s.WIP2[15]))
	} else {

		var IP1, Port1 string
		if Port1 = s.dbd.R(string(juuid), "Port1"); Port1 == "" {
			return nil
		}
		if IP1 = s.dbd.R(string(juuid), "IP1"); IP1 == "" {
			return nil
		}
		var natAddr1 *net.UDPAddr // 第一次请求的网关地址
		if natAddr1, err = net.ResolveUDPAddr("udp", string(IP1)+":"+string(Port1)); e.Errlog(err) {
			return err
		}

		if step == 30 { //30

			if len(da) != 20 {
				return nil
			}

			if strconv.Itoa(raddr.Port) == string(Port1) { //两次请求端口相同、锥形NAT，需进一步判断 回复40和50
				if err = S(conn2, natAddr1, append(juuid, 40)); e.Errlog(err) { //4
					return err
				}
				if err = S(conn, natAddr1, append(juuid, 50)); e.Errlog(err) { //5
					return err
				}
				s.dbd.U(string(juuid), "step", "50")

			} else { // 两次请求端口不同

				if raddr.Port == int(da[18])<<8+int(da[19]) && Port1 == s.dbd.R(string(juuid), "c1") {
					// 两次网关端口与使用端口相同，公网IP 100

					s.dbd.U(string(juuid), "step", "100")
					if err = S(conn, natAddr1, append(juuid, 100)); e.Errlog(err) {
						return err
					}

				} else { //对称NAT

					if raddr.Port-natAddr1.Port <= 5 { // 相连，为顺序NAT
						if err = S(conn, natAddr1, append(juuid, 110)); e.Errlog(err) {
							return err
						}
						s.dbd.U(string(juuid), "step", "110")

					} else { // 无序对称NAT
						if err = S(conn, natAddr1, append(juuid, 250)); e.Errlog(err) {
							return err
						}
						s.dbd.U(string(juuid), "step", "250")
					}

				}
			}

		} else if step == 60 {
			if len(da) != 18 {
				return nil
			}

			if err = S(ip2conn, raddr, append(juuid, 70)); e.Errlog(err) {
				return nil
			}
			if err = S(conn, raddr, append(juuid, 80)); e.Errlog(err) {
				return nil
			}
			s.dbd.U(string(juuid), "step", "80")

		} else if step == 120 { // 第二IP收到的
			if raddr.Port-natAddr1.Port < 10 { //完全顺序对称NAT
				s.dbd.U(string(juuid), "step", "230")
				S(conn, raddr, append(juuid, 230))

			} else { //IP限制顺序对称NAT
				s.dbd.U(string(juuid), "step", "240")
				S(conn, raddr, append(juuid, 240))
			}

		} else if step == 180 || step == 190 || step == 200 || step == 210 || step == 220 {
			s.dbd.U(string(juuid), "step", strconv.Itoa(int(step)))
		}
	}
	return nil
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

	var c1, c2 int = port, port + 1

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)
	var wip2 net.IP

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
		var n int
		go func() {
			for flag {
				n, err = conn.Read(da)
				da = da[:n]
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
	// 发送函数
	var S = func(conn *net.UDPConn, d []byte) error {
		for i := 0; i < s.Iterate; i++ {
			if _, err := conn.Write(d); err != nil {
				return err
			}
		}
		return nil
	}

	/* 开始 */

	// 发 10
	da = append(da, 10, uint8(c1>>8), uint8(c1))
	if err = S(conn, da); e.Errlog(err) {
		return -1, err
	}

	// 收 20
	if err = R(20); err != nil {
		return distinguish(err)
	}
	if len(da) >= 22 {
		wip2 = net.IPv4(da[18], da[19], da[20], da[21])
	} else {
		fmt.Println(len(da))
		return -1, errors.New("step 20 : Data length less than 22")
	}

	// 第二端口发 30
	if err = S(conn2, append(juuid, 30, uint8(c2>>8), uint8(c2))); e.Errlog(err) {
		return -1, err
	}

	// 收  40,50 ,100,250,90,100,110
	err = R(40, 50, 100, 250, 90, 100, 100)
	if err != nil {
		return distinguish(err)

	} else if da[17] == 250 { //无序对称NAT
		return 250, nil
	} else if da[17] == 40 || da[17] == 50 { // 区分端口限制锥形
		if da[17] == 50 {
			if err = R(40); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					S(conn, append(juuid, 220)) // 收到50，收不到40; 端口限制
					return 220, nil
				}
				e.Errlog(err)
				return -1, err
			}
		}
		// 至此，起码已经收到40；为完全锥形或IP限制锥形

		// 发60
		if err = S(conn, append(juuid, 60)); e.Errlog(err) {
			return 0, err
		}

		// 收 第二IP的包70 或 80
		err = R(0xc, 70, 80)
		if e.Errlog(err) {
			return distinguish(err)

		} else if da[17] == 80 {
			if err = R(70); err != nil { //收到80，尝试接收70
				if strings.Contains(err.Error(), "timeout") { //收不到70 IP限制锥形
					S(conn, append(juuid, 200))
					return 210, nil
				}
			}

			// 至此，已经接收到7 完全锥形NAT
			S(conn, append(juuid, 200))
			return 200, nil
		}
	} else if da[17] == 90 || da[17] == 100 { // 区分具有防火墙的公网IP
		if da[17] == 100 {
			if err = R(90); err != nil {
				if strings.Contains(err.Error(), "time") { // 收不到90
					// 具有防火墙的公网IP
					S(conn, append(juuid, 190))
					return 190, nil
				}
				return -1, err
			}
			// 至此，至少已经收到了90 公网IP
			S(conn, append(juuid, 180))
			return 180, nil
		}
	} else if da[17] == 110 { // 请求sever2
		ip2conn, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: c1}, &net.UDPAddr{IP: wip2, Port: s.s1})
		if e.Errlog(err) {
			return -1, err
		}
		defer ip2conn.Close()
		S(ip2conn, append(juuid, 120))

		// 接收回复 230 240
		if err = R(230, 240); err != nil {
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