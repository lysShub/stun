package net

import (
	"context"
	"net"
	"syscall"
)

func (sd *sysDialer) dialUDP(ctx context.Context, laddr, raddr *UDPAddr) (*net.UDPConn, error) {
	fd, err := internetSocket(ctx, sd.network, laddr, raddr, syscall.SOCK_DGRAM, 0, "dial", sd.Dialer.Control)
	if err != nil {
		return nil, err
	}

	// err = windows.SetsockoptInt(windows.Handle((uintptr)(unsafe.Pointer(fd))), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	// if err != nil {
	// 	return nil, err
	// }

	return newUDPConn(fd), nil
}
