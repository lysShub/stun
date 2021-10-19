package net

import (
	"context"
	"errors"
)

func DialUDP(network string, laddr, raddr *UDPAddr) (*UDPConn, error) {

	switch network {
	case "udp", "udp4", "udp6":
	default:
		return nil, &OpError{Op: "dial", Net: network, Source: laddr, Addr: raddr, Err: errors.New(network)}
	}
	if raddr == nil {
		return nil, &OpError{Op: "dial", Net: network, Source: laddr, Addr: nil, Err: errors.New("missing address")}
	}
	sd := &sysDialer{network: network, address: raddr.String()}
	c, err := sd.dialUDP(context.Background(), laddr, raddr)
	if err != nil {
		return nil, &OpError{Op: "dial", Net: network, Source: laddr, Addr: raddr, Err: err}
	}
	return c, nil
}

func newUDPConn(fd *netFD) *UDPConn { return &UDPConn{conn{fd}} }

type UDPConn struct {
	conn
}

type UDPAddr struct {
	IP   IP
	Port int
	Zone string // IPv6 scoped addressing zone
}

func (a *UDPAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	ip := ipEmptyString(a.IP)
	if a.Zone != "" {
		return JoinHostPort(ip+"%"+a.Zone, itoa.Itoa(a.Port))
	}
	return JoinHostPort(ip, itoa.Itoa(a.Port))
}

func (a *UDPAddr) isWildcard() bool {
	if a == nil || a.IP == nil {
		return true
	}
	return a.IP.IsUnspecified()
}

func (a *UDPAddr) opAddr() Addr {
	if a == nil {
		return nil
	}
	return a
}
