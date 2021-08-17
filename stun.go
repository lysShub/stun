package stun

import (
	"errors"
	"net"
	"time"

	"github.com/lysShub/mapdb"
)

//  无论客户端还是服务器都需要两个IP(IP1和IP2)。同一个VPS绑定两张网卡；这两张网卡的私网IP分别是a、b，公网IP分别是x，y。则在客户端IP1、IP2分别配置为x、y，在服务器IP1、IP2分别配置为a、b。

type STUN struct {
	Iterate   int           // 同数据包重复发送次数，确保UDP可靠，默认5
	MatchTime time.Duration // 匹配时长
	TimeOut   time.Duration // 超时时间
	ExtPorts  int           // 泛端口范围，默认7

	Port int // 端口，使用多个端口则依次递增
}

var err error
var errSever error = errors.New("sever no reply or network timeout")

// sconn Sever Conn
type sconn struct {
	STUN

	s1, s2     int          // 端口
	lip1, lip2 net.IP       // 内网第一IP, 内网第二IP
	wip2       net.IP       // 内网第二IP(第二网卡)对应的公网IP
	conn1      *net.UDPConn // 第一IP: 第一端口
	conn2      *net.UDPConn // 第一IP: 第二端口
	conn3      *net.UDPConn // 第二IP: 第一端口

	dbj *mapdb.Db // NAT类型判断的数据库
	dbt *mapdb.Db // NAT穿隧数据库
}

// cconn Client Conn
type cconn struct {
	STUN
	sever    net.IP // 服务器IP
	cp1, cp2 int    // 本地(客户端)使用端口, client port
	flag     []byte
	raddr    *net.UDPAddr // 对方地址
}
