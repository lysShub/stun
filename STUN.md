

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

​		NAT类型判断流程。

| 序号                        | 发送者    | 接收者    | 数据       | 说明                                                         |
| --------------------------- | --------- | --------- | ---------- | ------------------------------------------------------------ |
| <font color='red'>-1</font> | ---       | ---       | ---        | 发生错误                                                     |
| <font color='red'>0</font>  | ---       | ---       | ---        | 服务器无回复，可能服务器宕机或无网络                         |
| 1                           | client:c1 | sever:s1  | Juuid:1:c1 | 开始、c1占用2字节；sever应保存Juuid、网关端口，及使用端口    |
| 2                           | sever:s1  | client:c1 | Juuid:2    | sever回复client，client接受到2后将执行3；没有接收到返回0     |
| 3                           | client:c2 | sever:s1  | Juuid:3:c2 | client使用的第二端口请求sever, sever比较两次(流程1和3)请求的网关端口是否相等。相等需要进一步判断(锥形NAT；4、5)。不相等则有对称形NAT和公网IP两种情况；如果两次请求的网关端口分别和使用端口(c1、c2)相同为公网IP(9)，否则为对称NAT；如果两次网关端口相邻则为顺序对称NAT(e)，否则为无序对称NAT(f)。有一定的误判率。 |
| 4                           | sever:s2  | client:c1 | Juuid:4    | sever使用第二端口进行回复，client不能收到则表示为端口限制形NAT(d)，否则为完全锥形或IP限制锥形NAT(6) |
| 5                           | sever:s1  | client:c1 | Juuid:5    | 表示服务器执行了4                                            |
| 6                           | client:c1 | sever:s1  | Juuid:6    | 客户端回复，为完全或IP限制锥形NAT，如须进一步区分、执行7,8；否则c |
| 7                           | sever2:s1 | client:c1 | Juuid:7    | sever使用第二IP回复client，如果client能收到则a，否则b        |
| 8                           | sever:s1  | client:c1 | Juuid:8    | 表示服务器执行了7                                            |
| <font color='red'>9</font>  | sever:s1  | client:c1 | Juuid:9    | 公网IP                                                       |
| <font color='red'>a</font>  | client:c1 | sever:s1  | Juuid:a    | client收到7，完全锥形nat                                     |
| <font color='red'>b</font>  | client:c1 | sever:s1  | Juuid:b    | client收到8且没有收到7，IP限制形nat                          |
| <font color='red'>c</font>  | sever:s1  | client:c1 | Juuid:c    | 完全锥形或IP限制锥形NAT                                      |
| <font color='red'>d</font>  | client:c1 | sever:s1  | Juuid:d    | client收到5但没有收到4，端口限制nat                          |
| <font color='red'>e</font>  | sever:s1  | client:c1 | Juuid:f    | 顺序对称形NAT                                                |
| <font color='red'>f</font>  | sever:s1  | client:c1 | Juuid:f    | 无序对称NAT                                                  |



说明：

- s1,s2是服务器的第一第二使用端口，c1,c2是客户端的第一第二使用端口。要求1和2是相连端口(1小)，但不要求s和c相同。

- 红色代码表示可能返回值，表中以16进制表示。

- 由于区分完全锥形和IP限制锥形需要不同的IP，稍有麻烦，这里是否将其加以区分的权力开放。在实际程序是使用第二网卡实现的、只需提供第二网卡的IP即可。

- 5和8的作用类似TCP中的ACK，当client接收到它们后，如果在之后无法接收到它们对应的数据则判定其对应数据不可达。

- 顺序NAT和无序NAT都是NAT类型网关；顺序NAT网关下，新建立的映射的网关的端口是连续的。实现中、两者距离小于7则判断为顺序NAT。

  

<font color="red">--------------------------------------------------------------------分割线-------------------------------------------------------------------------------------------</font>

NAT类型判断流程。

| 序号                        | 发送者    | 接收者    | 数据       | 说明                                                         |
| --------------------------- | --------- | --------- | ---------- | ------------------------------------------------------------ |
| <font color='red'>-1</font> | ---       | ---       | ---        | 发生错误                                                     |
| <font color='red'>0</font>  | ---       | ---       | ---        | 服务器无回复，可能服务器宕机或无网络                         |
| 1                           | client:c1 | sever:s1  | Juuid:1:c1 | 开始、c1占用2字节；sever应保存Juuid、网关端口，及使用端口    |
| 2                           | sever:s1  | client:c1 | Juuid:2    | sever回复client，client接受到2后将执行3；没有接收到返回0     |
| 3                           | client:c2 | sever:s1  | Juuid:3:c2 | client使用的第二端口请求sever, sever比较两次(流程1和3)请求的网关端口是否相等。相等需要进一步判断(锥形NAT；4、5)。不相等则有对称形NAT和公网IP两种情况；如果两次请求的网关端口分别和使用端口(c1、c2)相同可能为公网IP(9)，否则为对称NAT()。 |
| 4                           | sever:s2  | client:c1 | Juuid:4    | sever使用第二端口进行回复，client不能收到则表示为端口限制形NAT(d)，否则为完全锥形或IP限制锥形NAT(6) |
| 5                           | sever:s1  | client:c1 | Juuid:5    | 表示服务器执行了4                                            |
| 6                           | client:c1 | sever:s1  | Juuid:6    | 客户端回复，为完全或IP限制锥形NAT，如须进一步区分、执行7,8；否则c |
| 7                           | sever2:s1 | client:c1 | Juuid:7    | sever使用第二IP回复client，如果client能收到则a，否则b        |
| 8                           | sever:s1  | client:c1 | Juuid:8    | 表示服务器执行了7                                            |
| 9                           |           |           |            |                                                              |
|                             |           |           |            |                                                              |
|                             |           |           |            |                                                              |
| <font color='red'>9</font>  | sever:s1  | client:c1 | Juuid:9    | 公网IP                                                       |
| <font color='red'>a</font>  | client:c1 | sever:s1  | Juuid:a    | client收到7，完全锥形nat                                     |
| <font color='red'>b</font>  | client:c1 | sever:s1  | Juuid:b    | client收到8且没有收到7，IP限制形nat                          |
| <font color='red'>c</font>  | sever:s1  | client:c1 | Juuid:c    | 完全锥形或IP限制锥形NAT                                      |
| <font color='red'>d</font>  | client:c1 | sever:s1  | Juuid:d    | client收到5但没有收到4，端口限制nat                          |
| <font color='red'>e</font>  | sever:s1  | client:c1 | Juuid:f    | 顺序对称形NAT                                                |
| <font color='red'>f</font>  | sever:s1  | client:c1 | Juuid:f    | 无序对称NAT                                                  |





顺序对称NAT属于对称NAT的一种，具有对称NAT的特征，及四元组中任何一元的改变都会新建映射；如果这个新建映射有规律(端口)那么就有实现穿隧的可能。在研究实验中，我们找到一种规律，顺序；及在一定条件下，新建映射的NAT网关端口是连续的。我们定义顺序对称NAT：对于新建对称NAT映射，无论本地端口是什么，也无论目的地端口是什么；只要新映射的目的地IP相同，那么新映射的NAT网关端口是相连的。（少数情况可能有相隔几个端口的情况）

<font color="red">--------------------------------------------------------------------分割线-------------------------------------------------------------------------------------------</font>



### NAT穿透

​		以下流程能对所有可以穿隧的情况进行穿隧，是一种宏观、高屋建瓴式的设计。而且易于兼容针对无序对称NAT端口探测算法。

| 序号 | 发   | 收   | data                    | 说明                                                         |
| ---- | ---- | ---- | ----------------------- | ------------------------------------------------------------ |
| 10   | S和R | SV   | Tuuid:10:t:ep           | 双方Tuuid要相同、占用17个字节。10表示任务流程码，占用1个字节。t表示自己的NAT类型，占用一个字节。ep为泛端口长度，如果双方不同取其长，占用2个字节 |
| 20   | SV   | S和R | Tuuid:20:t:ep:rIP:rPort | 双方都执行完10后SV回复双方。t表示对方的NAT类型。ep是泛端口长度。rIP和rPort是对方的网关IP和主端口、IP占用4个字节，端口2个字节。 |
| 30   | S和R | R和S | Tuuid:30                | 双方监听的同时向对方网关主端口的泛端口发送此数据包。当接某一方接收到对方的此数据包时停止发送并执行40；否则超时后返回、无法完成穿隧。 |
| 40   | S或R | R或S | Tuuid:40                | 接收到30后的回复，回复一定数量的40包后返回30包的raddr。另一方接收到此数据包立即停止发送30，并返回此数据包的raddr、穿隧成功 |

说明：

- ​	某端口的泛端口是指此端口及其相连的几个端口、如端口100的泛端口可以是：[100、101、102、103]，此时p为4



