

# STUN

​		UDP穿隧；虽与rfc中的STUN同名，但不是对其的实现。

​		功能的实现基于几种NAT其本身的性质。

### NAT类型

​		NAT分为静态与动态，本文指动态NAT；可分为：

| 名称                              | 映射（映射建立时，）                    | 动作（映射建立后，）                                         |
| --------------------------------- | --------------------------------------- | ------------------------------------------------------------ |
| 完全锥形(Full Cone)               | 请求任何目的地，NAT网关分配同一地址     | NAT会接收所有数据包                                          |
| IP限制锥形(Restricted Cone)       | (与完全锥形相同)                        | NAT只会接收来自指定IP的数据包，这个指定IP是映射创建时的目的地IP |
| 端口限制锥形(PortRestricted Cone) | (与完全锥形相同)                        | NAT只会接收来自指定IP且指定端口的数据包，这个指定地址也是映射创建时的目的地地址 |
| 对称形(Symmetric NAT)             | 四元组有任何不同，NAT网关分配不同的地址 | （与端口限制锥形相同）                                       |

说明：IP限制锥形大多时称为限制锥形(Restricted Cone)，我称其为IP限制锥形。



<font color="red">--------------------------------------------------------------------分割线-------------------------------------------------------------------------------------------</font>

NAT类型判断流程。使用资源：

​		sever：两个IP（sever1、sever2），两个端口（S1、S2）（sever1:S1、sever1:S2、sever2:S1）

​		client：一个IP（client，网卡内网IP），两个端口（C1、C2）（client:C1、client:C2）

| 序号                         | 发送者    | 接收者    | 数据        | 说明                                                         |
| ---------------------------- | --------- | --------- | ----------- | ------------------------------------------------------------ |
| <font color='red'>-1</font>  | ---       | ---       | ---         | 发生错误                                                     |
| <font color='red'>0</font>   | ---       | ---       | ---         | 服务器无回复，可能服务器宕机或无网络                         |
| 10                           | client:C1 | sever:S1  | Juuid:1:C1  | 开始、C1占用2字节为client使用端口；sever应保存Juuid、网关端口，及使用端口 |
| 20                           | sever:S1  | client:C1 | Juuid:2:ip2 | sever回复client，client接受到20后将执行30；没有接收到返回0。ip2是第二网卡的公网IP，占用4个字节 |
| 30                           | client:C2 | sever:S1  | Juuid:30    | client使请求sever第二端口S2, sever比较两次（流程10和30）请求的网关端口是否相等：相等为锥形NAT需要进一步判断(40、50)。不相等则有对称形NAT和公网IP两种情况：如果两次请求的网关端口分别和使用端口(C1、C2)相同则为公网IP(90)；否则为对称NAT，如果两次请求的网关端口范围大于5则无序对称NAT(250)、否则顺序对称NAT(110)。 |
| 40                           | sever:S3  | client:C1 | Juuid:40    | sever使用第三端口进行回复，client不能收到则表示为端口限制形NAT(220)，否则为完全锥形或IP限制锥形NAT(60) |
| 50                           | sever:P1  | client:C1 | Juuid:500   | 表示服务器执行了40                                           |
| 60                           | client:C1 | sever:S1  | Juuid:60    | 客户端收到40后的回复，为完全或IP限制锥形NAT；执行70、80      |
| 70                           | sever2:S1 | client:C1 | Juuid:70    | sever使用第二IP回复client，如果client能收到则完全锥形(200)，否则IP限制锥形(210) |
| 80                           | sever:S1  | client:C1 | Juuid:80    | 表示服务器执行了70                                           |
| 90                           | sever2:S1 | client:C1 | Juuid:90    | sever使用第二IP回复client，如果client能收到此数据包，则公网IP(180)，否则具有防火墙的公网IP(190) |
| 100                          | sever:S1  | client:C1 | Juuid:100   | 表示服务器执行了90                                           |
| 110                          | sever:S1  | client:C1 | Juuid:110   | 服务器告知客户端执行120                                      |
| 120                          | client:C1 | sever2:S1 | Juuid:120   | 服务器收到此数据包后；判断IP和10的网关IP是否相同，如果不相同250；否则继续判断和10的网关端口是否相等或相连，相等为IP锥形顺序对称形(237)，相连则完全顺序对称NAT(230)，否则IP限制顺序对称NAT(240)。 |
|                              |           |           |             |                                                              |
| <font color='red'>180</font> | client:C1 | sever:S1  | Juuid:180   | 公网IP                                                       |
| <font color='red'>190</font> | client:C1 | sever:S1  | Juuid:190   | 具有防火墙的公网IP                                           |
| <font color='red'>200</font> | client:C1 | sever:S1  | Juuid:200   | 完全锥形NAT                                                  |
| <font color='red'>210</font> | client:C1 | sever:S1  | Juuid:210   | IP限制锥形NAT                                                |
| <font color='red'>220</font> | client:C1 | sever:S1  | Juuid:220   | 端口限制锥形NAT                                              |
| <font color='red'>230</font> | sever:S1  | client:C1 | Juuid:230   | 完全顺序对称NAT                                              |
| <font color='red'>237</font> | sever:S1  | client:C1 | Juuid:237   | IP锥形顺序对称NAT                                            |
| <font color='red'>240</font> | sever:S1  | client:C1 | Juuid:240   | IP限制顺序对称NAT                                            |
| <font color='red'>25?</font> | sever:S1  | client:C1 | Juuid:250   | 无序对称NAT（250：同IP随机端口，251：随机IP随机端口）        |

1. 关于对称NAT
   - **顺序对称NAT**：对于新建对称NAT映射，无论本地地址是多少，分配的NAT网关的IP是相同的，且端口有相邻的规律。

     

       - <font style="font-size:small;">**完全顺序对称NAT**</font>：新建顺序对称NAT映射时，NAT网关分配相邻的端口

         

       - <font style="font-size:small;">**IP锥形顺序对称NAT**</font>：四元组中raddr.IP不变时, 新建映射分配的NAT端口为a、a+1、a+2... ; 此时如果新建raddr.IP改变时, 分配的NAT端口为a

         

       - <font style="font-size:small">**IP限制顺序对称NAT**</font>：新建顺序对称NAT映射时；只有请求的目的地IP相同，那么NAT网关分配的端口才是相邻的；如果IP不同，那么NAT网关分配的端口是随机的。
       <font>&nbsp;</font>
   - **无序对称NAT**：对称NAT中除了顺序对称NAT外都是无序对称NAT。对于新建对称NAT映射，无论什么情况，新映射的NAT网关端口是随机的，甚至新映射的NAT网关的IP是随机的（此种网络常见于校园网，完全屏蔽了种子下载）。

​	2. 流程中有使用超时机制，在糟糕的网络环境下可能存在误判的概率。

<font color="red">--------------------------------------------------------------------分割线-------------------------------------------------------------------------------------------</font>



### NAT穿透

- 250与250组合无法进行穿隧

  

​	 基于以上对NAT类型的分类，将穿透组合分为2种：




   - 双方NAT中没有240及250的，A类组合
   - 双方NAT中存在240或250的，B类组合

B类组合更加复杂一些。

| 序号 | 发   | 收   | data                    | 说明                                                         |
| ---- | ---- | ---- | ----------------------- | ------------------------------------------------------------ |
| 10   | S和R | SV   | Tuuid:10:t:ep           | 双方Tuuid要相同、占用17个字节。10表示任务流程码，占用1个字节。t表示自己的NAT类型，占用一个字节。ep为泛端口长度，如果双方不同取其长，占用2个字节 |
| 20   | SV   | S和R | Tuuid:20:t:ep:rIP:rPort | 双方都执行完10后SV回复双方。t表示对方的NAT类型。ep是泛端口长度。rIP和rPort是对方的网关IP和主端口、IP占用4个字节，端口2个字节。 |
| 30   | S或R | R或S | Tuuid:30:               |                                                              |
| 30   | S和R | R和S | Tuuid:30                | 双方监听的同时向对方网关主端口的泛端口发送此数据包。当接某一方接收到对方的此数据包时停止发送并执行40；否则超时后返回、无法完成穿隧。 |
| 40   | S或R | R或S | Tuuid:40                | 接收到30后的回复，回复一定数量的40包后返回30包的raddr。另一方接收到此数据包立即停止发送30，并返回此数据包的raddr、穿隧成功 |

说明：

- ​	某端口的泛端口是指此端口及其相连的几个端口、如端口100的泛端口可以是：[100、101、102、103]，此时ep为4

**B类组合：**


