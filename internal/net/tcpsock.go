package net

import "syscall"

type TCPAddr struct {
	IP   IP
	Port int
	Zone string // IPv6 scoped addressing zone
}

func (a *TCPAddr) sockaddr(family int) (syscall.Sockaddr, error) {
	if a == nil {
		return nil, nil
	}
	return ipToSockaddr(family, a.IP, a.Port, a.Zone)
}

func ipToSockaddr(family int, ip IP, port int, zone string) (syscall.Sockaddr, error) {
	switch family {
	case syscall.AF_INET:
		if len(ip) == 0 {
			ip = IPv4zero
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return nil, &AddrError{Err: "non-IPv4 address", Addr: ip.String()}
		}
		sa := &syscall.SockaddrInet4{Port: port}
		copy(sa.Addr[:], ip4)
		return sa, nil
	}
	return nil, &AddrError{Err: "invalid address family", Addr: ip.String()}
}
