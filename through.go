package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/lysShub/e"
)

// ThroughSever
func (s *STUN) throughSever(conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {

	tuuid := da[:17]
	if da[17] == 10 {
		if len(da) < 21 {
			return errors.New("长度小于21")
		}
		if s.dbt.R(string(tuuid), "ip1") == "" {
			s.dbt.Ct(string(tuuid), map[string]string{
				"ip1":   raddr.IP.String(),
				"port1": strconv.Itoa(raddr.Port),
				"nat1":  strconv.Itoa(int(da[18])),
				"ep1":   strconv.Itoa(int(da[19])<<8 + int(da[20])),
			})
			fmt.Print("第一个", raddr.IP.String(), strconv.Itoa(raddr.Port), strconv.Itoa(int(da[18])), strconv.Itoa(int(da[19])<<8+int(da[20])))
			// 221.197.232.84 55804 15 3
			if s.dbt.R(string(tuuid), "ip1") == "" {
				fmt.Println("-------------------------写入失败-------------------------------")
			}

		} else if s.dbt.R(string(tuuid), "ip2") == "" {
			s.dbt.Ct(string(tuuid), map[string]string{
				"ip2":   raddr.IP.String(),
				"port2": strconv.Itoa(raddr.Port),
				"nat2":  strconv.Itoa(int(da[18])),
				"ep2":   strconv.Itoa(int(da[19])<<8 + int(da[20])),
			})
			fmt.Print("第二个", raddr.IP.String(), strconv.Itoa(raddr.Port), strconv.Itoa(int(da[18])), strconv.Itoa(int(da[19])<<8+int(da[20])))

			/* 回复 */
			if err = s.send20(tuuid, raddr, conn); e.Errlog(err) {
				return err
			}
		}

	} else {
		fmt.Println("不存在")
	}

	return nil
}

// ThroughClient
// 返回对方网关地址和对方NAT类型
func (s *STUN) throughClient(tuuid []byte, port, natType int) (*net.UDPAddr, int, error) {

	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: port}, &net.UDPAddr{IP: net.ParseIP(s.Sever), Port: s.s1}); e.Errlog(err) {
		return nil, 0, err
	}
	defer conn.Close()

	// 发10
	for i := 0; i < s.Iterate; i++ {
		if _, err := conn.Write(append(tuuid, 10, uint8(natType), uint8(s.ExtPorts>>8), uint8(s.ExtPorts))); e.Errlog(err) {
			return nil, 0, err
		}
	}

	// 等待匹配完成 收20
	var rnat, ep int        // nat类型 泛端口长度
	var cRaddr *net.UDPAddr // 对方使用端口对应的网关地址
	if rnat, ep, cRaddr, err = s.read20(conn, tuuid); e.Errlog(err) {
		return nil, 0, errors.New("sever no reply")
	}
	conn.Close()
	fmt.Println("匹配成功")

	// 开始穿隧
	if conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: port}); e.Errlog(err) {
		return nil, 0, err
	}
	defer conn.Close()

	var flag bool = true
	go func() { // 向对方泛端口发送数据 30
		for flag {
			for i := cRaddr.Port; i < cRaddr.Port+ep; i++ {
				conn.WriteToUDP(append(tuuid, 30), &net.UDPAddr{IP: cRaddr.IP, Port: i})
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	var ch chan *net.UDPAddr = make(chan *net.UDPAddr)
	go func() { // 接收数据 30或40
		var da []byte = make([]byte, 64)
		var nRaddr *net.UDPAddr
		for flag {
			if _, nRaddr, err = conn.ReadFromUDP(da); e.Errlog(err) {
				continue
			}
			if bytes.Equal(tuuid, da[:17]) && (da[17] == 30 || da[17] == 40) {
				if da[17] == 30 { // 收到30，回复40后退出
					for i := 0; i < s.Iterate*4; i++ {
						if _, err = conn.WriteToUDP(append(tuuid, 40), nRaddr); e.Errlog(err) {
							continue
						}
					}
				}
				// 收到40，退出

				flag = false // 退出读、发协程
				ch <- nRaddr
			}
		}
	}()

	select {
	case r := <-ch:
		// 穿隧成功
		return r, rnat, nil
	case <-time.After(time.Second * 2):
		// 穿隧失败
		return nil, rnat, nil
	}
}

func (s *STUN) send20(tuuid []byte, raddr *net.UDPAddr, conn *net.UDPConn) error {
	var rPort1 int
	if rPort1, err = strconv.Atoi(s.dbt.R(string(tuuid), "port1")); e.Errlog(err) {
		return err
	}
	var r1, r2 *net.UDPAddr = nil, raddr
	r1 = &net.UDPAddr{IP: net.ParseIP(s.dbt.R(string(tuuid), "ip1")), Port: rPort1}
	var conn1, conn2 *net.UDPConn = nil, conn
	if conn1, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.s1}, r1); err != nil {
		return err
	}
	var ep string = s.dbt.R(string(tuuid), "ep1")
	if ep < s.dbt.R(string(tuuid), "ep2") {
		ep = s.dbt.R(string(tuuid), "ep2")
	}
	var epint int
	if epint, err = strconv.Atoi(ep); e.Errlog(err) {
		return err
	}

	for i := 0; i < s.Iterate; i++ {
		if _, err = conn2.WriteToUDP(append(tuuid, 20, uint8(epint), r1.IP[12], r1.IP[13], r1.IP[14], r1.IP[15], uint8(r1.Port>>8), uint8(r1.Port)), raddr); err != nil {
			return err
		}
		if _, err = conn1.Write(append(tuuid, 20, uint8(epint), r2.IP[12], r2.IP[13], r2.IP[14], r2.IP[15], uint8(r2.Port>>8), uint8(r2.Port))); err != nil {
			return err
		}
	}
	return nil
}
func (s *STUN) read20(conn *net.UDPConn, tuuid []byte) (int, int, *net.UDPAddr, error) {
	// 返回对方nat类型、泛端口长度、对方网关地址
	var b []byte = make([]byte, 64)
	var wg chan error = make(chan error)
	var raddr *net.UDPAddr
	var t, ep int = 0, 0
	var flag bool = true
	go func() {
		var n int
		for flag {
			if n, err = conn.Read(b); err != nil {
				wg <- err
				return
			}
			if bytes.Equal(b[:17], tuuid) && b[17] == 20 && n >= 27 {
				t = int(b[18])
				ep = int(b[19])<<8 + int(b[20])
				raddr = &net.UDPAddr{IP: net.IPv4(b[21], b[22], b[23], b[24]), Port: int(b[25])<<8 + int(b[26])}
				wg <- nil
				return
			} else {
				fmt.Println("读取到但是不符合条件", b[17], n)
			}
		}
	}()

	select {
	case i := <-wg:
		if i != nil {
			return 0, 0, nil, i
		} else {
			return t, ep, raddr, nil
		}
	case <-time.After(s.MatchTime): // 匹配时间
		flag = true //退出协程
		return 0, 0, nil, errors.New("timeout")
	}
}
