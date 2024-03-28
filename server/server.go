package server

import (
	"chatroom/constants"
	"chatroom/utils"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

// 聊天服务器对象
type ChatServer struct {
	ServerIP         string           // 服务器对应的IP
	ServerPort       string           // 服务器对应的端口号
	userMap          map[string]*User // userName(default:"IP:Port")->User 对应每个用户的map
	BroadcastChannel chan string      // 广播的channel
	mapLock          sync.RWMutex     // 对应userMap的读写锁，防止并发数据访问
}

// 创建聊天服务器
func NewChatServer(serverIP, serverPort string) *ChatServer {
	chatServer := &ChatServer{
		ServerIP:         serverIP,
		ServerPort:       serverPort,
		userMap:          make(map[string]*User),
		BroadcastChannel: make(chan string),
	}

	go chatServer.listenAndSendBroadMsg()

	return chatServer
}

// 监听BroadcastChannel，发送广播消息
func (c *ChatServer) listenAndSendBroadMsg() {
	for {
		if msg, open := <-c.BroadcastChannel; open {
			log.Println("BroadcastChannel have message:", msg)
			c.mapLock.Lock()
			for _, user := range c.userMap {
				_, err := user.Conn.Write([]byte(msg))
				utils.CheckError(err, fmt.Sprintf("%s write", user.Conn.RemoteAddr()))
			}
			c.mapLock.Unlock()
		}
	}
}

// 私聊的处理逻辑
func (c *ChatServer) broadHandler(msgBody string) {
	c.BroadcastChannel <- msgBody
}

// 监听对应端口，执行handle
func (c *ChatServer) Start() {
	localAddress := fmt.Sprintf("%s:%s", c.ServerIP, c.ServerPort)
	log.Printf("Local Address: %s\n", localAddress)
	listener, err := net.Listen("tcp", localAddress)
	utils.CheckError(err, "Listener", listener)
	for {
		conn, err := listener.Accept()
		utils.CheckError(err, "Accept", conn)
		conn = c.storeUser(conn)
		go c.msgHandle(conn)
	}
}

// 处理消息每一个用户消息的逻辑
func (c *ChatServer) msgHandle(curConn net.Conn) {
	readBytes := make([]byte, 1024)
	for {
		n, err := curConn.Read(readBytes)
		//log.Printf("接收字节长度:%d, 字节内容是:%s\n", n, readBytes)
		if utils.CheckError(err, "Read") {
			return
		}
		if n == 0 || err == io.EOF {
			remoteAddr := curConn.RemoteAddr().String()
			log.Printf("%s 用户已下线\n", remoteAddr)
			delete(c.userMap, remoteAddr)
			return
		}
		c.ParseMsg(string(readBytes[0:n-1]), curConn)
	}
}

// 保存链接的逻辑, 返回的是该用户的链接
func (c *ChatServer) storeUser(conn net.Conn) net.Conn {
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("%s 用户已上线", remoteAddr)
	remoteAddrSplit := strings.Split(remoteAddr, ":")
	remoteIP, remotePort := remoteAddrSplit[0], remoteAddrSplit[1]
	// userMap如果存在该user的话
	if user, isPresent := c.userMap[remoteAddr]; isPresent {
		return user.Conn
	}
	user := NewUser(remoteAddr, remoteIP, remotePort, conn, c.userMap)
	utils.SendMessage(conn, fmt.Sprintf("Hello, %s\n", user.UserName))
	c.userMap[remoteAddr] = user
	return conn
}

// 解析消息格式并处理msg
// <option>|<...>
// eg: to|<name>|<msgbody>
// eg: broad|<msgbody>
func (c *ChatServer) ParseMsg(msg string, curConn net.Conn) {
	const FormatN = 3
	remoteAddr := curConn.RemoteAddr().String()
	msgSplit := strings.Split(msg, "|")
	if len(msgSplit) > FormatN {
		utils.SendMessage(curConn, "你的消息格式不对，请重新输入.\n")
		return
	}
	msgOption := msgSplit[0]
	log.Println("Message option:", msgOption)
	if checkOption(msgOption) {
		utils.SendMessage(curConn, constants.IntroduceStr)
		return
	}
	if msgOption == constants.QuitOption {
		remoteAddr = curConn.RemoteAddr().String()
		utils.SendMessage(curConn, fmt.Sprintf("Bye~ %s\n", c.userMap[remoteAddr].UserName))
		c.TerminalConnect(curConn, remoteAddr)
	} else if msgOption == constants.PrivateChatOption {
		distUserName, msgBody := msgSplit[1], strings.Join(msgSplit[2:], "")
		if user, isPresent := c.userMap[distUserName]; !isPresent {
			utils.SendMessage(curConn, fmt.Sprintf("你发送的%s不存在\n", distUserName))
			return
		} else {
			user.privateMsgHandler(distUserName + "#" + msgBody + "\n")
		}
	} else if msgOption == constants.BroadOption {
		msgBody := strings.Join(msgSplit[1:], "") + "\n"
		c.broadHandler(msgBody)
	} else if msgOption == constants.ShowAllOnlineUsersOption {
		var userNames string
		for userName := range c.userMap {
			userNames += userName + "\n"
		}
		utils.SendMessage(curConn, userNames)
	} else if msgOption == constants.MyNameOption {
		utils.SendMessage(curConn, fmt.Sprintf("你的名字是:%s\n", remoteAddr))
	}
}

// 检查option操作
func checkOption(option string) bool {
	return option != constants.PrivateChatOption &&
		option != constants.BroadOption &&
		option != constants.ShowAllOnlineUsersOption &&
		option != constants.MyNameOption &&
		option != constants.QuitOption
}

func (c *ChatServer) TerminalConnect(conn net.Conn, remoteAddr string) {
	delete(c.userMap, remoteAddr)
	conn.Close()
}
