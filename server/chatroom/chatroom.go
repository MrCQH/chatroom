package chatroom

import (
	"chatroom/constants"
	"chatroom/server/user"
	"chatroom/utils"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type Chatroom struct {
	RoomId           atomic.Int64          // 房间ID, 自增的ID，atomicInt
	UserMap          map[string]*user.User // 聊天室对应的userName(default:"IP:Port")->User 对应每个用户的map
	usersMaxCapacity int                   // 房间的最大容量
	BroadcastChannel chan string           // 广播的channel
	mapLock          sync.Mutex            // 对应userMap的锁，防止并发数据访问
}

func NewChatroom() *Chatroom {
	cr := &Chatroom{
		UserMap:          make(map[string]*user.User),
		usersMaxCapacity: 500,
		BroadcastChannel: make(chan string),
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
			cr.mapLock.Lock()
			for _, user := range cr.UserMap {
				_, err := user.Conn.Write([]byte(msg))
				utils.CheckError(err, fmt.Sprintf("%s write", user.Conn.RemoteAddr()))
			}
			cr.mapLock.Unlock()
		}
	}
}

// 广播的处理逻辑
func (cr *Chatroom) broadHandler(msgBody string) {
	cr.BroadcastChannel <- msgBody
}

// 用户进入房间的逻辑, 检查并保存user, 返回当前分配 成功/失败
func (cr *Chatroom) AddUserToRoom(user *user.User) bool {
	cr.mapLock.Lock()
	defer cr.mapLock.Unlock()
	curNumberOfUser := len(cr.UserMap)
	if curNumberOfUser > cr.usersMaxCapacity {
		utils.SendMessage(user.Conn, fmt.Sprintf("当前房间已满, 已分配到ID为%d的房间"))
		return false
	}
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
		cr.parseMsg(string(readBytes[0:n-1]), curConn)
	}
}

// 解析消息格式并处理msg
// <option>|<...>
// eg: 0|<name>|<msgbody>
// eg: 1|<msgbody>
func (cr *Chatroom) parseMsg(msg string, curConn net.Conn) {
	const FormatN = 3
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
		remoteAddr = curConn.RemoteAddr().String()
		utils.SendMessage(curConn, fmt.Sprintf("Bye~ %s\n", cr.UserMap[remoteAddr].UserName))
		cr.TerminalConnect(curConn, remoteAddr)
	case constants.PrivateChatOption:
		distUserName, msgBody := msgSplit[1], strings.Join(msgSplit[2:], "")
		if curUser, isPresent := cr.UserMap[distUserName]; !isPresent {
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
		for userName := range cr.UserMap {
			userNames += userName + "\n"
		}
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

func (cr *Chatroom) TerminalConnect(conn net.Conn, remoteAddr string) {
	delete(cr.UserMap, remoteAddr)
	conn.Close()
}
