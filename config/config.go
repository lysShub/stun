package config

import "time"

// 泛端口长度, 端口长度不大于次值表示端口相连
const ExtPorts int = 5

// 非阻塞请求的超时时间
const TimeOut time.Duration = time.Second * 3

// 穿透匹配时间长度
const MatchPeriod time.Duration = time.Minute * 10

// UDP冗余发送次数, 为确保数据可达
const ResendTimes int = 5
