package chatroom_manager

import (
	"chatroom/server/chatroom"
	"chatroom/server/user"
	"log"
	"math/rand"
	"sync/atomic"
	"time"
)

// 通过channel原子的操作聊天室
type OperateChatroom struct {
	option   int                // 操作标识符: 0增加，1删除 可以拓展...
	chatroom *chatroom.Chatroom // 被操作Chatroom对象
}

// 管理聊天室的对象
type ChatroomManager struct {
	chatroomManagerId      atomic.Int64          // 自增的ID
	chatrooms              []*chatroom.Chatroom  // 对应所有聊天室
	chatroomMaxCapacity    int                   // 所有聊天室的总容量
	OperateChatroomChannel chan *OperateChatroom // 操作聊天室的channel
}

func NewChatroomManager() *ChatroomManager {
	chatroomManager := &ChatroomManager{
		chatrooms:              make([]*chatroom.Chatroom, 10),
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
			op, distChatroom := operateChatroom.option, operateChatroom.chatroom
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
	for index, chatroom := range cm.chatrooms {
		if chatroom == distChatroom {
			cm.chatrooms = append(cm.chatrooms[:index], cm.chatrooms[index+1:]...)
			log.Printf("已删除ID为%d的聊天室", distChatroom.RoomId.Load())
			return
		}
	}
	log.Printf("ID为%d的聊天室不存在", distChatroom.RoomId.Load())
}

// 封装增加 Chatroom 操作
func (cm *ChatroomManager) _addChatroom(distChatroom *chatroom.Chatroom) {
	if len(cm.chatrooms) > cm.chatroomMaxCapacity {
		log.Printf("聊天室已满，无法装下ID为%d的聊天室", distChatroom.RoomId.Load())
		return
	}
	cm.chatrooms = append(cm.chatrooms, distChatroom)
	log.Printf("已增加ID为%d的聊天室\n", distChatroom.RoomId.Load())
}

// 对外暴露的 AddChatroom
func (cm *ChatroomManager) AddChatroom(distChatroom *chatroom.Chatroom) {
	cm.OperateChatroomChannel <- &OperateChatroom{0, distChatroom}
}

// 对外暴露的 DeleteChatroom
func (cm *ChatroomManager) DeleteChatroom(distChatroom *chatroom.Chatroom) {
	cm.OperateChatroomChannel <- &OperateChatroom{1, distChatroom}
}

// 递归的分配房间给用户，随机进入一个已经存在的房间
// TODO 支持断线重连
func (cm *ChatroomManager) AssignRoomToUser(user *user.User) (bool, *chatroom.Chatroom) {
	if len(cm.chatrooms) > cm.chatroomMaxCapacity {
		return false, nil
	}
	const MaxNumberOfRetries = 100 // 每一层随机找房间的最大重试次数
	for i := 0; i < MaxNumberOfRetries; i++ {
		chatroomIndex := rand.Intn(len(cm.chatrooms))
		// 防止初次没有房间，一直continue到循环外，增加房间
		if cm.chatrooms[chatroomIndex] == nil {
			continue
		}
		return cm.chatrooms[chatroomIndex].AddUserToRoom(user), cm.chatrooms[chatroomIndex]
	}
	cm.AddChatroom(chatroom.NewChatroom())
	// 继续调用自己，去分配房间
	return cm.AssignRoomToUser(user)
}

// 每秒检查 Chatroom.UserMap，如果为len == 0, 则删除该Chatroom
func (cm *ChatroomManager) TimeWorkDeleteChatroom() {
	for {
		for _, cr := range cm.chatrooms {
			if len(cr.UserMap) == 0 {
				cm._deleteChatroom(cr)
			}
		}
		time.Sleep(time.Second)
	}

}
