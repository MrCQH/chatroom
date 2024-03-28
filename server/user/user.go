package user

import (
	"chatroom/utils"
	"net"
	"strings"
)

// 用户对象
type User struct {
	UserName           string           // 对应用户名称
	UserIP             string           // 对应用户的IP地址
	UserPort           string           // 对应用户的端口号
	Conn               net.Conn         // 对应聊天用户的链接
	PrivateChatChannel chan string      // 对应私聊的channel
	UserMap            map[string]*User // 拿到Server的userMap TODO 改为每一个聊天室的Map
}

func NewUser(userName, userIP, userPort string, conn net.Conn, userMap map[string]*User) *User {
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
			utils.SendMessage(u.UserMap[distName].Conn, msgBody)
		}
	}
}

// 私聊的处理逻辑
func (u *User) PrivateMsgHandler(msgBody string) {
	u.PrivateChatChannel <- msgBody
}
