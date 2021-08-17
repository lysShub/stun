package stun

import (
	"bytes"
	"errors"
	"net"
	"time"
)

func (s *sconn) throughSever() {

}

func (s *cconn) throughClient(lNatType int, id [16]byte) error {
	if err = checkNatType(lNatType); err != nil {
		return err
	}
	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1}, &net.UDPAddr{IP: s.sever, Port: s.cp1}); err != nil {
		return err
	}
	defer conn.Close()
	s.flag = append([]byte{'T'}, id[:]...)

	var da []byte = make([]byte, 64)
	var n int

	// 注册 [flag 0 natType]
	var reg bool = false
	go func() {
		for {
			if n, err = conn.Read(da); err == nil { // [flag 1] 注册成功
				if n >= 18 && bytes.Equal(append(s.flag, 1), da[:18]) {
					reg = true
					return
				}
			}
		}
	}()
	for i := 0; i < 15 && !reg; i++ {
		if _, err = conn.Write(append(s.flag, 0, byte(lNatType))); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 200)
	}
	if !reg {
		return errors.New("register failed")
	}

	// 匹配 (15s超时)
	var rNatType int
	go func() {
		for {
			if n, err = conn.Read(da); err == nil { // [flag 3 rNatType raddr] 匹配成功
				if n >= 25 && bytes.Equal(s.flag, da[:17]) && da[17] == 3 {
					s.raddr = &net.UDPAddr{IP: net.IPv4(da[18], da[19], da[20], da[21]), Port: int(da[22])<<8 + int(da[23])}
					rNatType = int(da[24])
					return
				}
			}
		}
	}()
	for i := 0; i < 30 && s.raddr == nil; i++ {
		if _, err = conn.Write(append(s.flag, 2)); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 500)
	}
	if s.raddr == nil {
		return errors.New("match timeout")
	}
	conn.Close()

	// 开始穿透
	var class int = classify(lNatType, rNatType)
	if class == 0 {
		return errors.New("NAT类型不支持穿透")
	} else if class == 10 {

	} else if class == 20 {

	} else if class == 30 {

	} else if class == 40 {

	} else if class == 110 {

	} else if class == 120 {

	} else if class == 130 {

	} else if class == 140 {

	}

	return nil
}

func classify(lNatType, rNatType int) int {
	var t = lNatType + rNatType
	if lNatType >= rNatType {

		if lNatType == 230 && rNatType == 230 {
			return 20
		} else if lNatType == 250 && rNatType == 220 {
			return 30
		} else if lNatType == 250 && rNatType == 230 {
			return 40
		} else if t <= 460 {
			return 10
		}

	} else {
		if rNatType == 230 {
			return 110
		} else if lNatType == 220 && (rNatType == 240 || rNatType == 250) {
			return 140
		} else if lNatType == 230 && rNatType == 250 {
			return 130
		} else if lNatType <= 210 && rNatType != 251 {
			return 110
		}
	}
	return 0 // 无法穿透
}

func checkNatType(natType int) error {
	if natType == 180 || natType == 190 || natType == 200 || natType == 210 || natType == 220 || natType == 230 || natType == 240 || natType == 250 || natType == 251 {
		return nil
	}
	return errors.New("invalid nat type")
}
