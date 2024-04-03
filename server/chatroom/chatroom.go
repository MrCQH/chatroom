package chatroom

import (
	"chatroom/constants"
	"chatroom/parameter"
	"chatroom/server/user"
	"chatroom/utils"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

// 聊天室的接口，实现 聊天室 必须实现该接口
type IChatroom interface {
	AddUserToRoom(*user.User) bool
	MsgHandle(*user.User)
	TerminalUserConnect(*user.User)
}

type Chatroom struct {
	RoomId           int                   // 房间ID, 自增的ID，atomicInt
	UserMap          map[string]*user.User // 聊天室对应的userName(default:"IP:Port")->User 对应每个用户的map
	usersMaxCapacity int                   // 房间的最大容量
	BroadcastChannel chan string           // 广播的channel
	SignChannel      chan bool             // 房间退出的信号，判断该房间是否被删除
}

// curChatroomCnt当前房间数，为了计算当前房间的ID
func NewChatroom(curChatroomCnt int) *Chatroom {
	cr := &Chatroom{
		UserMap:          make(map[string]*user.User),
		usersMaxCapacity: parameter.UsersMaxCapacity,
		BroadcastChannel: make(chan string),
		SignChannel:      make(chan bool),
	}
	cr.RoomId = curChatroomCnt + 1
	go cr.listenAndSendBroadMsg()
	return cr
}

// 监听BroadcastChannel，发送广播消息
func (cr *Chatroom) listenAndSendBroadMsg() {
	for {
		if msg, open := <-cr.BroadcastChannel; open {
			log.Printf("%d BroadcastChannel have message: %s", cr.RoomId, msg)
			for _, u := range cr.UserMap {
				_, err := u.Conn.Write([]byte(msg))
				// 如果发送失败就 尝试重复对该用户补偿发送10次 每次间隔1秒，为了避免对方网络波动等相关问题
				if utils.CheckError(err, fmt.Sprintf("%s broadMsg write", u.Conn.RemoteAddr())) {
					for i := 0; i < 10; i++ {
						if _, err := u.Conn.Write([]byte(msg)); err == nil {
							break
						}
						time.Sleep(time.Second)
					}
				}
			}
		}
	}
}

// 广播的处理逻辑
func (cr *Chatroom) broadHandler(msgBody string) {
	log.Println("已经成功发送了广播消息")
	cr.BroadcastChannel <- msgBody
}

// 用户进入房间的逻辑, 检查并保存user, 返回当前分配 成功/失败
func (cr *Chatroom) AddUserToRoom(user *user.User) bool {
	if len(cr.UserMap)+1 > cr.usersMaxCapacity {
		utils.SendMessage(user.Conn, fmt.Sprintf("当前房间已满，你无法进入"))
		log.Printf("当前房间ID为%d已满，房间人数为%d，%s的用户无法进入", cr.RoomId, len(cr.UserMap), user.UserName)
		return false
	}
	utils.SendMessage(user.Conn, fmt.Sprintf("你已分配到ID为%d的房间, %d", cr.RoomId))
	log.Printf("用户名字为%s，已分配到ID为%d的房间", user.UserName, cr.RoomId)
	cr.UserMap[user.UserName] = user
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
			delete(cr.UserMap, remoteAddr)
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
		utils.SendMessage(curConn, "你的消息格式不对，请重新输入.\n"+constants.DynamicConstIntroduceStr())
		log.Println(curConn, "你的消息格式不对，请重新输入.\n"+constants.DynamicConstIntroduceStr())
		return
	}
	// 不是正确的option格式
	msgOption, err := strconv.Atoi(msgSplit[0])
	if utils.CheckError(err, "Strconv.Atoi") {
		utils.SendMessage(curConn, "你的消息格式不对，请重新输入.\n"+constants.DynamicConstIntroduceStr())
		log.Println(curConn, "你的消息格式不对，请重新输入.\n"+constants.DynamicConstIntroduceStr())
		return
	}

	log.Println("Message option:", msgOption)

	switch msgOption {
	case constants.QuitOption:
		log.Println("Quit remoteAddr:", remoteAddr)
		u, ok := cr.UserMap[remoteAddr]
		if !ok {
			log.Panicf("SafeUserMap没有用户名为: %s的用户", u.UserName)
		}
		utils.SendMessage(curConn, fmt.Sprintf("Bye~ %s\n", u.UserName))
		log.Println(curConn, fmt.Sprintf("Bye~ %s\n", u.UserName))
		cr.TerminalUserConnect(user)
	case constants.PrivateChatOption:
		distUserName, msgBody := msgSplit[1], strings.Join(msgSplit[2:], "")
		if curUser, isPresent := cr.UserMap[distUserName]; !isPresent {
			utils.SendMessage(curConn, fmt.Sprintf("你发送的%s不存在\n", distUserName))
			log.Println(curConn, fmt.Sprintf("你发送的%s不存在\n", distUserName))
			return
		} else {
			curUser.PrivateMsgHandler(distUserName + "#" + msgBody + "\n")
		}
	case constants.BroadOption:
		msgBody := strings.Join(msgSplit[1:], "") + "\n"
		cr.broadHandler(msgBody)
	case constants.ShowAllOnlineUsersOption:
		var userNames string
		for userName := range cr.UserMap {
			userNames += userName + "\n"
		}
		utils.SendMessage(curConn, userNames)
	case constants.MyNameOption:
		utils.SendMessage(curConn, fmt.Sprintf("你的名字是:%s\n", remoteAddr))
	default: // 格式不对，返回重新输入
		utils.SendMessage(curConn, constants.DynamicConstIntroduceStr())
		log.Println(curConn, constants.DynamicConstIntroduceStr())
		return
	}
}

// 用户退出连接时，所做的后处理
func (cr *Chatroom) TerminalUserConnect(user *user.User) {
	log.Printf("删除时: Id为%d的房间的UserMap地址为%p\n", cr.RoomId, cr.UserMap)
	delete(cr.UserMap, user.Conn.RemoteAddr().String())
	cr.SignChannel <- true
	user.Conn.Close()
}
