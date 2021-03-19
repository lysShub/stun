package stun

type STUN struct {
	FirstPort  uint16 // 第一端口，客户端和服务器设置要相同
	SecondPort uint16
}
