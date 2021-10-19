package net

import (
	"syscall"

	"stun/internal/windows"
)

var wsaSocketFunc func(int32, int32, int32, *syscall.WSAProtocolInfo, uint32, uint32) (syscall.Handle, error) = windows.WSASocket
var listenFunc func(syscall.Handle, int) error
