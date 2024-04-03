package server

import (
	"chatroom/parameter"
	"chatroom/server/chatroom"
	"chatroom/server/chatroom_manager"
	"chatroom/server/user"
	"chatroom/utils"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net"
	"strings"
	"time"
)

// 聊天服务器对象
type ChatServer struct {
	ServerIP          string                            // 服务器对应的IP
	ServerPort        string                            // 服务器对应的端口号
	userMap           *user.SafeUserMap                 // userName(default:"IP:Port")->*User 对应每个用户的map
	EnterRoomChannel  chan *user.User                   // 进入房间的channel，顺序处理每一个用户的连接，可以增加buffer cap去增加用户并发连接数
	IChatroomManager  chatroom_manager.IChatroomManager // 该服务器对应的 IChatroomManager
	userMongoDatabase *mongo.Database                   // mongo中User数据库
}

// 创建聊天服务器
func NewChatServer(serverIP, serverPort string) *ChatServer {
	chatServer := &ChatServer{
		ServerIP:         serverIP,
		ServerPort:       serverPort,
		userMap:          user.NewSafeUserMap(),
		EnterRoomChannel: make(chan *user.User),                  //可以增加buffer cap去增加用户并发连接数(生产者)
		IChatroomManager: chatroom_manager.NewChatroomManager(0), // 目前只有一个manager去管理
	}

	database, err := connectToMongo(parameter.DatabaseUrl, parameter.DatabaseName,
		parameter.Timeout, parameter.DatabaseConnectPoolSize)
	if err != nil {
		log.Println("数据库连接失败", err)
		return nil
	}
	chatServer.userMongoDatabase = database
	go chatServer.consumEnterUser()

	return chatServer
}

// 连接到数据库，如何设置 poolSize 参数，默认选用 poolSize最后一个参数作为连接池的大小
func connectToMongo(url, databaseName string, timeout time.Duration, connPoolSize ...uint64) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	o := options.Client().ApplyURI(url)
	if len(connPoolSize) != 0 {
		o.SetMaxConnecting(connPoolSize[len(connPoolSize)-1])
	}
	client, err := mongo.Connect(ctx, o)
	if utils.CheckError(err, "client") {
		return nil, err
	}
	log.Printf("已成功连接数据库:\n"+
		"                      url为:%s\n"+
		"                      databaseName为:%s\n"+
		"                      连接池大小为:%d\n", url, databaseName, connPoolSize)
	return client.Database(databaseName), nil
}

// 监听对应端口，执行handle
func (c *ChatServer) Start() {
	localAddress := fmt.Sprintf("%s:%s", c.ServerIP, c.ServerPort)
	log.Printf("Local Address: %s\n", localAddress)
	listener, err := net.Listen("tcp", localAddress)
	utils.CheckError(err, "Listener", listener)
	for {
		conn, err := listener.Accept()
		//log.Printf("-----------------------Remote connect info: %s-----------------------\n", conn.RemoteAddr().String())
		utils.CheckError(err, "Accept", conn)
		curUser := c.storeUser(conn)
		go c.userEnterRoom(curUser)
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

// 作为消费者，消费 EnterRoomChannel 的用户，单个协程，顺序消费用户
func (c *ChatServer) consumEnterUser() {
	for {
		log.Printf("尝试消费用户\n")
		if u, open := <-c.EnterRoomChannel; open {
			log.Printf("%s 用户被消费\n", u.Conn.RemoteAddr().String())
			c.consumProcess(u)
		}
	}
}

// 消费任务的逻辑
// 如果消费成功，就开一个协程处理
// 如果消费失败，就进行将消息
func (c *ChatServer) consumProcess(u *user.User) {
	log.Println("EnterRoomUser:", u.UserName)
	chatroomManager, ok := c.IChatroomManager.(*chatroom_manager.ChatroomManager)
	if !ok {
		log.Panicln("*chatroom.Chatroom 没有实现 IChatroom 接口")
	}
	if isFound, IChatroom := chatroomManager.AssignRoomToUser(u); !isFound {
		utils.SendMessage(u.Conn, "本聊天室服务器分配已满或是没有分配到房间")
		log.Println("本聊天室服务器分配已满或是没有分配到房间")
		// 保证User不丢失，没来得及消费的User，重新放入 EnterRoomChannel，重新消费
		go c.userEnterRoom(u)
	} else {
		cr, ok := IChatroom.(*chatroom.Chatroom)
		if !ok {
			log.Panicln("*chatroom.Chatroom 没有实现 Ichatroom 接口")
		}
		log.Printf("已经分配聊天室，ID:%d\n", cr.RoomId)
		//go cr.MsgHandle(u)
		go func() {
			log.Println("有一个协程进来了")
			time.Sleep(60 * time.Second)
		}()
	}
}
