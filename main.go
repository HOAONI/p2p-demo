package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type receivedMsg struct {
	canRead bool
	who     string
	msg     string
}

type nodesBroadcast struct {
	NodeName string `json:"nodeName"`
	NodeAddr string `json:"nodeAddr"`
}

type filesAsk struct {
	AskForFiles bool `json:"askForFiles"`
}

type filesMessage struct {
	Address string   `json:"address"`
	Files   []string `json:"files"`
}
type whoAskFiles struct {
	IPAddress string `json:"IPAddress"`
	Port      string `json:"port"`
}

// 单个节点的文件信息
type nodeFileInfo struct {
	OwnerIP  string   `json:"ownerIP"`
	NodeName string   `json:"node_name"`
	Files    []string `json:"files"`
}

// 响应消息，包含多个节点的文件信息
type fileResponse struct {
	Nodes []nodeFileInfo `json:"nodes"`
}

var msgTemp = receivedMsg{canRead: false}

var nodes = make(map[string]host.Host)           // 用于存储创建的节点  (！！同名问题没有处理！！)
var knownNodes = make(map[string]string)         // 存储接收到的在线节点
var knownFiles = make(map[string][]nodeFileInfo) // 存储接收到的文件回应
var knownFilesLock sync.Mutex
var whichNode = ""

// 定义通信协议 ID
const protocolID = protocol.ID("/libp2p/example/1.0.0")
const ipAddr = "172.20.10.3"
const port = "8080"
const portFile = "9099"
const portListenFileResp = "9090"

func main() {

	ctx := context.Background()

	// 创建 Gin 路由
	r := gin.Default()

	r.Static("/", "./static")

	// 监听节点广播
	go broadcastListen()
	go listenFileReqAndResp()
	go collectFileResponses()

	// 定义创建节点的 API
	r.POST("/create-node", func(c *gin.Context) {
		var request struct {
			Name string `json:"name"` // 节点名称
		}

		// 解析 JSON 请求
		if err := c.ShouldBindJSON(&request); err != nil {
			// 如果 JSON 格式有误，返回 400 错误
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
			return
		}

		// 检查是否提供了名称
		if request.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "节点名称不能为空"})
			return
		}

		// 创建 Libp2p 节点
		node, err := newNode("/ip4/" + ipAddr + "/tcp/0")
		if err != nil {
			// 如果节点创建失败，返回 500 错误
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建节点失败"})
			return
		}

		// 设置流处理器，用于处理其他节点发来的消息
		node.SetStreamHandler(protocolID, handleStream)

		// 保存节点到全局 map
		nodes[request.Name] = node
		broadcast(node, request.Name)

		c.JSON(http.StatusOK, gin.H{
			"message": "节点创建成功",
			"name":    request.Name,
			"address": node.Addrs()[0].String(),
			"nodeID":  node.ID().String(),
		})

	})

	// 连接节点
	r.POST("/connect-node", func(c *gin.Context) {
		// 定义接受连接请求的结构体
		var request struct {
			From string `json:"from"`
			To   string `json:"to"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "节点路径错误！",
				"code":    "0",
			})
			return
		}

		if request.To == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "节点路径为空！",
				"code":    "0",
			})
			return
		}

		node := nodes[request.From]
		// 设置流处理器，用于处理其他节点发来的消息
		node.SetStreamHandler(protocolID, handleStream)

		if err := connectToPeer(ctx, node, request.To); err != nil {
			fmt.Println("连接失败:", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "连接失败",
				"code":    "0",
			})
		} else {
			fmt.Println("连接成功!")
			c.JSON(http.StatusOK, gin.H{
				"message": "连接成功",
				"code":    "1",
			})
		}

	})

	// 发送信息
	r.POST("/send-msg", func(c *gin.Context) {
		// 定义接受连接请求的结构体
		var request struct {
			From string `json:"from"`
			To   string `json:"to"`
			Msg  string `json:"msg"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"msg":  "绑定失败",
				"code": "0",
			})
			return
		}
		if request.To == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "节点路径为空！",
				"code":    "0",
			})
			return
		}

		node := nodes[request.From]
		// 设置流处理器，用于处理其他节点发来的消息
		node.SetStreamHandler(protocolID, handleStream)

		if err := connectToPeer(ctx, node, request.To); err != nil {
			fmt.Println("连接失败")
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "连接失败",
				"code":    "0",
			})
		} else {
			// 处理多地址，只提取最后的哈希值（节点 ID）
			maddr, err := multiaddr.NewMultiaddr(request.To)
			if err != nil {
				fmt.Printf("无效的多地址: %v\n", err)
			}
			to, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				fmt.Printf("无效的节点地址信息: %v\n", err)
			}

			fmt.Println("to.ID.String():", to.ID.String())
			if err := sendMessage(ctx, node, to.ID.String(), request.Msg); err != nil {
				fmt.Printf("error:%v\n", err)
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "发送失败",
					"code":    "0",
				})
				return
			} else {
				fmt.Println("发送成功")
				c.JSON(http.StatusOK, gin.H{
					"message": "发送成功",
					"code":    "1",
				})
			}
		}

	})

	// 点击刷新按钮，显示其他节点传来的消息
	r.POST("/refresh-msg", func(c *gin.Context) {

		if msgTemp.canRead {
			c.JSON(http.StatusOK, gin.H{
				"who": msgTemp.who,
				"msg": msgTemp.msg,
			})
			msgTemp.canRead = false
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"who": "",
				"msg": "",
			})
		}

	})

	// 接受上传来的文件
	r.POST("/upload-file", func(c *gin.Context) {
		// 1. 获取上传的文件
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "无法获取文件"})
			return
		}
		nName := c.PostForm("nodeName")
		// nAddr := c.PostForm("nodeAddr")
		// 2. 创建存储目录
		uploadDir := "./uploads" + "/" + nName // 当前程序目录下的 "uploads/[节点名称]" 文件夹
		err = os.MkdirAll(uploadDir, os.ModePerm)
		if err != nil {
			c.JSON(500, gin.H{"error": "创建目录失败"})
			return
		}
		// 3. 保存文件到目录
		filePath := filepath.Join(uploadDir, file.Filename) // 构建文件路径
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(500, gin.H{"error": "保存文件失败"})
			return
		}
		// 4. 返回成功响应
		c.JSON(200, gin.H{"message": "文件上传成功", "path": filePath})
	})

	// 刷新文件列表，供用户下载
	r.POST("/refresh-file-list", func(c *gin.Context) {

		// 	对外广播文件列表请求
		whichNode = c.PostForm("nodeName")
		clear(knownFiles)
		broadcastFilesAsk(ipAddr, portFile)

		// 等待监听协程收集其他节点传来的文件列表信息
		time.Sleep(2 * time.Second)
		knownFilesLock.Lock()
		c.JSON(http.StatusOK, gin.H{"files": knownFiles[whichNode]})
		fmt.Println(knownFiles)
		knownFilesLock.Unlock()

	})

	// 查询现有节点
	r.POST("/exist-nodes", func(c *gin.Context) {
		if len(knownNodes) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  "0",
				"msg":   "当前没有节点在线",
				"nodes": []gin.H{},
			})
			return
		}

		nodeList := make([]gin.H, 0)
		for k, v := range knownNodes {
			nodeList = append(nodeList, gin.H{
				"nodeName": k,
				"nodeAddr": v,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code":  "1",
			"msg":   "成功",
			"nodes": nodeList,
		})
	})

	// 文件下载路由
	r.POST("/download-file", downloadFile)

	// 启动服务
	r.Run(":" + port) // 服务监听在 8080 端口
}

// 创建 Libp2p 节点函数
func newNode(address string) (host.Host, error) {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings(address), // 监听地址
	)
	if err != nil {
		log.Println("创建节点失败:", err)
		return nil, err
	}
	fmt.Println("节点创建成功:", node.ID(), "地址:", node.Addrs())
	return node, nil
}

// 处理传入的流（用于接收消息）
func handleStream(stream network.Stream) {
	fmt.Println("收到来自节点的连接:", stream.Conn().RemotePeer().String())
	// 确保函数退出时关闭流
	defer stream.Close()

	// 创建一个缓冲读取器，用于读取流中的数据
	buf := bufio.NewReader(stream)
	for {
		// 按行读取消息
		msg, err := buf.ReadString('\n')
		if err != nil {
			fmt.Println("读取消息出错:", err)
			return
		}

		// 把消息传入消息体
		if !msgTemp.canRead {
			msgTemp.who = stream.Conn().RemotePeer().String()
			msgTemp.msg = msg
			msgTemp.canRead = true
		}

		// 打印收到的消息
		fmt.Printf("收到消息: %s", msg)
	}
}

// 连接到一个节点
func connectToPeer(ctx context.Context, node host.Host, peerAddr string) error {
	// 解析用户输入的多地址
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("无效的多地址: %v", err)
	}

	// 从多地址中提取 Peer 信息
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("无效的节点地址信息: %v", err)
	}

	// 使用 libp2p 节点连接到目标节点
	if err := node.Connect(ctx, *info); err != nil {
		return fmt.Errorf("连接到节点失败: %v", err)
	}

	fmt.Println("已连接到节点:", info.ID.String())
	return nil
}

// 向目标节点发送消息
func sendMessage(ctx context.Context, node host.Host, peerID, message string) error {
	// 解析用户输入的 Peer ID
	peerIDObj, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("无效的 Peer ID: %v", err)
	}

	// 创建一个到目标节点的新流
	stream, err := node.NewStream(ctx, peerIDObj, protocolID)
	if err != nil {
		return fmt.Errorf("创建流失败: %v", err)
	}
	// 确保函数退出时关闭流
	defer stream.Close()

	// 向流中写入消息
	_, err = stream.Write([]byte(message + "\n"))
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	return nil
}

// 向外广播节点上线消息
func broadcast(h host.Host, nName string) {
	// 将其节点信息存储在结构体中广播出去
	msg := nodesBroadcast{
		NodeName: nName,
		NodeAddr: h.Addrs()[0].String() + "/p2p/" + h.ID().String(),
	}
	fmt.Println("准备广播：", msg)
	conn, err := net.Dial("udp", "255.255.255.255:"+port)
	// conn, err := net.Dial("udp", "255.255.255.255:8082")
	if err != nil {
		fmt.Println("创建 upd 失败")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("序列化数据失败")
	}
	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("发送 udp 广播失败")
	}

}

// 监听节点上线广播消息
func broadcastListen() {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		fmt.Println("监听 UDP 地址出错")
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("启动 UDP 监听失败")
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("UDP 读取失败")
			continue
		}
		var info nodesBroadcast
		err = json.Unmarshal(buffer[:n], &info)
		if err != nil {
			fmt.Println("反序列化失败")
			continue
		}
		// 检测是否为自己广播的节点，忽略自己的消息
		flag := 1
		for _, v := range nodes {
			theNodeIhave := v.Addrs()[0].String() + "/p2p/" + v.ID().String()
			if theNodeIhave == info.NodeAddr {
				flag = 0
				break
			}
		}
		if flag == 1 {
			knownNodes[info.NodeName] = info.NodeAddr
			fmt.Println("收到节点在线广播:", info)
		}
	}
}

// 广播文件列表请求消息
func broadcastFilesAsk(myIpAddress string, myPort string) {
	msg := whoAskFiles{
		IPAddress: myIpAddress,
		Port:      myPort,
	}
	fmt.Println("发送文件查询广播")
	conn, err := net.Dial("udp", "255.255.255.255:"+myPort)
	if err != nil {
		fmt.Println("创建 upd 失败")
		return
	}
	defer conn.Close()
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("序列化数据失败")
		return
	}
	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("发送 udp 广播失败")
	}
}

// 监听文件查询回应
func listenFileReqAndResp() {
	addr, err := net.ResolveUDPAddr("udp", ":"+portFile)
	// addr, err := net.ResolveUDPAddr("udp", ":9699")
	if err != nil {
		fmt.Println("监听 UDP 地址出错")
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("启动 UDP 监听失败")
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, whereReqFrom, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("UDP 读取失败")
			continue
		}
		var info whoAskFiles
		err = json.Unmarshal(buffer[:n], &info)
		if err != nil {
			fmt.Println("反序列化失败")
			continue
		}

		var filesInfo []nodeFileInfo
		for k, _ := range nodes {
			nodeID := k
			fileList := getFilesFromDirectory("./uploads/" + nodeID + "/")
			if fileList == nil {
				continue
			}
			filesInfo = append(filesInfo, nodeFileInfo{
				OwnerIP:  ipAddr,
				NodeName: nodeID,
				Files:    fileList,
			})
		}
		// 创建响应
		response := fileResponse{
			Nodes: filesInfo,
		}
		data, _ := json.Marshal(response)

		// 将响应单播回请求者
		fmt.Println("请求文件 IP 来自：", whereReqFrom.IP.String())
		udpAddr, err := net.ResolveUDPAddr("udp", whereReqFrom.IP.String()+":"+portListenFileResp)
		// udpAddr, err := net.ResolveUDPAddr("udp", whereReqFrom.IP.String()+":"+"9790")
		conn.WriteToUDP(data, udpAddr)
	}
}

// 查询目录的文件列表
func getFilesFromDirectory(baseDir string) []string {
	// 检查 baseDir 是否存在并且是一个目录
	info, err := os.Stat(baseDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var files []string
	// 使用 filepath.Walk 遍历目录
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// 忽略错误
			return nil
		}
		if !info.IsDir() {
			files = append(files, info.Name())
		}
		return nil
	})

	return files
}

// 监听其他节点传来的文件信息
func collectFileResponses() {

	addr, err := net.ResolveUDPAddr("udp", ":"+portListenFileResp)
	if err != nil {
		log.Fatalf("无法解析UDP地址: %v", err)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("监听UDP地址失败: %v", err)
		return
	}
	defer conn.Close() // 确保在退出前关闭连接

	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var response fileResponse
		err = json.Unmarshal(buf[:n], &response)
		if err != nil {
			continue
		}

		for _, node := range response.Nodes {
			knownFilesLock.Lock()
			knownFiles[whichNode] = append(knownFiles[whichNode], node)
			fmt.Println("成功监听到文件！")
			knownFilesLock.Unlock()
		}
	}
}

func downloadFile(c *gin.Context) {
	var request struct {
		OwnerIP  string `json:"ownerIP"`
		NodeName string `json:"nodeName"`
		FileName string `json:"fileName"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据包"})
		return
	}

	// 假设文件存储路径结构为 /data/{NodeName}/{FileName}
	fmt.Println("对方 IP 为：", request.OwnerIP)
	basePath := "./uploads"
	filePath := filepath.Join(basePath, request.NodeName, request.FileName)
	fmt.Println("本地文件下载执行")

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件未找到"})
		return
	}

	// 设置响应头并返回文件
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // 允许所有来源
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", request.FileName))
	c.File(filePath)
}
