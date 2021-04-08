package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

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
			s.reply3(conn, tuuid)
		}

	} else {
		if s.dbt.Et(string(tuuid)) {
			if da[17] == 5 {
				bstep := s.dbt.R(string(tuuid), "step")
				if bstep < strconv.Itoa(int(da[18])) || bstep == "" {
					s.dbt.U(string(tuuid), "step", strconv.Itoa(int(da[18])))
					s.reply6(conn, tuuid, raddr.Port, int(da[18]))
				}
			}
		} else {
			fmt.Println("tuuid不存在")
		}
	}

	return nil
}

func (s *STUN) reply3(conn *net.UDPConn, tuuid []byte) {
	var send = func(tuuid []byte, r1, r2 *net.UDPAddr, v1 uint8) {
		var v2 uint8 = 0
		if v1 == 0 {
			v2 = 1
		}
		for i := 0; i < 5; i++ {
			_, err1 := conn.WriteToUDP(append(tuuid, 3, v1, r2.IP[12], r2.IP[13], r2.IP[14], r2.IP[15], uint8(r2.Port>>8), uint8(r2.Port)), r1)
			_, err2 := conn.WriteToUDP(append(tuuid, 3, v2, r1.IP[12], r1.IP[13], r1.IP[14], r1.IP[15], uint8(r1.Port>>8), uint8(r1.Port)), r2)
			com.Errorlog(err1, err2)
		}
	}

	r1, err1 := net.ResolveUDPAddr("udp", s.dbt.R(string(tuuid), "ip1")+":"+s.dbt.R(string(tuuid), "port1"))
	r2, err2 := net.ResolveUDPAddr("udp", s.dbt.R(string(tuuid), "ip2")+":"+s.dbt.R(string(tuuid), "port2"))
	if com.Errorlog(err1, err2) {
		return
	}

	if s.dbt.R(string(tuuid), "type1") < s.dbt.R(string(tuuid), "type2") { // 1更低
		send(tuuid, r1, r2, 0)
		s.dbt.U(string(tuuid), "lower", "1")
	} else {
		send(tuuid, r1, r2, 1)
		s.dbt.U(string(tuuid), "lower", "2")
	}
}
func (s *STUN) reply6(conn *net.UDPConn, tuuid []byte, port, step int) {
	l := s.dbt.R(string(tuuid), "lower")
	var hr string = "0"
	if l < "1" {
		hr = "1" //通知高的一方
	}
	haddr, err := net.ResolveUDPAddr("udp", s.dbd.R(string(tuuid), "ip"+hr)+":"+s.dbd.R(string(tuuid), "port"+hr))
	com.Errorlog(err)

	for i := 0; i < 5; i++ {
		_, err = conn.WriteToUDP(append(tuuid, 6, byte(port), uint8(step)), haddr)
		com.Errorlog(err)
	}
}

func (s *STUN) ThroughClient(conn *net.UDPConn, tuuid []byte, natType int) (*net.UDPAddr, error) {
	var extPorts int = 3 // 泛端口范围

	for i := 0; i < 5; i++ {
		if _, err := conn.Write(append(tuuid, 1, uint8(natType))); err != nil {
			return nil, err
		}
	}

	// 等待回复
	var j int
	var raddr *net.UDPAddr
	if j, raddr, err = s.read3(conn, tuuid, time.Second*5); err != nil {
		return nil, errors.New("sever no reply")
	}

	// 开始穿隧
	if j == 0 { // 	NAT限制低的一方

		// 请求高的一方的主端口的泛端口
		laddr, err := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
		if com.Errorlog(err) {
			return nil, err
		}
		conn.Close()
		conn, err = net.DialUDP("udp", laddr, raddr)
		for i := 0; i < extPorts; i++ {

			for j := 0; j < 10; j++ {
				// conn.Write(append())
			}

			conn, err = upRUDPConn(conn)
			if com.Errorlog(err) {
				return nil, err
			}
		}

	} else if j == 1 { // NAT限制高的一方

	} else {

		fmt.Println("非法j:", j)
	}

	return nil, nil
}

// 注意读取容量 第一个返回参数应该是0或1、表示相对NAT限制高低，第二第三个参数返回的是对方的IP和主端口
func (s *STUN) read3(conn *net.UDPConn, tuuid []byte, td time.Duration) (int, *net.UDPAddr, error) {
	var b []byte = make([]byte, 64)
	var wg chan int = make(chan int)
	var raddr *net.UDPAddr
	go func() {
		for {
			_, err = conn.Read(b)
			if err != nil {
				fmt.Println("读取错误", err)
				wg <- 2
				return
			}
			if bytes.Equal(b[:17], tuuid) && b[17] == 3 && len(b) < 25 {
				rIP := net.IPv4(b[19], b[20], b[21], b[22])
				rPort := int(b[23])<<8 + int(b[24])
				raddr, err = net.ResolveUDPAddr("udp", rIP.String()+":"+strconv.Itoa(rPort))
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
	case i := <-wg: // 无需操作
		if i != 0 || i != 1 {
			return 0, nil, errors.New("error")
		} else {
			return i, raddr, nil
		}

	case <-time.After(td): // 匹配时间
		return 0, nil, errors.New("timeout")
	}
}

// 将conn对方的端口加1
func upRUDPConn(conn *net.UDPConn) (*net.UDPConn, error) {
	var upPort = func(addr string) (*net.UDPAddr, error) {
		uaddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		rIP := uaddr.IP.String()
		rPort := uaddr.Port + 1
		naddr, err := net.ResolveUDPAddr("udp", rIP+":"+strconv.Itoa(rPort))
		if err != nil {
			return nil, err
		}
		return naddr, nil
	}
	conn.Close()
	nRaddr, err := upPort(conn.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	nLaddr, err := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}

	nconn, err := net.DialUDP("udp", nLaddr, nRaddr)
	if err != nil {
		return nil, err
	}
	return nconn, nil
}
