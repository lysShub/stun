package net

import (
	"context"
	"runtime"
	"syscall"
)

func internetSocket(ctx context.Context, net string, laddr, raddr sockaddr, sotype, proto int, mode string, ctrlFn func(string, string, syscall.RawConn) error) (fd *netFD, err error) {
	if (runtime.GOOS == "aix" || runtime.GOOS == "windows" || runtime.GOOS == "openbsd") && mode == "dial" && raddr.isWildcard() {
		raddr = raddr.toLocal(net)
	}
	family, ipv6only := favoriteAddrFamily(net, laddr, raddr, mode)
	return socket(ctx, net, family, sotype, proto, ipv6only, laddr, raddr, ctrlFn)
}

func favoriteAddrFamily(network string, laddr, raddr sockaddr, mode string) (family int, ipv6only bool) {
	switch network[len(network)-1] {
	case '4':
		return syscall.AF_INET, false
	case '6':
		return syscall.AF_INET6, true
	}

	if mode == "listen" && (laddr == nil || laddr.isWildcard()) {
		if supportsIPv4map() || !supportsIPv4() {
			return syscall.AF_INET6, false
		}
		if laddr == nil {
			return syscall.AF_INET, false
		}
		return laddr.family(), false
	}

	if (laddr == nil || laddr.family() == syscall.AF_INET) &&
		(raddr == nil || raddr.family() == syscall.AF_INET) {
		return syscall.AF_INET, false
	}
	return syscall.AF_INET6, false
}
