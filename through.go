package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/lysShub/stun/internal/com"
)

// throughSever
func (s *STUN) throughSever(da []byte, raddr *net.UDPAddr) error {

	if len(da) != 18 {
		fmt.Println("长度不为18")
		return nil
	}
	if da[17] == 1 { // 首次请求

		tuuid := da[:17]
		if s.dbt.Et(string(tuuid)) { //双方中第二个请求

			// 记录
			s.dbt.U(string(tuuid), "ip2", raddr.IP.String())
			s.dbt.U(string(tuuid), "port2", strconv.Itoa(raddr.Port))

			// 回复
			var ip1 = net.ParseIP(s.dbt.R(string(tuuid), "ip1"))
			if ip1 == nil {
				com.Errorlog(errors.New("can't read ip1, tuuid is:" + visibleSlice(tuuid)))
				return nil
			}
			var port1 = s.dbt.R(string(tuuid), "port1")

			// 回复当前
			bn := append(tuuid, 2, raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port), ip1[0], ip1[2], ip1[3], ip1[4], port1[0], port1[1])
			s.conn.WriteToUDP(bn, raddr)

			// 回复之前
			var portb int
			if portb, err = strconv.Atoi(port1); err != nil {
				com.Errorlog(errors.New("invalid port: " + err.Error()))
			}
			bb := append(tuuid, 2, ip1[0], ip1[2], ip1[3], ip1[4], uint8(portb>>8), uint8(portb), raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port))
			var raddr2, laddr *net.UDPAddr
			if raddr2, err = net.ResolveUDPAddr("udp", ip1.String()+":"+port1); com.Errorlog(err) {
				return nil
			}
			if laddr, err = net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port))); com.Errorlog(err) {
				return nil
			}
			var conn *net.UDPConn
			if conn, err = net.DialUDP("udp", laddr, raddr2); com.Errorlog(err) {
				return nil
			}
			defer conn.Close()

			if _, err = conn.Write(bb); com.Errorlog(err) {
				return nil
			}

		} else { // 双方中第一个请求
			s.dbt.U(string(tuuid), "ip1", raddr.IP.String())
			s.dbt.U(string(tuuid), "port1", strconv.Itoa(raddr.Port))

		}
	} else {
		// 不可能

	}

	return nil
}

func (s *STUN) throughClient(tuuid []byte) error {

	_, err = s.conn.Write(tuuid)
	if err != nil {
		return err
	}

	var b []byte = make([]byte, 512)
	for i := 0; i <= 6; i++ { // 等待5s
		err = s.conn.SetReadDeadline(time.Now().Add(time.Second * 1))
		if err != nil {
			return err
		}
		_, err = s.conn.Read(b)
		if err != nil {
			return err
		}
		if bytes.Equal(b[:17], tuuid) {
			break
		} else if i == 6 {
			return errors.New("sever no reply")
		}
	}

	// lip := net.ParseIP(string(b[18:23]))
	// lport := int(b[23])<<8 + int(b[24])
	rip := net.ParseIP(string(b[24:29]))
	rport := int(b[29])<<8 + int(b[30])
	fmt.Println("对方IP", rip.String(), rport)

	laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port)))
	if err != nil {
		return err
	}
	var conns []*net.UDPConn
	for i := 0; i < 5; i++ {
		raddr, err := net.ResolveUDPAddr("udp", rip.String()+":"+strconv.Itoa(rport))
		if err != nil {
			return err
		}
		conn, err := net.DialUDP("udp", laddr, raddr)
		defer conn.Close()
		if err != nil {
			if i == 0 {
				return err
			}
			i--
			continue
		}
		conns = append(conns, conn)
	}

	// 繁杂操作
	// 收
	var ch chan int = make(chan int, 1)
	var da []byte = make([]byte, 64)
	var flag bool = false
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		for i, v := range conns {
			index := i
			conn := v
			go func() {
				for !flag {
					conn.Read(da)
					if bytes.Equal(da[:17], tuuid) && da[17] == 3 {
						wg.Done()
						ch <- index
					}
				}
			}()
		}
		wg.Wait() // 阻塞
	}()
	// 发
	_, err = conns[0].Write(append(tuuid, 3))
	if err != nil {
		return err
	}

	go func() {
		for !flag {
			for _, v := range conns {
				v.Write(append(tuuid, 3))
			}
			time.Sleep(time.Millisecond * 200)
		}
	}()

	var wh int
	select { //阻塞 5s
	case wh = <-ch:
	case <-time.After(time.Second * 5):
		return errors.New("超时无法完成穿隧")
	}

	// 成功一半
	for i, v := range conns {
		if i != wh {
			v.Close()
		}
	}
	conns[wh].Write(append(tuuid, 3))
	conns[wh].Write(append(tuuid, 4))
	for i := 0; i < 20; i++ {
		conns[wh].SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		_, err = conns[wh].Read(da)
		if err != nil {
			if i >= 15 {
				break //返回
			} else {
				i--
			}
		} else if bytes.Equal(da[:17], tuuid) && da[17] == 4 {
			conns[wh].Write(append(tuuid, 4))
		}
	}

	//

	return nil
}

func visibleSlice(b []byte) string {
	var r string
	for _, v := range b {
		r = r + strconv.Itoa(int(v)) + ""
	}
	return r
}
