package server

import (
	"chatroom/server/chatroom"
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
	ServerIP         string                            // 服务器对应的IP
	ServerPort       string                            // 服务器对应的端口号
	userMap          *user.SafeUserMap                 // userName(default:"IP:Port")->*User 对应每个用户的map
	EnterRoomChannel chan *user.User                   // 进入房间的channel
	IChatroomManager chatroom_manager.IChatroomManager // 该服务器对应的 IChatroomManager
}

// 创建聊天服务器
func NewChatServer(serverIP, serverPort string) *ChatServer {
	chatroomManager := chatroom_manager.NewChatroomManager()
	chatServer := &ChatServer{
		ServerIP:         serverIP,
		ServerPort:       serverPort,
		userMap:          user.NewSafeUserMap(),
		EnterRoomChannel: make(chan *user.User),
		IChatroomManager: chatroomManager,
	}

	go chatroomManager.LogPerCheckCurAllocChatroomNumber()

	go chatServer.consumEnterUser()

	return chatServer
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
		curUser := c.storeUser(conn)
		c.userEnterRoom(curUser)
	}
}

// 保存链接的逻辑, 如果历史访问过，返回的是该用户，否则返回一个新用户
func (c *ChatServer) storeUser(conn net.Conn) *user.User {
	remoteAddr := conn.RemoteAddr().String()
	remoteAddrSplit := strings.Split(remoteAddr, ":")
	remoteIP, remotePort := remoteAddrSplit[0], remoteAddrSplit[1]
	// userMap如果存在该user的话
	if u, isPresent := c.userMap.GetUser(remoteAddr); isPresent {
		return u
	}
	u := user.NewUser(remoteAddr, remoteIP, remotePort, conn, c.userMap)
	utils.SendMessage(conn, fmt.Sprintf("Hello, %s\n", u.UserName))
	c.userMap.SetUser(remoteAddr, u)
	return u
}

// 作为生产者，将用户放进 EnterRoomChannel 中
func (c *ChatServer) userEnterRoom(user *user.User) {
	log.Printf("%s 用户已上线\n", user.Conn.RemoteAddr().String())
	c.EnterRoomChannel <- user
}

// 作为消费者，消费 EnterRoomChannel 的用户
func (c *ChatServer) consumEnterUser() {
	for {
		log.Printf("尝试消费用户\n")
		if u, open := <-c.EnterRoomChannel; open {
			log.Printf("%s 用户被消费\n", u.Conn.RemoteAddr().String())
			c.consumProcess(u)
		}
	}
}

func (c *ChatServer) consumProcess(u *user.User) {
	log.Println("EnterRoomUser:", u.UserName)
	chatroomManager, ok := c.IChatroomManager.(*chatroom_manager.ChatroomManager)
	if !ok {
		log.Panicln("*chatroom.Chatroom 没有实现 IChatroom 接口")
	}
	if isFound, Ichatroom := chatroomManager.AssignRoomToUser(u); !isFound {
		utils.SendMessage(u.Conn, "本聊天室服务器分配已满")
		log.Println("本聊天室服务器分配已满")
		chatroomManager.AddChatroom(chatroom.NewChatroom())
		chatroomManager.AssignRoomToUser(u)
	} else {
		cr, ok := Ichatroom.(*chatroom.Chatroom)
		if !ok {
			log.Panicln("*chatroom.Chatroom 没有实现 Ichatroom 接口")
		}
		log.Printf("已经分配聊天室，ID为%d\n", cr.RoomId.Load())
		go cr.MsgHandle(u)
	}
}
