package poll

import "syscall"

var CloseFunc func(syscall.Handle) error = syscall.Closesocket
