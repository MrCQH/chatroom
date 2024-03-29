package chatroom_manager

import (
	"chatroom/server/chatroom"
	"chatroom/server/user"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// 聊天管理者的接口，实现以下 聊天室管理者 功能必须实现该接口
type IChatroomManager interface {
	AddChatroom(chatroom.IChatroom)
	DeleteChatroom(chatroom.IChatroom)
	AssignRoomToUser(*user.User) (bool, chatroom.IChatroom)
}

// 通过channel原子的操作聊天室
type OperateChatroom struct {
	option    int                // 操作标识符: 0增加，1删除 可以拓展...
	IChatroom chatroom.IChatroom // 被操作Chatroom对象
}

// 管理聊天室的对象
type ChatroomManager struct {
	chatroomManagerId      atomic.Int64          // 自增的ID
	IChatrooms             []chatroom.IChatroom  // 对应所有聊天室
	chatroomMaxCapacity    int                   // 所有聊天室的总容量
	OperateChatroomChannel chan *OperateChatroom // 维护聊天室的channel
	ChatroomLock           sync.Mutex            // 增加聊天室的锁
}

func NewChatroomManager() *ChatroomManager {
	chatroomManager := &ChatroomManager{
		IChatrooms:             make([]chatroom.IChatroom, 0), // 初始容量为0
		chatroomMaxCapacity:    100,
		OperateChatroomChannel: make(chan *OperateChatroom),
	}
	chatroomManager.chatroomManagerId.Add(1)
	go chatroomManager.listenAndOperateChatroom()
	return chatroomManager
}

// 监听和操作标识符
func (cm *ChatroomManager) listenAndOperateChatroom() {
	for {
		if operateChatroom, open := <-cm.OperateChatroomChannel; open {
			op := operateChatroom.option
			distChatroom, ok := operateChatroom.IChatroom.(*chatroom.Chatroom)
			if !ok {
				log.Panicln("*chatroom.Chatroom 没有实现 IChatroom 接口")
			}
			if op == 0 { // 增加
				cm._addChatroom(distChatroom)
			} else if op == 1 { // 删除
				cm._deleteChatroom(distChatroom)
			}
		}
	}
}

// 封装删除 Chatroom 操作
func (cm *ChatroomManager) _deleteChatroom(distChatroom *chatroom.Chatroom) {
	for index, chatroom := range cm.IChatrooms {
		if chatroom == distChatroom {
			cm.IChatrooms = append(cm.IChatrooms[:index], cm.IChatrooms[index+1:]...)
			log.Printf("已删除ID为%d的聊天室", distChatroom.RoomId.Load())
			return
		}
	}
	log.Printf("ID为%d的聊天室不存在", distChatroom.RoomId.Load())
}

// 封装增加 Chatroom 操作
func (cm *ChatroomManager) _addChatroom(IDistChatroom chatroom.IChatroom) {
	if distChatroom, ok := IDistChatroom.(*chatroom.Chatroom); !ok {
		log.Panicln("*chatroom.Chatroom 没有实现 IChatroom 接口")
	} else {
		if len(cm.IChatrooms) > cm.chatroomMaxCapacity {
			log.Printf("聊天室已满，无法装下ID为%d的聊天室", distChatroom.RoomId.Load())
			return
		}
		cm.IChatrooms = append(cm.IChatrooms, distChatroom)
		go cm.perCheckDeleteChatroom(distChatroom)
		log.Printf("已增加ID为%d的聊天室\n", distChatroom.RoomId.Load())
	}
}

// 对外暴露的 AddChatroom
func (cm *ChatroomManager) AddChatroom(distChatroom chatroom.IChatroom) {
	cm.OperateChatroomChannel <- &OperateChatroom{0, distChatroom}
}

// 对外暴露的 DeleteChatroom
func (cm *ChatroomManager) DeleteChatroom(distChatroom chatroom.IChatroom) {
	cm.OperateChatroomChannel <- &OperateChatroom{1, distChatroom}
}

// 递归的分配房间给用户，随机进入一个已经存在的房间
// TODO 支持断线重连
func (cm *ChatroomManager) AssignRoomToUser(user *user.User) (bool, chatroom.IChatroom) {
	if len(cm.IChatrooms) > cm.chatroomMaxCapacity {
		log.Printf("聊天室已经超过最大分配额度了，%d", len(cm.IChatrooms))
		return false, nil
	}
	// 初始化分配，防止rand.Intn(0)
	if len(cm.IChatrooms) == 0 {
		cr := chatroom.NewChatroom()
		cm._addChatroom(cr)
		return cr.AddUserToRoom(user), cr
	}
	const MaxNumberOfRetries = 100 // 每一层随机找房间的最大重试次数
	for i := 0; i < MaxNumberOfRetries; i++ {
		chatroomIndex := rand.Intn(len(cm.IChatrooms))
		// 防止初次没有房间，一直continue到循环外，增加房间
		if cm.IChatrooms[chatroomIndex] == nil {
			continue
		}
		return cm.IChatrooms[chatroomIndex].AddUserToRoom(user), cm.IChatrooms[chatroomIndex]
	}
	cm.AddChatroom(chatroom.NewChatroom())
	// 继续调用自己，去分配房间
	return cm.AssignRoomToUser(user)
}

// 每秒检查 distChatroom.UserMap，如果为len == 0, 则删除该 distChatroom
func (cm *ChatroomManager) perCheckDeleteChatroom(IDistChatroom chatroom.IChatroom) {
	distChatroom, ok := IDistChatroom.(*chatroom.Chatroom)
	if !ok {
		log.Panicln("*chatroom.Chatroom 没有实现 IChatroom 接口")
	}
	for {
		if _, open := <-distChatroom.SignChannel; open {
			if distChatroom.UserMap.Len() == 0 {
				cm.DeleteChatroom(distChatroom)
				return
			}
		}
	}

}

// 每秒检查聊天室数量
func (cm *ChatroomManager) LogPerCheckCurAllocChatroomNumber() {
	for {
		n := len(cm.IChatrooms)
		fmt.Println("The current chat room number is:", n)
		for i := 0; i < n; i++ {
			cr := cm.IChatrooms[i].(*chatroom.Chatroom)
			//log.Printf("第%d个聊天室的用户数量有%d位，用户分别是%v\n", i+1, len(cr.UserMap), cr.UserMap)
			log.Printf("第%d个聊天室的用户数量有%d位\n", i+1, cr.UserMap.Len())
		}
		time.Sleep(time.Second)
	}
}
