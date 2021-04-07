

# STUN

​		UDP穿隧；实现了[rfc3489](https://tools.ietf.org/html/rfc3489)的功能，但并没有遵循其协议。

​		本仓库包含两个主要功能：NAT类型判断和NAT穿隧。都需要Sever支持，由于没有遵循标准rfc协议，所以服务器需要自己搭建。

​		功能的实现基于几种NAT其本身的性质。

### NAT类型

```php
网络地址转换（英语：Network Address Translation，缩写：NAT；又称网络掩蔽、IP掩蔽）在计算机网络中是一种在IP数据包通过路由器或防火墙时重写来源IP地址或目的IP地址的技术。这种技术被普遍使用在有多台主机但只通过一个公有IP地址访问互联网的私有网络中。它是一个方便且得到了广泛应用的技术。当然，NAT也让主机之间的通信变得复杂，导致了通信效率的降低。--wikipedia
```

​		严格意义上，NAT网关分为：静态NAT、动态NAT、PAT(端口多路复用)，PAT和NAPT(网络地址端口转换)相似。我们的对象是NAPT，这样的话、如果NAPT能完成穿隧，那么动态NAT和静态NAT也能实现。以下NAPT成为NAT。

​		NAT分类与性质：

| name                              | directions                                                   |
| --------------------------------- | ------------------------------------------------------------ |
| 完全锥形(Full Cone)               | 映射建立后，NAT会接收并转发所有数据包到映射对应的内网机器上  |
| IP限制锥形(Restricted Cone)       | 映射建立后，NAT只会接收并转发来自指定IP的数据包，这个指定IP在映射创建时确定 |
| 端口限制锥形(PortRestricted Cone) | 映射建立后，NAT只会接收并转发来自指定IP且指定端口的数据包，即来自指定地址的数据包 |
| 对称形(Symmetric NAT)             | 对”来“的数据包具有端口限制锥形的规则同时；如果”去“的数据包的四元组发生改变，还将会创建新的映射 |

说明：

- 映射的建立只能是由内网机器请求公网IP时在NAT网关创建。四元组即通信双方的IP和端口。
- IP限制锥形大多时称为限制锥形(Restricted Cone)，称其为IP限制锥形更明了。

### NAT类型判断

​		设计中我们可以不对完全锥形和IP限制锥形进行区分，因为区分需要两个公网IP、不很方便；如果需要区分，建议在一台VPS上绑定2张网卡。

​		client和sever都需要两个端口，称为第一端口和第二端口，下表中选取19987作为第一端口、19988作为第二端口。当前版本需要clietn和sever的第一第二端口相同，下一个版本将进行改进。

| 序号                       | 发送者       | 接收者       | 数据    | 说明                                                         |
| -------------------------- | ------------ | ------------ | ------- | ------------------------------------------------------------ |
| 1                          | client:19987 | sever:19987  | Juuid:1 | 开始，sever应保存Juuid、对方网关端口                         |
| 2                          | sever:19987  | client:19987 | Juuid:2 | sever回复client，client接受到2后将执行3，没有接收到执行2     |
| 3                          | client:19988 | sever:19987  | Juuid:3 | client使用的第二端口回复sever, sever比较两次(1和3)请求的网关端口是否相等。相等需要进一步判断(是锥形NAT，4)。不相等则有对称形NAT和公网IP两种情况；如果两次请求的网关端口和设定端口相同为公网IP(此例中第一次client网关端口为19987、第二次为19988)，否则为对称NAT(d)。有一定的误判率。 |
| 4                          | sever:19988  | client:19987 | Juuid:4 | sever使用第二端口进行回复，client不能收到则表示为端口限制形NAT(c)，否则为完全或IP限制锥形NAT(6) |
| 5                          | sever:19987  | client:19987 | Juuid:5 | 表示服务器执行了4                                            |
| <font color='red'>6</font> | client:19987 | sever:19987  | Juuid:6 | 收到5且收到4，为完全或IP限制锥形NAT，如须进一步区分、执行7,8；否则返回6 |
| 7                          | sever2:19987 | client:19987 | Juuid:7 | 可选，sever使用第二IP回复client                              |
| 8                          | sever:19987  | client:19987 | Juuid:8 | 可选，表示服务器执行了7，接下来执行a,b                       |
| <font color='red'>9</font> | sever:19987  | client:19987 | Juuid:9 | 公网IP                                                       |
| <font color='red'>a</font> | client:19987 | sever:19987  | Juuid:a | client收到8且收到7，完全锥形nat                              |
| <font color='red'>b</font> | client:19987 | sever:19987  | Juuid:b | client收到8且没有收到7，IP限制形nat                          |
| <font color='red'>c</font> | client:19987 | sever:19987  | Juuid:c | client收到5但没有收到4，端口限制nat                          |
| <font color='red'>d</font> | sever:19987  | client:19987 | Juuid:d | 对称形nat                                                    |
| <font color='red'>e</font> | client:19987 | sever:19987  | Juuid:e | 可能服务器关闭                                               |
| <font color='red'>f</font> | client:19987 | sever:19987  | Juuid:e | 异常情况                                                     |

说明：

- 红色即是可能返回值。
- 由于UDP不可靠，实际程序实现中同一数据包被发送多次。
- 数据中Juuid是16字节的唯一ID，`:`实际不存在，只是为了便于观看；最后字节数据序号(16进制)



