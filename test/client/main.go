package main

import (
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func main() {
	go func() {
		fd := dial()
		send(fd, net.UDPAddr{IP: net.ParseIP("172.29.92.248"), Port: 19986})
	}()

	go func() {
		fd := dial()
		send(fd, net.UDPAddr{IP: net.ParseIP("172.29.92.248"), Port: 19986})
	}()

	go func() {
		fd := dial()
		send(fd, net.UDPAddr{IP: net.ParseIP("172.29.92.248"), Port: 19986})
	}()
	time.Sleep(time.Hour)
}

func dial() windows.Handle {
	var wsaData windows.WSAData
	err := windows.WSAStartup(2<<16+2, &wsaData)
	if err != nil {
		panic(err)
	}

	fd, err := windows.Socket(windows.AF_INET, windows.SOCK_DGRAM, windows.IPPROTO_UDP) //windows.IPPROTO_ICMP
	if err != nil {
		panic(err)
	}

	err = windows.SetsockoptInt(windows.Handle((uintptr)(unsafe.Pointer(fd))), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	if err != nil {
		panic(err)
	}
	return fd
}

func send(fd windows.Handle, raddr net.UDPAddr) {
	defer func() {
		windows.Closesocket(fd)
		windows.WSACleanup()
	}()

	var rAddr windows.RawSockaddrInet4
	rAddr.Family = windows.AF_INET
	rAddr.Port = 19986
	ips := [4]byte{raddr.IP[12], raddr.IP[13], raddr.IP[14], raddr.IP[15]}
	rAddr.Addr = ips
	q := (*windows.RawSockaddrAny)(unsafe.Pointer(&rAddr))
	sAddr, err := q.Sockaddr()
	if err != nil {
		panic(err)
	}

	for {
		err = windows.Sendto(fd, da, 0, sAddr) //d是完整的传输层数据包
		if err != nil {
			panic(err)
		}
	}

}

var da []byte = []byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
