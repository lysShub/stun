package stun

import (
	"bytes"
	"errors"
	"net"
	"time"

	"github.com/lysShub/stun/internal/com"
)

// 实现穿隧

func (s *client) Action10() error {

	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1}, s.raddr); err != nil {
		return err
	}
	defer conn.Close()

	var da []byte = make([]byte, 60)
	var r bool = false
	go func() {
		for {
			if n, err := conn.Read(da); err == nil {
				if n >= 17 && bytes.Equal(da[:17], s.flag) {
					r = true
					return
				}
			}
		}
	}()
	for i := 0; i < 30 && !r; i++ {
		if _, err := conn.Write(s.flag); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 100)
	}

	if r {
		return errors.New("穿透成功: " + conn.LocalAddr().String() + ", " + conn.RemoteAddr().String())
	} else {
		return errors.New("穿透失败")
	}
}

func (s *client) Action20() error {
	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1}, &net.UDPAddr{IP: s.raddr.IP, Port: s.raddr.Port + 1}); err != nil {
		return err
	}
	defer conn.Close()

	var da []byte = make([]byte, 60)
	var r bool = false
	go func() {
		for {
			if n, err := conn.Read(da); err == nil {
				if n >= 17 && bytes.Equal(da[:17], s.flag) {
					r = true
					return
				}
			}
		}
	}()
	for i := 0; i < 30 && !r; i++ {
		if _, err := conn.Write(s.flag); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 100)
	}

	if r {
		return errors.New("穿透成功: " + conn.LocalAddr().String() + ", " + conn.RemoteAddr().String())
	} else {
		return errors.New("穿透失败")
	}
}

func (s *client) Action30() error {

	var connCh chan *net.UDPConn = make(chan *net.UDPConn)

	var end bool = false
	go func() { // 新建conn
		var lPort = s.cp1
		for !end && lPort-s.cp1 <= 255 {
			if conn, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: lPort}, s.raddr); err == nil {
				connCh <- conn
			}
			lPort++
			time.Sleep(time.Millisecond * 100)
		}
	}()

	var re chan *net.UDPConn = make(chan *net.UDPConn)
	var heat []*net.UDPConn = make([]*net.UDPConn, 0, 256)
	go func() {
		for v := range connCh {
			var g = v
			heat = append(heat, g)
			go func() {
				var da []byte = make([]byte, 64)
				for !end {
					if n, err := g.Read(da); err == nil {
						if n >= 17 && bytes.Equal(da[:17], s.flag) {
							re <- g
							end = true
							return
						}
					}
				}
			}()
		}
	}()

	go func() {
		for i := 0; i < len(heat); i++ {
			heat[i].Write(s.flag)
			time.Sleep(time.Millisecond * 20)
		}
	}()

	var err error
	select {
	case v := <-re:
		err = errors.New("穿透成功: " + v.LocalAddr().String() + ", " + v.RemoteAddr().String())
	case <-time.After(time.Second * 5):
		err = errors.New("穿透失败")
	}

	for i := 0; i < len(heat); i++ {
		heat[i].Close()
	}
	return err
}

func (s *client) Action40() error {

	var connCh chan *net.UDPConn = make(chan *net.UDPConn)

	var end bool = false
	go func() { // 新建conn
		var lPort = s.cp1
		for !end && lPort-s.cp1 <= 255 {
			if conn, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: lPort}, &net.UDPAddr{IP: s.raddr.IP, Port: s.raddr.Port + 1}); err == nil {
				connCh <- conn
			}
			lPort++
			time.Sleep(time.Millisecond * 100)
		}
	}()

	var re chan *net.UDPConn = make(chan *net.UDPConn)
	var heat []*net.UDPConn = make([]*net.UDPConn, 0, 256)
	go func() {
		for v := range connCh {
			var g = v
			heat = append(heat, g)
			go func() {
				var da []byte = make([]byte, 64)
				for !end {
					if n, err := g.Read(da); err == nil {
						if n >= 17 && bytes.Equal(da[:17], s.flag) {
							re <- g
							end = true
							return
						}
					}
				}
			}()
		}
	}()

	go func() {
		for i := 0; i < len(heat); i++ {
			heat[i].Write(s.flag)
			time.Sleep(time.Millisecond * 20)
		}
	}()

	var err error
	select {
	case v := <-re:
		err = errors.New("穿透成功: " + v.LocalAddr().String() + ", " + v.RemoteAddr().String())
	case <-time.After(time.Second * 5):
		err = errors.New("穿透失败")
	}

	for i := 0; i < len(heat); i++ {
		heat[i].Close()
	}
	return err
}

/* ---------------------------------------------- */

func (s *client) Action110() error {
	return s.Action10()
}

func (s *client) Action120() error {
	return s.Action20()
}

func (s *client) Action130() error {
	time.Sleep(time.Second)

	var rPort int
	for rPort = com.RandPort(); rPort == s.raddr.Port; {
	}

	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1}, &net.UDPAddr{IP: s.raddr.IP, Port: rPort}); err != nil {
		return err
	}
	defer conn.Close()

	conn.Write(s.flag)

	var da []byte = make([]byte, 60)
	var r bool = false
	go func() {
		for {
			if n, err := conn.Read(da); err == nil {
				if n >= 17 && bytes.Equal(da[:17], s.flag) {
					r = true
					return
				}
			}
		}
	}()
	time.Sleep(time.Second)

	if r {
		return errors.New("穿透成功: " + conn.LocalAddr().String() + ", " + conn.RemoteAddr().String())
	} else {
		return errors.New("穿透失败")
	}
}

func (s *client) Action140() error {
	var connCh chan *net.UDPConn = make(chan *net.UDPConn)

	var end bool = false
	go func() { // 新建conn
		for i := 0; i <= 255; i++ {
			if conn, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1}, &net.UDPAddr{IP: s.raddr.IP, Port: com.RandPort()}); err == nil {
				connCh <- conn
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	var re chan *net.UDPConn = make(chan *net.UDPConn)
	var heat []*net.UDPConn = make([]*net.UDPConn, 0, 256)
	go func() {
		for v := range connCh {
			var g = v
			heat = append(heat, g)
			go func() {
				var da []byte = make([]byte, 64)
				for !end {
					if n, err := g.Read(da); err == nil {
						if n >= 17 && bytes.Equal(da[:17], s.flag) {
							re <- g
							end = true
							return
						}
					}
				}
			}()
		}
	}()

	go func() {
		for i := 0; i < len(heat); i++ {
			heat[i].Write(s.flag)
			time.Sleep(time.Millisecond * 20)
		}
	}()

	var err error
	select {
	case v := <-re:
		err = errors.New("穿透成功: " + v.LocalAddr().String() + ", " + v.RemoteAddr().String())
	case <-time.After(time.Second * 5):
		err = errors.New("穿透失败")
	}

	for i := 0; i < len(heat); i++ {
		heat[i].Close()
	}
	return err
}

func (s *client) Action150() error {
	time.Sleep(time.Second)

	var connCh chan *net.UDPConn = make(chan *net.UDPConn)

	var end bool = false
	go func() { // 新建conn
		for i := 0; i <= 255; i++ {
			if conn, err := net.DialUDP("udp", &net.UDPAddr{IP: nil, Port: s.cp1}, &net.UDPAddr{IP: s.raddr.IP, Port: com.RandPort()}); err == nil {
				connCh <- conn
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	var re chan *net.UDPConn = make(chan *net.UDPConn)
	var heat []*net.UDPConn = make([]*net.UDPConn, 0, 256)
	go func() {
		for v := range connCh {
			var g = v
			heat = append(heat, g)
			go func() {
				var da []byte = make([]byte, 64)
				for !end {
					if n, err := g.Read(da); err == nil {
						if n >= 17 && bytes.Equal(da[:17], s.flag) {
							re <- g
							end = true
							return
						}
					}
				}
			}()
		}
	}()

	go func() {
		for i := 0; i < len(heat); i++ {
			heat[i].Write(s.flag)
			time.Sleep(time.Millisecond * 20)
		}
	}()

	var err error
	select {
	case v := <-re:
		err = errors.New("穿透成功: " + v.LocalAddr().String() + ", " + v.RemoteAddr().String())
	case <-time.After(time.Second * 5):
		err = errors.New("穿透失败")
	}

	for i := 0; i < len(heat); i++ {
		heat[i].Close()
	}
	return err
}
