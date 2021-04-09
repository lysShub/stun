package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/stun/internal/com"
)

// ThroughSever
func (s *STUN) throughSever(conn *net.UDPConn, da []byte, raddr *net.UDPAddr) error {

	tuuid := da[:17]
	if da[17] == 1 {
		if s.dbt.R(string(tuuid), "ip1") == "" {
			s.dbt.Ct(string(tuuid), map[string]string{
				"ip1":   raddr.IP.String(),
				"port1": strconv.Itoa(raddr.Port),
				"type1": strconv.Itoa(int(da[18])),
			})

		} else if s.dbt.R(string(tuuid), "ip2") == "" {
			s.dbt.Ct(string(tuuid), map[string]string{
				"ip2":   raddr.IP.String(),
				"port2": strconv.Itoa(raddr.Port),
				"type2": strconv.Itoa(int(da[18])),
			})
			/* 回复 */
			if err = s.reply3(tuuid, da, raddr, conn); e.Errlog(err) {
				return err
			}
		}

	} else {
		if s.dbt.Et(string(tuuid)) {
			if da[17] == 5 {
				if s.dbt.R(string(tuuid), "step") == "" {
					s.dbt.U(string(tuuid), "step", "5")
					s.reply6(conn, tuuid, raddr.Port)
				}
			}
		} else {
			fmt.Println("tuuid不存在")
		}
	}

	return nil
}
func (s *STUN) reply3(tuuid, da []byte, raddr *net.UDPAddr, conn *net.UDPConn) error {
	// 回复3
	var rPort1 int
	if rPort1, err = strconv.Atoi(s.dbt.R(string(tuuid), "port1")); e.Errlog(err) {
		return err
	}
	r1 := &net.UDPAddr{IP: net.ParseIP(s.dbt.R(string(tuuid), "ip1")), Port: rPort1}
	var conn1 *net.UDPConn
	if conn1, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.S1}, r1); e.Errlog(err) {
		return err
	}

	if s.dbt.R(string(tuuid), "type1") < strconv.Itoa(int(da[18])) { // 1更低(0)
		for i := 0; i < s.Iterate; i++ {
			if _, err = conn.WriteToUDP(append(tuuid, 3, 1, r1.IP[12], r1.IP[13], r1.IP[14], r1.IP[15], uint8(r1.Port>>8), uint8(r1.Port)), raddr); e.Errlog(err) {
				return err
			}
			if _, err = conn1.Write(append(tuuid, 3, 0, raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port))); e.Errlog(err) {
				return err
			}
		}

		s.dbt.U(string(tuuid), "lower", "1")
	} else {
		for i := 0; i < s.Iterate; i++ {
			if _, err = conn.WriteToUDP(append(tuuid, 3, 0, r1.IP[12], r1.IP[13], r1.IP[14], r1.IP[15], uint8(r1.Port>>8), uint8(r1.Port)), raddr); e.Errlog(err) {
				return err
			}
			if _, err = conn1.Write(append(tuuid, 3, 1, raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port))); e.Errlog(err) {
				return err
			}
		}

		s.dbt.U(string(tuuid), "lower", "2")
	}
	return nil
}
func (s *STUN) reply6(conn *net.UDPConn, tuuid []byte, port int) error {
	//通知高的一方
	lr := s.dbt.R(string(tuuid), "lower")
	if lr == "" {
		return errors.New("no lower")
	}
	var hr string = "2"
	if lr == "2" {
		hr = "1"
	}
	var lport int
	if lport, err = strconv.Atoi(s.dbd.R(string(tuuid), "port"+hr)); e.Errlog(err) {
		return err
	}
	for i := 0; i < s.Iterate; i++ {
		if _, err = conn.WriteToUDP(append(tuuid, 6, uint8(lport), uint8(lport>>8)), &net.UDPAddr{IP: net.ParseIP(s.dbd.R(string(tuuid), "ip"+hr)), Port: port}); e.Errlog(err) {
			return err
		}
	}
	return nil
}

//
func (s *STUN) ThroughClient(tuuid []byte, natType int) (*net.UDPAddr, error) {

	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.C1}, &net.UDPAddr{IP: s.SIP, Port: s.S1}); e.Errlog(err) {
		return nil, err
	}
	defer conn.Close()

	for i := 0; i < s.Iterate; i++ {
		if _, err := conn.Write(append(tuuid, 1, uint8(natType))); e.Errlog(err) {
			return nil, err
		}
	}

	// 等待回复
	var j int
	var raddr *net.UDPAddr // 对方使用端口对应的网关地址
	if j, raddr, err = s.read3(conn, tuuid, s.TimeOut); e.Errlog(err) {
		return nil, errors.New("sever no reply")
	}

	// 开始穿隧
	if j == 0 { // 	NAT限制低的一方
		/* 使用端口请求对方使用端口对应网关端口的范端口 */
		conn.Close()
		conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.C1}, raddr)
		for i := 0; i < s.ExtPorts; i++ {

			for j := 0; j < s.Iterate; j++ {
				if _, err = conn.Write(append(tuuid, 4, uint8(s.C1>>8), uint8(s.C1))); e.Errlog(err) { //4
					return nil, err
				}
			}
			if conn, err = upRUDPConn(conn); com.Errlog(err) { // 更新conn
				return nil, err
			}
		}
		conn.Close()
		// 通知sver 5
		if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.C1}, &net.UDPAddr{IP: s.SIP, Port: s.S1}); e.Errlog(err) {
			return nil, err
		}
		for i := 0; i < s.Iterate; i++ {
			if _, err = conn.Write(append(tuuid, 5, uint8(s.C1>>8), uint8(s.C1))); e.Errlog(err) {
				return nil, err
			}
		}
		conn.Close()

		// 监听
		if conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: s.C1}); e.Errlog(err) {

		}

	} else if j == 1 { // NAT限制高的一方

	} else {

		fmt.Println("非法j:", j)
	}

	return nil, nil
}
func (s *STUN) read3(conn *net.UDPConn, tuuid []byte, td time.Duration) (int, *net.UDPAddr, error) {
	// 注意读取容量 第一个返回参数应该是0或1、表示相对NAT限制高低，第二第三个参数返回的是对方的IP和主端口

	var b []byte = make([]byte, 64)
	var wg chan int = make(chan int)
	var raddr *net.UDPAddr
	var flag bool = true
	go func() {
		for flag {
			_, err = conn.Read(b)
			if err != nil {
				fmt.Println("读取错误", err)
				flag = false
				wg <- 2
				return
			}
			if bytes.Equal(b[:17], tuuid) && b[17] == 3 && len(b) < 25 {
				rIP := net.IPv4(b[19], b[20], b[21], b[22])
				rPort := int(b[23])<<8 + int(b[24])
				raddr, err = net.ResolveUDPAddr("udp", rIP.String()+":"+strconv.Itoa(rPort))
				flag = false
				if err != nil {
					fmt.Println("生成对方地址错误", err)
					wg <- 2
					return
				}
				wg <- int(b[18])
				return
			} else {
				fmt.Println("读取到但是不符合条件")
			}
		}
	}()

	select {
	case i := <-wg:
		if i == 2 {
			return 0, nil, errors.New("error: invlid setep 3 data")
		} else {
			return i, raddr, nil
		}
	case <-time.After(td): // 匹配时间
		return 0, nil, errors.New("timeout")
	}
}
func upRUDPConn(conn *net.UDPConn) (*net.UDPConn, error) {
	// 将conn对方的端口加1

	var nLaddr, nRaddr *net.UDPAddr

	nRaddr = func(addr string) *net.UDPAddr {
		uaddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil
		}
		rIP := uaddr.IP.String()
		rPort := uaddr.Port + 1
		naddr, err := net.ResolveUDPAddr("udp", rIP+":"+strconv.Itoa(rPort))
		if err != nil {
			return nil
		}
		return naddr
	}(conn.RemoteAddr().String())
	if nRaddr == nil {
		return nil, errors.New("invlid nRaddr")
	}
	conn.Close()

	if nLaddr, err = net.ResolveUDPAddr("udp", conn.LocalAddr().String()); err != nil {
		return nil, err
	}

	var nConn *net.UDPConn
	if nConn, err = net.DialUDP("udp", nLaddr, nRaddr); err != nil {
		return nil, err
	}
	return nConn, nil
}
