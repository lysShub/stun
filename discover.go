package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/lysShub/stun/internal/com"
)

// DiscoverSever
// 参数为第一端口和第二端口，接收到的数据，对方的地址
func (s *STUN) discoverSever(s1, s2 int, da []byte, raddr *net.UDPAddr) error {
	var t int = 5 // 回复重复包数
	var step uint16 = 0
	var juuid []byte

	if len(da) < 18 {
		fmt.Println("长度小于18")
		return nil
	}
	step = uint16(da[17])
	juuid = da[:17]

	// 处理step
	v := s.dbd.R(string(juuid), "step")
	if v != "" {
		var st int
		if st, err = strconv.Atoi(v); err != nil {
			return nil
		}
		if st >= int(step) { // 记录已经存在
			fmt.Println("拦截", juuid, step)
			return nil
		} else { // 更新step
			s.dbd.U(string(juuid), "step", strconv.Itoa(int(step)))
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
		if len(da) != 20 {
			fmt.Println("第一次长度不为20")
			return nil
		}
		var D map[string]string = make(map[string]string)
		D["step"] = "1"
		D["nIP1"] = raddr.IP.String()                        // 第一NAT网关IP
		D["nPort1"] = strconv.Itoa(raddr.Port)               // 第一NAT网关端口
		D["cl"] = strconv.Itoa(int(da[18])<<8 + int(da[19])) // 第一使用端口

		s.dbd.Ct(string(juuid), D)

		da[17] = 2
		if err = mS(raddr, da); err != nil {
			return err
		}
	} else {

		var nIP1, nPort1 string
		if nPort1 = s.dbd.R(string(juuid), "nPort1"); nPort1 == "" {
			fmt.Println("没有获取到数据")
			return nil
		}
		if nIP1 = s.dbd.R(string(juuid), "nIP1"); nIP1 == "" {
			fmt.Println("没有获取到数据")
			return nil
		}
		var rNatAddr1 *net.UDPAddr
		if rNatAddr1, err = net.ResolveUDPAddr("udp", string(nIP1)+":"+string(nPort1)); err != nil {
			return err
		}

		if step == 3 { //3

			if len(da) != 20 {
				fmt.Println("第3次长度不为20")
				return nil
			}

			if strconv.Itoa(raddr.Port) == string(nPort1) { //两次请求端口相同、锥形NAT，需进一步判断 回复4和5

				da[17] = 4 //4
				if err = mS(rNatAddr1, da); err != nil {
					return err
				}

				da[17] = 5 //5
				if err = mS(rNatAddr1, da); err != nil {
					return err
				}

			} else {
				if raddr.Port == int(da[18])<<8+int(da[19]) && nPort1 == s.dbd.R(string(juuid), "c1") { // 两次网关与使用端口相同，公网IP 9

					s.dbd.U(string(juuid), "type", "9")
					da[17] = 9
					if err = mS(rNatAddr1, da); err != nil {
						return err
					}

				} else {
					s.dbd.U(string(juuid), "type", "13")

					da[17] = 0xd
					if err = mS(rNatAddr1, da); err != nil {
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

			} else { // 不区分，也回复6
				s.dbd.U(string(juuid), "type", "6")
				da[17] = 6
				mS(rNatAddr1, da)
			}

		} else if step == 0xa || step == 0xb || step == 0xc { //a b c
			s.dbd.U(string(juuid), "type", strconv.Itoa(int(step)))
		}
	}
	return nil
}

// DiscoverClient
// 参数：c1,c2,s1,s2是端口，sever是服务器IP或域名
func (s *STUN) DiscoverClient(c1, c2, s1, s2 int, sever string) (int16, error) {
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

	var t int = 5                          // 回复重复包数
	var td time.Duration = time.Second * 5 //读取超时
	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)
	var da []byte = []byte(juuid)

	laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(c1))
	if err != nil {
		return 0, err
	}
	laddr2, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(c1))
	if err != nil {
		return 0, err
	}
	raddr, err := net.ResolveUDPAddr("udp", sever+":"+strconv.Itoa(s1))
	if err != nil {
		return 0, err
	}
	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	conn2, err := net.DialUDP("udp", laddr2, raddr)
	if err != nil {
		return 0, err
	}
	defer conn2.Close()
	/* 初始化完成 */

	// 读取函数，收到对应的数据包返回nil
	var R = func(td time.Duration, shouleCode ...uint8) error {
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
		case <-time.After(td):
			flag = false
			return errors.New("timeout")
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

	/* 开始 */

	// 发 1
	da = append(da, 1, uint8(c1>>8), uint8(c1))
	if err = S(conn, da); err != nil {
		return 0, err
	}

	// 收 2
	if err = R(td, 2); err != nil {
		return 0xe, errSever
	}

	// 第二端口发 3
	da[17] = 3
	if err = S(conn2, da[:18]); err != nil {
		return 0, err
	}

	// 收  9,d,4,5
	err = R(td, 9, 0xd, 4, 5)
	if err != nil {
		return 0, err
	} else if da[17] == 9 { //公网IP
		return 9, nil

	} else if da[17] == 0xd { //对称NAT
		return 0xd, nil

	} else if da[17] == 4 || da[17] == 5 {
		if da[17] == 5 {
			err = R(td, 4)
			if err != nil {
				if strings.Contains(err.Error(), "time") {
					da[17] = 0xc // 收到5，收不到4
					S(conn, da[:18])
					return 0xc, nil
				}
				return 0, err
			}
		}
		// 至此，起码已经收到4；为完全锥形或IP限制锥形，接下来可能有进一步判断

		// 发6
		da = append(juuid, 6)
		if err = S(conn, da); err != nil {
			return 0, err
		}

		// 收 第二IP的包Juuid:7 或 Juuid:8 或 6和超时(没有区分)
		err = R(td, 6, 7, 8)
		if err != nil || da[17] == 6 {
			if strings.Contains(err.Error(), "time") { //超时或返回6 不区分
				return 6, nil
			}
			return 0, err

		} else if da[17] == 8 || da[17] == 7 {
			if da[17] == 8 { //收到8，尝试接收7

				err = R(td, 7)
				if err != nil {
					if strings.Contains(err.Error(), "time") { //超时
						da[17] = 0xb
						S(conn, da[:18])
						return 0xb, nil
					}
					return 0, err
				}
			}

			// 至此，已经接收到7 完全锥形NAT
			da[17] = 0xa
			S(conn, da[:18])

			return 0xa, nil
		}
	}

	return 0xf, nil // 异常
}
