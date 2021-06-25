package com

import (
	"bytes"
	"crypto/rand"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/lysShub/e"
	uuid "github.com/satori/go.uuid"
)

var err error

func Errlog(err ...error) bool {
	var haveErr bool = false
	for i, e := range err {
		if e != nil {
			haveErr = true
			_, fp, ln, _ := runtime.Caller(1) //行数

			writers := []io.Writer{
				// errLogHandle, // *os.File
				os.Stdout, //标准输出，最后编译时可以删除
			}
			logger := log.New(io.MultiWriter(writers...), "", log.Ldate|log.Ltime) //|log.Lshortfile
			logger.Println(fp + ":" + strconv.Itoa(ln) + "." + strconv.Itoa(i+1) + "==>" + e.Error())
		}
	}
	return haveErr
}

// CreateUUID 生成id
// 16字节
func CreateUUID() []byte {
	var r []byte
	for {
		r = uuid.Must(uuid.NewV4(), nil).Bytes()
		if !bytes.Contains(r, []byte("`")) {
			return r
		}
	}
}

// GetLocalIP Get LAN IPv4
func GetLocalIP() (net.IP, error) {
	con, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.ParseIP("114.114.114.114"), Port: 443})
	if err != nil {
		return nil, err
	}
	defer con.Close()
	return net.ParseIP(strings.Split(con.LocalAddr().String(), ":")[0]), nil
}

func RandPort() int {
	b := new(big.Int).SetInt64(int64(52000))
	i, err := rand.Int(rand.Reader, b)
	r := int(i.Int64()) + 100
	if e.Errlog(err) {
		return 52942
	}
	return r
}

// 选出出现最多的值
func selectMost(l []int) int {
	var m map[int]int = make(map[int]int)
	for _, v := range l {
		m[v] = m[v] + 1
	}
	var c, r int = 0, 0
	for k, v := range m {
		if v > c {
			c = v
			r = k
		}
	}
	return r
}

func domainToIP(sever string) (string, error) {
	if r := net.ParseIP(sever); r == nil { //可能是域名
		var ips []net.IP
		if ips, err = net.LookupIP(sever); err != nil {
			return "", err
		}
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String(), nil
			}
		}
	}
	return sever, nil
}

func IsLanIP(ip net.IP) bool {
	// (a==10 || (a==172 && (b>=16 && b<32)) || (a==192 && b==168))
	if ip[12] == 10 || (ip[12] == 172 && (ip[13] >= 16 && ip[13] < 32)) || (ip[12] == 192 && ip[13] == 168) {
		return true
	} else {
		return false
	}
}
