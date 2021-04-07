package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lysShub/stun/internal/com"
)

// ThroughSever
func (s *STUN) throughSever(da []byte, raddr *net.UDPAddr) error {

	if len(da) != 18 {
		fmt.Println("长度不为18")
		return nil
	}
	if da[17] == 1 { // 首次请求

		tuuid := da[:17]
		if s.dbt.Et(string(tuuid)) { //双方中第二个请求
			if s.dbt.R(string(tuuid), "ip2") == "" {

			}

			fmt.Println("双方中第二个请求")
			// 记录
			s.dbt.U(string(tuuid), "ip2", raddr.IP.String())
			s.dbt.U(string(tuuid), "port2", strconv.Itoa(raddr.Port))

			// 回复
			var ip1 = net.ParseIP(s.dbt.R(string(tuuid), "ip1"))
			if ip1 == nil {
				com.Errorlog(errors.New("can't read ip1, tuuid is:" + visibleSlice(tuuid)))
				return nil
			}
			var sbp = s.dbt.R(string(tuuid), "port1")
			var portb int
			if portb, err = strconv.Atoi(sbp); err != nil {
				com.Errorlog(errors.New("invalid port: " + err.Error()))
				return nil
			}
			// 回复当前
			bn := append(tuuid, 2, raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port), ip1[12], ip1[13], ip1[14], ip1[15], uint8(portb>>8), uint8(portb))
			for i := 0; i < 5; i++ {
				if _, err = s.conn.WriteToUDP(bn, raddr); com.Errorlog(err) {
					return nil
				}
			}

			// 回复之前
			bb := append(tuuid, 2, ip1[12], ip1[13], ip1[14], ip1[15], uint8(portb>>8), uint8(portb), raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15], uint8(raddr.Port>>8), uint8(raddr.Port))
			var firstraddr *net.UDPAddr
			if firstraddr, err = net.ResolveUDPAddr("udp", ip1.String()+":"+sbp); com.Errorlog(err) {
				return nil
			}
			for i := 0; i < 5; i++ {
				if _, err = s.conn.WriteToUDP(bb, firstraddr); com.Errorlog(err) {
					return nil
				}
			}

		} else { // 双方中第一个请求
			if s.dbt.R(string(tuuid), "ip1") == "" {
				fmt.Println("双方中第一个请求")
				s.dbt.Ct(string(tuuid), map[string]string{
					"ip1":   raddr.IP.String(),
					"port1": strconv.Itoa(raddr.Port),
				})
			}
		}
	}

	return nil
}

// ThroughClient
func (s *STUN) ThroughClient(tuuid []byte) (*net.UDPAddr, error) {

	if err = s.Init(true); err != nil { // 必须，s.conn会被重置
		return nil, err
	}

	var portRange int = 5 // 端口猜测范围

	_, err = s.conn.Write(append(tuuid, 1))
	if err != nil {
		return nil, err
	}

	// 等待回复
	var b []byte = make([]byte, 512)
	var wg chan int = make(chan int)
	go func() {
		for {
			_, err = s.conn.Read(b)
			if err != nil {
				return
			}
			if bytes.Equal(b[:17], tuuid) {
				fmt.Println("映射建立，成功一半")
				wg <- 0
				break
			}
		}
	}()
	select {
	case <-wg: // 无需操作
	case <-time.After(time.Second * 30): // 匹配时间
		return nil, errors.New("sever no reply")
	}

	/*  开始穿隧  */
	rip := parseIP(b[24:29])
	rport := int(b[29])<<8 + int(b[30])
	fmt.Println("对方IP", rip.String(), rport) //

	s.conn.Close() // 关闭稍后重置
	var laddr *net.UDPAddr
	if laddr, err = net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(s.Port))); err != nil {
		return nil, err
	}
	var conns []*net.UDPConn
	var raddrs []*net.UDPAddr
	for i := 0; i < portRange; i++ { // 探测与相连的5个端口
		raddr, err := net.ResolveUDPAddr("udp", rip.String()+":"+strconv.Itoa(rport+i))
		if err != nil {
			return nil, err
		}
		raddrs = append(raddrs, raddr)
		conn, err := net.DialUDP("udp", laddr, raddr)
		if err != nil {
			if strings.Contains(err.Error(), "one") { // 端口被占用
				//  dial udp :4433->103.1.1.3:19986: bind: Only one usage of each socket address (protocol/network address/port) is normally permitted.
				portRange++
				continue
			} else {
				fmt.Println("错误序号", i)
				return nil, err
			}
		}
		defer conn.Close()

		conns = append(conns, conn)
	}

	// 收
	var ch chan int = make(chan int, 1)
	var da []byte = make([]byte, 64)
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		for i, v := range conns {
			index := i
			conn := v
			go func() {
				for {
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
	go func() {
		for {
			for i, v := range conns {
				for j := 0; j < 5; j++ {
					v.Write(append(tuuid, 3))
				}
				if i == 0 {
					time.Sleep(time.Millisecond * 100)
				}
			}
			time.Sleep(time.Millisecond * 50)
		}
	}()

	var wh int
	select { //阻塞 5s
	case wh = <-ch:
	case <-time.After(time.Second * 50): //超时时间
		return nil, errors.New("超时无法完成穿隧")
	}

	for i := 0; i < 20; i++ {
		conns[wh].Write(append(tuuid, 3))
	}

	return raddrs[wh], nil
}

func visibleSlice(b []byte) string {
	var r string
	for _, v := range b {
		r = r + strconv.Itoa(int(v)) + ""
	}
	return r
}

func parseIP(b []byte) net.IP {
	var s string
	for i := 0; i < 4; i++ {
		if i == 3 {
			s += strconv.Itoa(int(b[i]))
		} else {
			s += strconv.Itoa(int(b[i])) + `.`
		}
	}
	return net.ParseIP(s)
}
