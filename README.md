# P2P Demo

基于 libp2p 的 P2P 通信与文件共享演示项目。

## 功能特性

- **P2P 节点创建** - 创建 libp2p 节点，支持 TCP 传输
- **节点发现** - 通过 UDP 广播发现网络中其他节点
- **节点连接** - 使用 Multiaddr 连接其他节点
- **消息通信** - P2P 节点间消息传递
- **文件共享** - 上传和下载文件到其他节点
- **Web 界面** - 提供可视化操作界面

## 技术栈

- Go
- libp2p
- Gin (Web 框架)

## 运行方式

```bash
go run main.go
```

服务启动后访问 `http://localhost:8080`

## API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/create-node` | POST | 创建 P2P 节点 |
| `/connect-node` | POST | 连接其他节点 |
| `/send-msg` | POST | 发送消息 |
| `/refresh-msg` | POST | 刷新消息 |
| `/upload-file` | POST | 上传文件 |
| `/refresh-file-list` | POST | 刷新文件列表 |
| `/exist-nodes` | POST | 查询在线节点 |
| `/download-file` | POST | 下载文件 |

## 项目结构

```
.
├── main.go          # 后端主代码
├── static/          # 前端静态文件
│   ├── index.html
│   ├── index.js
│   └── index.css
└── uploads/         # 文件存储目录
```