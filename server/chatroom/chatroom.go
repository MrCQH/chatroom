package chatroom

import (
	"chatroom/constants"
	"chatroom/server/user"
	"chatroom/utils"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
)

// 聊天室的接口，实现 聊天室 必须实现该接口
type IChatroom interface {
	AddUserToRoom(*user.User) bool
	MsgHandle(*user.User)
	TerminalUserConnect(*user.User)
}

type Chatroom struct {
	RoomId           atomic.Int64      // 房间ID, 自增的ID，atomicInt
	UserMap          *user.SafeUserMap // 聊天室对应的userName(default:"IP:Port")->User 对应每个用户的map
	usersMaxCapacity int               // 房间的最大容量
	BroadcastChannel chan string       // 广播的channel
	SignChannel      chan bool         // 房间退出的信号，判断该房间是否被删除
}

func NewChatroom() *Chatroom {
	cr := &Chatroom{
		UserMap:          user.NewSafeUserMap(),
		usersMaxCapacity: 100,
		BroadcastChannel: make(chan string),
		SignChannel:      make(chan bool),
	}
	cr.RoomId.Add(1)
	go cr.listenAndSendBroadMsg()
	return cr
}

// 监听BroadcastChannel，发送广播消息
func (cr *Chatroom) listenAndSendBroadMsg() {
	for {
		if msg, open := <-cr.BroadcastChannel; open {
			log.Printf("%d BroadcastChannel have message: %s", cr.RoomId.Load(), msg)
			cr.UserMap.Range(func(key any, value any) bool {
				u, success := value.(*user.User)
				if !success {
					panic("SafeUserMap 的 Value 不是*User类型")
				}
				_, err := u.Conn.Write([]byte(msg))
				utils.CheckError(err, fmt.Sprintf("%s write", u.Conn.RemoteAddr()))
				return true
			})
		}
	}
}

// 广播的处理逻辑
func (cr *Chatroom) broadHandler(msgBody string) {
	cr.BroadcastChannel <- msgBody
}

// 用户进入房间的逻辑, 检查并保存user, 返回当前分配 成功/失败
func (cr *Chatroom) AddUserToRoom(user *user.User) bool {
	if cr.UserMap.Len() >= cr.usersMaxCapacity {
		utils.SendMessage(user.Conn, fmt.Sprintf("当前房间已满，你无法进入"))
		return false
	}
	utils.SendMessage(user.Conn, fmt.Sprintf("你已分配到ID为%d的房间, %d", cr.RoomId.Load()))
	log.Printf("用户名字为%s，已分配到ID为%d的房间", user.UserName, cr.RoomId.Load())
	cr.UserMap.SetUser(user.UserName, user)
	return true
}

// 处理消息每一个用户消息的逻辑
func (cr *Chatroom) MsgHandle(user *user.User) {
	curConn := user.Conn
	readBytes := make([]byte, 512)
	for {
		n, err := curConn.Read(readBytes)
		//log.Printf("接收字节长度:%d, 字节内容是:%s\n", n, readBytes)
		if utils.CheckError(err, "Read") {
			return
		}
		if n == 0 || err == io.EOF {
			remoteAddr := curConn.RemoteAddr().String()
			log.Printf("%s 用户已下线\n", remoteAddr)
			cr.UserMap.DeleteUser(remoteAddr)
			return
		}
		cr.parseMsg(string(readBytes[0:n-1]), user)
	}
}

// 解析消息格式并处理msg
// <option>|<...>
// eg: 0|<name>|<msgbody>
// eg: 1|<msgbody>
func (cr *Chatroom) parseMsg(msg string, user *user.User) {
	const FormatN = 3
	curConn := user.Conn
	remoteAddr := curConn.RemoteAddr().String()
	msgSplit := strings.Split(msg, "|")
	if len(msgSplit) > FormatN {
		utils.SendMessage(curConn, "你的消息格式不对，请重新输入.\n"+constants.GetDynamicConstIntroduceStr())
		return
	}
	// 不是正确的option格式
	msgOption, err := strconv.Atoi(msgSplit[0])
	if utils.CheckError(err, "Strconv.Atoi") {
		utils.SendMessage(curConn, "你的消息格式不对，请重新输入.\n"+constants.GetDynamicConstIntroduceStr())
		return
	}

	log.Println("Message option:", msgOption)
	//if !cr.checkOptionCondition(msgOption) {
	//	utils.SendMessage(curConn, constants.GetDynamicConstIntroduceStr())
	//	return
	//}

	switch msgOption {
	case constants.QuitOption:
		log.Println("Quit remoteAddr:", remoteAddr)
		u, ok := cr.UserMap.GetUser(remoteAddr)
		if !ok {
			log.Panicf("SafeUserMap没有用户名为: %s的用户", u.UserName)
		}
		utils.SendMessage(curConn, fmt.Sprintf("Bye~ %s\n", u.UserName))
		cr.TerminalUserConnect(user)
	case constants.PrivateChatOption:
		distUserName, msgBody := msgSplit[1], strings.Join(msgSplit[2:], "")
		if curUser, isPresent := cr.UserMap.GetUser(distUserName); !isPresent {
			utils.SendMessage(curConn, fmt.Sprintf("你发送的%s不存在\n", distUserName))
			return
		} else {
			curUser.PrivateMsgHandler(distUserName + "#" + msgBody + "\n")
		}
	case constants.BroadOption:
		msgBody := strings.Join(msgSplit[1:], "") + "\n"
		cr.broadHandler(msgBody)
	case constants.ShowAllOnlineUsersOption:
		var userNames string
		cr.UserMap.Range(func(key any, value any) bool {
			userName := key.(string)
			userNames += userName + "\n"
			return true
		})
		utils.SendMessage(curConn, userNames)
	case constants.MyNameOption:
		utils.SendMessage(curConn, fmt.Sprintf("你的名字是:%s\n", remoteAddr))
	default: // 格式不对，返回重新输入
		utils.SendMessage(curConn, constants.GetDynamicConstIntroduceStr())
		return
	}
}

// 检查option操作， 满足返回true，否则返回false
//func (cr *Chatroom) checkOptionCondition(option int) bool {
//	return option <= constants.QuitOption && option >= 0
//}

func (cr *Chatroom) TerminalUserConnect(user *user.User) {
	log.Printf("删除时: Id为%d的房间的UserMap地址为%p\n", cr.RoomId.Load(), cr.UserMap)
	_, success := cr.UserMap.DeleteUser(user.Conn.RemoteAddr().String())
	if !success {
		log.Printf("username为%s的用户没有删除成功", user.UserName)
	}
	cr.SignChannel <- true
	user.Conn.Close()
}
