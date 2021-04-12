# 【P2P overlay network】基于公网虚拟私网工具

本项目提供云网络服务，用于多地区组网技术，相当于将多个地域网络虚拟统一局域网，方便团队跨地域网络访问；提供统一控制面板进行管理，并且利用P2P技术提升节点近点加速；如果多个节点处于内网，那么会优先采用内网进行互通；并且支持多云转发管理；根据网络时延动态路由；

原理图：

![image](https://upload-images.jianshu.io/upload_images/6796036-4626f4fd53fc907e.png?imageMogr2/auto-orient/strip%7CimageView2/2/w/1240)

版本特性：

- 支持 windows & linux 客户端、服务端程序
- 支持手动网络配置，云转发；
- 支持多网络平面隔离；

软件下载地址：[https://github.com/easymesh/easymesh/releases/](https://github.com/easymesh/easymesh/releases/)

根据您所需要部署的形态决定；客户端、服务端没有绑定限制，支持windows & linux 混合部署使用；提供 gateway 和 transfer 两个可执行文件；gateway 属于客户端，transfer 属于服务端，该版本transfer还不支持分布式部署，部署transfer需要准备一个公网IP地址；

## 安装部署

### 1、准备工作：

部署gateway程序需要完成如下平台的检查和安装；

#### windows

*   安装 tap windows 虚拟网卡驱动； [win10](https://build.openvpn.net/downloads/releases/tap-windows-9.24.2-I601-Win10.exe) [win7](https://build.openvpn.net/downloads/releases/tap-windows-9.24.2-I601-Win7.exe)

#### linux

*   查看tun设备是否存在; `ls -ail /dev/net/tun`

### 2、启动transfer程序
需要提前准备一台linux/windows云主机，并且绑定固定公网IP地址，不一定需要绑定域名；以及两个本地linux/windows测试节点；

#### 启动命令：

```
Usage of transfer.exe:
  -bind int
        transfer server bind port (default 8000)
  -debug
        debug mode
  -help
        usage
  -log string
        log dir (default "./")
  -nums int
        transfer server instance nums (default 1000)
  -public string
        public IP (default "www.domain.com")
  -token string
        access auth
```

- -bind: 所需要绑定的UDP起始端口，注意转发transfer服务支持绑定多个端口，每个端口分别对于一个namespace，配合 -nums 参数，可以创建多个独立转发地址空间；默认端口范伟为：8000~9000，如果其中一个端口被占用，则会忽略并跳过该端口；
- -debug: 调试模式，所以日志将打印到控制台，不会输出到目录；方便问题定位；
- -log: 运行日志的目录地址；默认会记录30天运行日志，并且支持zip压缩；建议您保留大约1GB以上磁盘空间；
- -nums: 命名空间数量，也对应服务实例数量，与-bind结合使用，请保留相应端口范围；
- -public: 云服务主机对外IP或者域名；需要公网可以访问的IPv4地址；
- -token: 用于校验gateway接入的身份；如果为空，会自动生成一个随机字符串；例如："s^I^ghGjkB7Zm$q14NWhxfQdS5E&FG7R"

注意：使用方式不区分windows、linux平台，启动后保持后台长时间运行即可；

### 3、启动gateway程序
在客户端侧，可以连接外网访问云主机的节点都可以；

```
Usage of gateway.exe:
  -debug
        debug mode
  -help
        usage
  -iface string
        interface or ip (default "eth0")
  -ip string
        virtual ip (default "172.168.0.1")
  -log string
        log dir (default "./")
  -token string
        access auth
  -trans string
        transfer public address (default "www.domain.com:8000")
```

*   -debug: 调试模式，所以日志将打印到控制台，不会输出到目录；方便问题定位；
*   -ip: 在当前虚拟网络中的虚拟地址IP，目前支持IPv4地址，例如：`172.168.x.x`，默认`255.255.0.0`网段，注意：不能与自身其他网卡网段冲突；
*   -log: 运行日志的目录地址；默认会记录30天运行日志，并且支持zip压缩；建议您保留大约1GB以上磁盘空间；
*   -token: 用于登陆认证的token，需要和transfer的token保持一致；必须填写该字段；
*   -trans: 连接相应转发服务，就是对应transfer的公网IP地址和端口；如果选用一个端口，那么其他需要加入同一个网络namespace的节点，端口需要保持一致；
*   -iface: 绑定本地网卡名称或者IP地址，比如：在linux环境下面默认eth0，而windows相对复杂；可以通过 控制面板 -> 网络与共享中心 -> 更改适配器设置 里面进行查看；例如截图：[](https://github.com/easymesh/docs/blob/master/windows_eth.png) 对应名称为: `vEthernet (wlan)`或者查看IP地址方式，例如：linux 通过命令 `ifconfig` 查看相应IP地址，例如如下eth0对应的IP地址为：`192.168.3.2`

```
eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 192.168.3.2  netmask 255.255.255.0  broadcast 192.168.3.255
        inet6 fe80::215:5dff:fe03:b00  prefixlen 64  scopeid 0x20<link>
...
```

windows通过查看ipconfig命令查看地址：

```
以太网适配器 vEthernet (wlan):
连接特定的 DNS 后缀 . . . . . . . :
本地链接 IPv6 地址. . . . . . . . : fe80::5523:9866:1b83:5ad7%9
IPv4 地址 . . . . . . . . . . . . : 192.168.3.11
子网掩码  . . . . . . . . . . . . : 255.255.255.0
默认网关. . . . . . . . . . . . . : 192.168.3.1
```

注意：使用方式不区分windows、linux平台，启动后保持后台运行即可；

### 4、测试连通性：
两个部署gateway的节点互ping对方的虚拟IP地址；结果如下表示成功；

```
root@node2:~# ping 172.168.3.1
PING 172.168.3.1 (172.168.3.1) 56(84) bytes of data.
64 bytes from 172.168.3.1: icmp_seq=1 ttl=126 time=2.78 ms
64 bytes from 172.168.3.1: icmp_seq=2 ttl=126 time=3.65 ms
64 bytes from 172.168.3.1: icmp_seq=3 ttl=126 time=2.85 ms
```

### 5、部署实例

在公有云主机启动部署transfer程序：

参考命令：

linux：
```
./transfer -public you.domain.com -bind 8000 -nums 1000 -token "s^I^ghGjkB7Zm$q14NWhxfQdS5E&FG7R"
```

注意：默认需要开启8000-9000的UDP端口；

本地部署gateway程序；准备两台节点；

参考命令：

windows:

```
gateway.exe -ip 172.168.3.1 -trans you.domain.com:8000 -iface "192.168.1.110" -token "s^I^ghGjkB7Zm$q14NWhxfQdS5E&FG7R"
```

linux：

```
./gateway -ip 172.168.3.2 -trans you.domain.com:8000 -iface "eth0" -token "s^I^ghGjkB7Zm$q14NWhxfQdS5E&FG7R"
```

两个节点相互ping对方虚拟IP；

```
root@node2:~# ping 172.168.3.1
PING 172.168.3.1 (172.168.3.1) 56(84) bytes of data.
64 bytes from 172.168.3.1: icmp_seq=1 ttl=126 time=2.78 ms
64 bytes from 172.168.3.1: icmp_seq=2 ttl=126 time=3.65 ms
64 bytes from 172.168.3.1: icmp_seq=3 ttl=126 time=2.85 ms
```

注意：如果两个节点在同一个局域网，那么他们互ping时延也会很低，如果多个地域，会自动走公网转发服务，所以时延会较长；

也可以通过 ipconfig、ifconfig 查看虚拟网卡IP地址信息；

```
mesh0: flags=4305<UP,POINTOPOINT,RUNNING,NOARP,MULTICAST>  mtu 1472
        inet 172.168.3.2  netmask 255.255.255.255  destination 172.168.3.2
...
```
