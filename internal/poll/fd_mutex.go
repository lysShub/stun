package poll

type fdMutex struct {
	state uint64
	rsema uint32
	wsema uint32
}
