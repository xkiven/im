# IM 服务项目



这是一个使用go完成的即时通讯（IM）服务项目，整合了 gRPC、MySQL、Redis、MongoDB 和 Kafka 等多种技术，提供用户注册、登录、通讯、添加好友等基础功能。

*项目地址*：[xkiven/im 在主版](https://github.com/xkiven/im/tree/master)

## 项目结构



项目主要包含以下几个重要的模块：

- `internal/mysql`：负责与 MySQL 数据库交互，包含用户模型定义及 MySQL 客户端的实现。
- `internal/redis`：实现 Redis 客户端，用于缓存用户信息。
- `internal/rpc/user`：处理用户相关的 RPC 请求，如用户注册和登录。
- `internal/svc`：定义服务上下文结构体，管理项目中各种客户端实例。
- `internal/config`：负责加载项目配置文件。

## 配置文件



项目使用 YAML 格式的配置文件，主要配置项包括：

- `Name`：服务名称。
- `Host`和 `Port`：服务监听的地址和端口。
- `UserRpc`，`MessageRpc`，`FriendRpc`  ：RPC 客户端配置。
- `Kafka`：Kafka 相关配置，如 Brokers 和 Topic。
- `MongoDB`：MongoDB 的 URI 和 Database 名称。
- `MySQL`：MySQL 数据库的数据源。
- `Redis`：Redis 的主机地址和密码。

## 主要功能



### **用户注册**

- 用户注册时，系统会先检查用户名是否已存在，若不存在则将用户信息插入到 MySQL 数据库，并将用户信息写入 Redis 缓存。

### **用户登录**

- 用户登录时，系统会先尝试从 Redis 缓存中获取用户信息进行验证，若验证失败则从 MySQL 数据库中验证。验证通过后，系统会生成 JWT 令牌并返回给用户。

### **好友的建立**

- 系统会使用websocket协议，将好友申请即时传给客户端，并把好友的建立数据储存在MongoDB上

### **发送并即时接收消息**

- 用户发送消息时会先判断是否为好友，再通过Kafka消息队列持久化流程，将消息使用websocket协议即时发送并接收消息
