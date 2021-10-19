package client

// NAT 类型判断

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"time"

	"stun/config"
	"stun/internal/com"
)

// findClient 发现NAT类型
// 	@port NAT findClient使用的客户端端口, 会使用port及port+1端口; 短时间多次允许时port应该互不相同
func (c *client) findClient(port int) (int, error) {

	// var juuid []byte = []byte{'J', c.id[0], c.id[1], c.id[2], c.id[3], c.id[4], c.id[5], c.id[6], c.id[7], c.id[8], c.id[9], c.id[10], c.id[11], c.id[12], c.id[13], c.id[14], c.id[15]}

	var juuid []byte = append([]byte{'J'}, com.CreateUUID()...)
	var da []byte = make([]byte, 64)

	var conn, conn2 *net.UDPConn
	if conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: port}); err != nil {
		return -1, err
	}
	defer conn.Close()
	var severAddr1 *net.UDPAddr = &net.UDPAddr{IP: c.sever, Port: 19986}
	if conn2, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: port + 1}, severAddr1); err != nil {
		return -1, err
	}
	defer conn2.Close()
	go func() {
		for {
			if n, err := conn.Read(da); err != nil {
				return
			} else {
				ch <- da[:n]
			}
		}
	}()

	/* 初始化完成 */
	// 10
	if err = c.Send(conn, append(juuid, 10, uint8(port>>8), uint8(port)), severAddr1); err != nil {
		return -1, err
	}

	// 20
	if da, err = r(juuid, 20); err != nil {
		return foo(err)
	} else if len(da) < 22 {
		return -1, errors.New("invalid data")
	}

	severIP2 := net.IPv4(da[18], da[19], da[20], da[21]) // sever第二IP

	// 30
	if err = c.Send(conn2, append(juuid, 30), nil); err != nil {
		return -1, err
	}

	// 收  40,50 ,100,250,90,100,110
	da, err = r(juuid, 40, 50, 100, 250, 90, 100, 110)

	if err != nil {
		return foo(err)
	} else if da[17] == 250 || da[17] == 251 { // 250、251
		return int(da[17]), nil

	} else if da[17] == 40 || da[17] == 50 {
		if da[17] == 50 {
			if _, err = r(juuid, 40); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					c.Send(conn, append(juuid, 220), severAddr1)
					return 220, nil
				}
				return -1, err
			}
		}

		if err = c.Send(conn, append(juuid, 60), severAddr1); err != nil {
			return 0, err
		}

		da, err = r(juuid, 70, 80)
		if err != nil {
			return foo(err)

		} else if da[17] == 80 {

			if _, err = r(juuid, 70); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					c.Send(conn, append(juuid, 210), severAddr1)
					return 210, nil
				}
			}

			c.Send(conn, append(juuid, 200), severAddr1)
			return 200, nil
		}
	} else if da[17] == 90 || da[17] == 100 {
		if da[17] == 100 {
			if _, err = r(juuid, 90); err != nil {
				if strings.Contains(err.Error(), "timeout") {

					c.Send(conn, append(juuid, 190), severAddr1)
					return 190, nil
				}
				return -1, err
			}

			c.Send(conn, append(juuid, 180), severAddr1)
			return 180, nil
		}
	} else if da[17] == 110 {

		if err = c.Send(conn, append(juuid, 120), &net.UDPAddr{IP: severIP2, Port: 19986}); err != nil {
			return -1, err
		}

		if da, err = r(juuid, 230, 237, 240, 250); err != nil {
			return foo(err)
		} else {
			return int(da[17]), nil
		}
	}

	return -1, errors.New("Exception") // 异常
}

var errSever error = errors.New("sever no reply or network timeout")

var ch chan []byte = make(chan []byte)

// r 接收函数
func r(juuid []byte, shouleCode ...uint8) ([]byte, error) {

	var deatline = time.After(config.TimeOut)
	var da = make([]byte, 64)

	for {
		select {
		case <-deatline:
			return nil, errors.New("timeout")
		case da = <-ch:
			if len(da) > len(juuid) && bytes.Equal(juuid, da[:len(juuid)]) {
				for _, v := range shouleCode {
					if v == da[17] {
						return da, nil
					}
				}
			}
		}
	}
}

func foo(err error) (int, error) {
	if strings.Contains(err.Error(), "timeout") {
		return 0, errSever
	}
	return -1, err
}
