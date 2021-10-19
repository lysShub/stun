package net

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

var socketFunc func(int, int, int) (syscall.Handle, error)

func sysSocket(family, sotype, proto int) (syscall.Handle, error) {
	s, err := wsaSocketFunc(int32(family), int32(sotype), int32(proto),
		nil, 0, windows.WSA_FLAG_OVERLAPPED|windows.WSA_FLAG_NO_HANDLE_INHERIT)
	if err == nil {
		return s, nil
	}
	// WSA_FLAG_NO_HANDLE_INHERIT flag is not supported on some
	// old versions of Windows, see
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms742212(v=vs.85).aspx
	// for details. Just use syscall.Socket, if windows.WSASocket failed.

	// See ../syscall/exec_unix.go for description of ForkLock.
	syscall.ForkLock.RLock()
	s, err = socketFunc(family, sotype, proto)
	if err == nil {
		syscall.CloseOnExec(s)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return syscall.InvalidHandle, os.NewSyscallError("socket", err)
	}
	return s, nil
}

func setDefaultListenerSockopts(s syscall.Handle) error {
	// Windows will reuse recently-used addresses by default.
	// SO_REUSEADDR should not be used here, as it allows
	// a socket to forcibly bind to a port in use by another socket.
	// This could lead to a non-deterministic behavior, where
	// connection requests over the port cannot be guaranteed
	// to be handled by the correct socket.
	return nil
}
