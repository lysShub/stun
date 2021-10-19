package net

import (
	"net"
	"syscall"
)

type sockaddr interface {
	net.Addr

	// family returns the platform-dependent address family
	// identifier.
	family() int

	// isWildcard reports whether the address is a wildcard
	// address.
	isWildcard() bool

	// sockaddr returns the address converted into a syscall
	// sockaddr type that implements syscall.Sockaddr
	// interface. It returns a nil interface when the address is
	// nil.
	sockaddr(family int) (syscall.Sockaddr, error)

	// toLocal maps the zero address to a local system address (127.0.0.1 or ::1)
	toLocal(net string) sockaddr
}
