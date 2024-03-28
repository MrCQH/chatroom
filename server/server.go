package server

import (
	"chatroom/server/chatroom_manager"
	"chatroom/server/user"
	"chatroom/utils"
	"fmt"
	"log"
	"net"
	"strings"
)

// 聊天服务器对象
type ChatServer struct {
	ServerIP   string                // 服务器对应的IP
	ServerPort string                // 服务器对应的端口号
	userMap    map[string]*user.User // userName(default:"IP:Port")->User 对应每个用户的map
}

// 创建聊天服务器
func NewChatServer(serverIP, serverPort string) *ChatServer {
	chatServer := &ChatServer{
		ServerIP:   serverIP,
		ServerPort: serverPort,
		userMap:    make(map[string]*user.User),
	}
	return chatServer
}

// 监听对应端口，执行handle
func (c *ChatServer) Start() {
	localAddress := fmt.Sprintf("%s:%s", c.ServerIP, c.ServerPort)
	log.Printf("Local Address: %s\n", localAddress)
	listener, err := net.Listen("tcp", localAddress)
	utils.CheckError(err, "Listener", listener)
	chatroomManager := chatroom_manager.NewChatroomManager()
	go chatroomManager.TimeWorkDeleteChatroom()
	for {
		conn, err := listener.Accept()
		utils.CheckError(err, "Accept", conn)
		curUser := c.storeUser(conn)
		go func(user *user.User) {
			if isFound, chatroom := chatroomManager.AssignRoomToUser(user); !isFound {
				utils.SendMessage(conn, "本聊天室服务器分配已满")
				return
			} else {
				log.Printf("已经分配聊天室，ID为%d\n", chatroom.RoomId.Load())
				chatroom.MsgHandle(user)
			}
		}(curUser)
	}
}

// 保存链接的逻辑, 如果历史访问过，返回的是该用户，否则返回一个新用户
func (c *ChatServer) storeUser(conn net.Conn) *user.User {
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("%s 用户已上线", remoteAddr)
	remoteAddrSplit := strings.Split(remoteAddr, ":")
	remoteIP, remotePort := remoteAddrSplit[0], remoteAddrSplit[1]
	// userMap如果存在该user的话
	if user, isPresent := c.userMap[remoteAddr]; isPresent {
		return user
	}
	user := user.NewUser(remoteAddr, remoteIP, remotePort, conn, c.userMap)
	utils.SendMessage(conn, fmt.Sprintf("Hello, %s\n", user.UserName))
	c.userMap[remoteAddr] = user
	return user
}
