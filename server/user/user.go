package user

import (
	"chatroom/utils"
	"log"
	"net"
	"strings"
)

// 用户对象
type User struct {
	UserName           string       // 对应用户名称
	UserIP             string       // 对应用户的IP地址
	UserPort           string       // 对应用户的端口号
	Conn               net.Conn     // 对应聊天用户的链接
	PrivateChatChannel chan string  // 对应私聊的channel
	UserMap            *SafeUserMap // 每一个聊天室的Map TODO 需要修改
}

func NewUser(userName, userIP, userPort string, conn net.Conn, userMap *SafeUserMap) *User {
	user := &User{
		UserName:           userName,
		UserIP:             userIP,
		UserPort:           userPort,
		Conn:               conn,
		PrivateChatChannel: make(chan string),
		UserMap:            userMap,
	}

	go user.listenAndSendPrivateMsg()
	return user
}

// 监听 PrivateChatChannel 处理message
func (u *User) listenAndSendPrivateMsg() {
	for {
		if msg, open := <-u.PrivateChatChannel; open {
			// 在私聊之前判断，一定有两个split
			splitMsg := strings.Split(msg, "#")
			distName, msgBody := splitMsg[0], splitMsg[1]
			distUser, ok := u.UserMap.GetUser(distName)
			if !ok {
				log.Printf("SafeUserMap 没有 %s", distName)
			}
			utils.SendMessage(distUser.Conn, msgBody)
		}
	}
}

// 私聊的处理逻辑
func (u *User) PrivateMsgHandler(msgBody string) {
	u.PrivateChatChannel <- msgBody
}
