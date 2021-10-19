package net

type rawConn struct {
	fd *netFD
}

func newRawConn(fd *netFD) (*rawConn, error) {
	return &rawConn{fd: fd}, nil
}
