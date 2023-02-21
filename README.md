# GoCryptoTCP

### 一句话简介

`GoCryptoTCP` 是一个基于 `Golang` 的传输层加密解决方案。

### 提出动机

做一些小项目时，有一些加密传输的场景，比如说需要传输账号密码等敏感信息，这就需要加密。`HTTPS` 协议需要有域名，还需要申请证书。也有一些服务商，提供短期的 `IP` 证书，其他限制也很多。

如果在不申请证书的情况下，可以考虑在 `TCP` 或者 `HTTP` 上，先交换密钥再传输信息。考虑到 `TCP` 更加灵活和底层，所以选择基于 `TCP` 进行加密。

所以，`GoCryptoTCP` 主要想解决的是在传输层上的加密传输。

### 工作原理

在 `TCP` 的 `3` 次握手之后，`GoCryptoTCP` 会额外做 `1` 次握手，互相交换 `RSA Public Key`，商定 `AES KEY` 并用公钥加密后交换。后续的消息可以自由选用 `RSA` 或 `AES`加密。

`GoCryptoTCP` 向下封装 `TCP`的细节，向上提供发送数据流和报文的接口。

### 项目演示

![img](/clientEg/eg.png)

为了演示项目，采用 `C/S` 架构搭建了公共通讯服务器。下载 `Releases` 中的 `client.public.exe`运行即可。

每次连接都会随机分配到一个 `ID`，这个 `ID` 由一个 `ID Pool`来管理，保证 `o(1)` 时间高效分发不重复。

发送方法：

```shell
to [id] [msg]
```

向系统 `ID 1000` 发送消息，可以得到回文。

如果公共服务器不在线，您可以下载 `Releases` 中的 `client.exe` 和 `server.exe`。先运行 `server.exe`，后运行 `client.exe`。

也可以自行在本地编译搭建：

在 `clientEg` 文件夹下执行：
```shell
go build -o client.exe && client.exe
```
在 `serverEg` 文件夹下执行：
```shell
go build -o server.exe && server.exe
```

### 未来计划

1. 加入验签
2. 测试并发性能，并优化

### 其他

1. `GoCryptoTCP` 目前只是一个学生项目，并不能解决中间人攻击问题。
2. 360的云特征引擎可能误报病毒。项目保证无毒，您也可以自行查看源码，自行编译。