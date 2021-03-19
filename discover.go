package stun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/lysShub/stun/internal/com"

	"github.com/lysShub/kvdb"
)

type Discover struct {
	// NAT类型判断

	// 服务器地址，IP或域名
	SeverAddr string
	// 第一端口，sever/client须相同, 19978
	FirstPort uint16
	// 第二端口，sever/client须相同, 19987
	SecondPort uint16
	// 第二网卡局域网IP，可选。如果不设置则不会区分IP与端口限制形NAT
	SeconNetCardIP net.IP
	seconConn      *net.UDPConn
}

var err error
var errSever error = errors.New("Server is not responding")

func (d *Discover) init() error {
	if d.SeconNetCardIP != nil {
		l, err := net.ResolveUDPAddr("udp", d.SeconNetCardIP.String()+":"+strconv.Itoa(int(d.FirstPort)))
		if err != nil {
			return nil
		}
		d.seconConn, err = net.ListenUDP("udp", l)
		if err != nil {
			d.seconConn = nil
			return err
		}
	}
	return nil
}

// Client client
func (d *Discover) Client() (int16, error) {
	// return code:
	// -1 error
	//  6 Full Cone or Restricted Cone
	//  9 No NAT(Public IP)
	//  a Full Cone
	//  b Restricted Cone
	//  c Port Restricted Cone
	//  d Symmetric Cone
	//  e Sever no response
	//  f Exceptions
	raddr1, err1 := net.ResolveUDPAddr("udp", d.SeverAddr+":"+strconv.Itoa(int(d.FirstPort)))
	laddr1, err2 := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(d.FirstPort)))
	laddr2, err3 := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(int(d.SecondPort)))
	if err = com.Errorlog(err1, err2, err3); err != nil {
		return -1, err
	}

	// conn1:f<==>f conn2：s<==>f
	conn1, err1 := net.DialUDP("udp", laddr1, raddr1)
	conn2, err2 := net.DialUDP("udp", laddr2, raddr1)
	if err = com.Errorlog(err1, err2, err3); err != nil {
		return -1, err
	}

	var juuid []byte
	juuid = append(juuid, 'J')
	juuid = append(juuid, com.CreateUUID()...)

	fmt.Println("开始")
	/*
	* 操作
	 */

	// 发Juuid:1
	da := []byte(juuid)
	da = append(da, 1)
	_, err1 = conn1.Write(da)
	if err1 != nil {
		return -1, err
	}

	// 收Juuid:2
	err1 = conn1.SetReadDeadline(time.Now().Add(time.Second))
	_, err2 = conn1.Read(da)
	if err = com.Errorlog(err1, err2); err != nil { //超时 服务器没有回复
		return 0xe, errSever
	} else if !bytes.Equal(da[:18], juuid) || da[18] != 2 {
		return -1, errors.New("Exceptions: need Juuid2, instead " + string(da[:19]))
	}
	fmt.Println("收到2")

	// 第二端口发Juuid:3
	da[18] = 3
	_, err1 = conn2.Write(da[:19])
	if err1 != nil {
		return -1, err1
	}

	// 收Juuid:9 或 Juuid:d 或 Juuid:5(收不到4) 或 Juuid:4(接下来应收到5)
	err1 = conn1.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
	_, err2 = conn1.Read(da)

	fmt.Println(string(da[18]))

	if err = com.Errorlog(err1, err2); err != nil { //超时或错误
		return 0xe, errSever

	} else if bytes.Equal(da[:18], juuid) && da[18] == 9 { //公网IP
		return 9, nil

	} else if bytes.Equal(da[:18], juuid) && da[18] == 0xd { //对称NAT
		return 0xd, nil

	} else if bytes.Equal(da[:18], juuid) && da[18] == 5 { //收到5收不到4 端口限制nat
		// 回复
		da[18] = 0xc
		_, _ = conn1.Write(da[:38])
		return 0xc, nil

	} else if bytes.Equal(da[:18], juuid) && da[18] == 4 { //收到4
		// 收 5
		err1 = conn1.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		_, err2 = conn1.Read(da)
		if err = com.Errorlog(err1, err2); err != nil {
			return 0xe, errSever
		}
		if bytes.Equal(da[:18], juuid) && da[18] == 5 { // 完全或IP限制锥形NAT
			// 收 第二IP的包Juuid:7 或 Juuid:8 或 超时(可能没有区分)
			err1 = conn1.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
			_, err2 = conn1.Read(da)

			if err = com.Errorlog(err1, err2); err != nil {
				return 6, nil // 完全或IP限制锥形NAT

			} else if bytes.Equal(da[:18], juuid) && da[18] == 8 { // IP限制
				// 回复
				da[18] = 0xb
				conn1.Write(da[:38])
				return 0xb, nil

			} else if bytes.Equal(da[:18], juuid) && da[18] == 7 { //收到7
				// 不用再接收8，已经收到7，确定为完全锥形

				da[18] = 0xa
				conn1.Write(da[:38])
				return 0xa, nil //完全锥形
			}
		} else {
			return 0xf, nil
		}
	}
	return 0xf, nil
}

//
func (d *Discover) Sever(ForgeSrcIPCanUse bool) error {
	// distinguish Full Cone and Restricted Cone need different IP,
	// We can forge src IP or using two VPS(network card)
	// note: Router usually discards forged IP packet
	if err = d.init(); err != nil {
		return err
	}

	laddr1, err1 := net.ResolveUDPAddr("udp", ":19987")
	laddr2, err2 := net.ResolveUDPAddr("udp", ":19988")
	lh, err3 := net.ListenUDP("udp", laddr1)
	lh2, err4 := net.ListenUDP("udp", laddr2)
	if err = com.Errorlog(err1, err2, err3, err4); err != nil {
		return err
	}
	defer lh.Close()

	var db = new(kvdb.KVDB)
	db.Type = 0
	db.RAMMode = true
	if err = db.Init(); err != nil {
		return err
	}
	const NatDiscover = "NatDiscover" //表名

	var da []byte = make([]byte, 256)
	var step uint16 = 0
	var juuid []byte = nil
	for {
		n, raddr, err1 := lh.ReadFromUDP(da)
		fmt.Println("收到数据", raddr.IP)
		if err != nil || n != 38 {
			continue
		}
		step = uint16(da[37])
		juuid = da[:37]

		if step == 1 { //1
			var D map[string][]byte = make(map[string][]byte)
			D["step"] = []byte{1}
			D["rIP"] = []byte(raddr.IP.String())
			D["rPort"] = []byte(strconv.Itoa(raddr.Port))
			err1 = db.SetTableRow(NatDiscover, string(juuid), D)
			D = nil

			// 回复
			da[37] = 2 //2
			_, err2 := lh.WriteToUDP(da[:38], raddr)
			if err = com.Errorlog(err1, err2); err != nil {
				continue
			}
			fmt.Println("回复了2")

		} else {
			rPort1 := db.ReadTableValue(NatDiscover, string(juuid), "rPort")
			rIP1 := db.ReadTableValue(NatDiscover, string(juuid), "rIP")
			if rPort1 == nil || rIP1 == nil {
				continue
			} else if step == 3 { //3
				raddr1, err1 := net.ResolveUDPAddr("udp", string(rIP1)+":"+string(rPort1))
				if err1 != nil {
					continue
				}

				if strconv.Itoa(raddr.Port) == string(rPort1) { //需进一步判断 回复4和5
					da[37] = 4 //4
					_, err1 = lh2.WriteToUDP(da[:38], raddr1)

					time.Sleep(time.Millisecond * 300)
					da[37] = 5 //5
					_, err2 = lh.WriteToUDP(da[:38], raddr1)
					if err = com.Errorlog(err1, err2); err != nil {
						continue
					}
				} else {
					if raddr.Port == 19988 && string(rPort1) == "19987" { // 公网IP 9
						err1 = db.SetTableValue(NatDiscover, string(juuid), "type", []byte{9})
						da[37] = 9
						_, err2 = lh.WriteToUDP(da[:38], raddr1)
						if err = com.Errorlog(err1, err2); err != nil {
							continue
						}
					} else { // 对称NAT d
						err1 = db.SetTableValue(NatDiscover, string(juuid), "type", []byte{0xd})
						da[37] = 0xd
						_, err2 = lh.WriteToUDP(da[:38], raddr1)
						if err = com.Errorlog(err1, err2); err != nil {
							continue
						}
					}
				}

			} else if step == 54 { //6

				if ForgeSrcIPCanUse { //回复 7 8
					// 回复 7(确保有效)
					da[37] = 55 // 7
					// rsfu := rawnet.SendForgeSrcIPUDP(raddr.IP, net.ParseIP(svcf.FORGESRCIP), 19987, uint16(raddr.Port), d[:38])

					_, err = d.seconConn.WriteToUDP(da[:18], raddr)
					if err = com.Errorlog(err); err != nil {
						continue
					}
					// 回复8
					da[37] = 56 //8
					_, err1 = lh.WriteToUDP(da[:38], raddr)
					if err = com.Errorlog(err1); err != nil {
						continue
					}

				} else {
					db.SetTableValue(NatDiscover, string(juuid), "type", []byte{6})
				}

			} else if step == 0xa || step == 0xb || step == 0xc { //a b c

				db.SetTableValue(NatDiscover, string(juuid), "type", []byte{uint8(step)})
			}
		}

	}
}
