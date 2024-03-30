package chatroom_manager

import (
	"chatroom/parameter"
	"chatroom/server/chatroom"
	"chatroom/server/user"
	"fmt"
	"log"
	"math/rand"
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
	mutexChannel           chan bool             // 确保实际确认是只有一个协程执行 actualConfirmation 方法
	finishConfirmChannel   chan bool             // 通知 确保执行完成 实际确认 actualConfirmation 方法
	blockAndStoreChannel   chan *user.User       // 在 actualConfirmation 方法时，其他协程存储的通道
}

func NewChatroomManager() *ChatroomManager {
	chatroomManager := &ChatroomManager{
		IChatrooms:             make([]chatroom.IChatroom, 0), // 一定要初始容量为0,否则LogPerCheckCurAllocChatroomNumber方法会空指针
		chatroomMaxCapacity:    parameter.ChatroomMaxCapacity,
		OperateChatroomChannel: make(chan *OperateChatroom),
		mutexChannel:           make(chan bool),
		finishConfirmChannel:   make(chan bool),
		blockAndStoreChannel:   make(chan *user.User, parameter.BlockBufferChannelSize),
	}
	// 在初始化的时候分配一个房间
	chatroomManager._addChatroom(chatroom.NewChatroom())

	// 一定要在 分配一个房间 之后 不然会空指针
	go chatroomManager.LogPerCheckCurAllocChatroomNumber()
	chatroomManager.chatroomManagerId.Add(1)
	go chatroomManager.listenAndOperateChatroom()
	go chatroomManager.listenFinishConfirm()
	go chatroomManager.sendMutexChannelSign()

	return chatroomManager
}

// 在创建 ChatroomManager 时启动，确保后续select只有一个协程拿到消息，单读执行 actualConfirmation 方法
// 其他协程只能等待，可以有多种等待方法实现
func (cm *ChatroomManager) sendMutexChannelSign() {
	cm.mutexChannel <- true
}

// 发送完成消息，确保执行完成 "实际确认" actualConfirmation 方法
func (cm *ChatroomManager) sendFinishConfirmChannel() {
	cm.finishConfirmChannel <- true
}

// 监听 finishConfirmChannel 确保执行完成收到消息，同步其他协程
// 实现其他协程的逻辑
func (cm *ChatroomManager) listenFinishConfirm() {
	for {
		if <-cm.finishConfirmChannel {
			// 其他协程尝试重新进入聊天室
			for u := range cm.blockAndStoreChannel {
				cm.AssignRoomToUser(u)
			}
		}
	}
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
		log.Printf("增加时: Id为%d的房间UserMap的地址为%p\n", distChatroom.RoomId.Load(), distChatroom.UserMap)
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
	if len(cm.IChatrooms) >= cm.chatroomMaxCapacity {
		log.Printf("聊天室已经超过最大分配额度了，%d", len(cm.IChatrooms))
		return false, nil
	}

	success, foundChatroom := cm.probabilitySamplingTryEnterRoom(user)
	if success {
		return true, foundChatroom
	}
	cm.tryGetMutexSign(user)
	// 释放 该消费协程，消费其他并发进来的协程
	return false, nil
}

// 尝试抢 “锁”（消息），为了单独执行 actualConfirmation 方法
func (cm *ChatroomManager) tryGetMutexSign(user *user.User) {
	select {
	case <-cm.mutexChannel:
		log.Println("有一个协程拿到了\"锁\"(消息)")
		go cm.actualConfirmation(user)
	default: // mutexChannel阻塞时的策略
		// 阻塞时的策略: 1. 休息 actualConfirmation 的时间? 2. 送进带缓冲的channel中等待
		// 实现 送进带缓冲的channel中等待 的逻辑
		cm.blockAndStoreChannel <- user
		//time.Sleep(time.Second)
		//cm.AssignRoomToUser(user)
	}
}

// 并发安全的方法:
// 房间满了？遍历所有房间，实际确认:
// 1. 房间确实满了：增加新聊天室
// 2. 房间没有满：将该用户送进空置聊天室
func (cm *ChatroomManager) actualConfirmation(user *user.User) {
	// 完成 actualConfirmation 之前所要做的操作
	defer cm.finishConfirmation()
	for i := 0; i < len(cm.IChatrooms); i++ {
		curIChatroom := cm.IChatrooms[i]
		// 成功找到聊天室
		if curIChatroom.AddUserToRoom(user) {
			return
		}
	}
	// 没有空置聊天室，尝试新创建一个聊天室
	cr := chatroom.NewChatroom()
	cm._addChatroom(cr)
}

// 完成 actualConfirmation 之前所要做的操作
func (cm *ChatroomManager) finishConfirmation() {
	// 发送完成消息，一定要在该方法上加锁
	go cm.sendFinishConfirmChannel()
	// 再次监听"锁"消息
	go cm.sendMutexChannelSign()
}

// 概率采样，尝试进房间
// 概率采样失败，需要实际确认 是否分配新房间
func (cm *ChatroomManager) probabilitySamplingTryEnterRoom(user *user.User) (bool, chatroom.IChatroom) {
	var chatroomIndex int
	for i := 0; i < parameter.MaxNumberOfRetries; i++ {
		chatroomIndex = rand.Intn(len(cm.IChatrooms))
		// 尝试进房间
		successEnter := cm.IChatrooms[chatroomIndex].AddUserToRoom(user)
		if !successEnter {
			continue
		}
		// 成功找到房间
		return true, cm.IChatrooms[chatroomIndex]
	}
	// 没有找到房间，概率采样失败
	return false, nil
}

// 每秒检查 distChatroom.UserMap，如果为len == 0, 则删除该 distChatroom
func (cm *ChatroomManager) perCheckDeleteChatroom(IDistChatroom chatroom.IChatroom) {
	distChatroom, ok := IDistChatroom.(*chatroom.Chatroom)
	if !ok {
		log.Panicln("*chatroom.Chatroom 没有实现 IChatroom 接口")
	}
	for {
		//log.Printf("房间ID为%d 的UserMap 长度为: %d\n", distChatroom.RoomId.Load(), distChatroom.UserMap.Len())
		//log.Println("distChatroom.SignChannel:", distChatroom.SignChannel)
		if _, open := <-distChatroom.SignChannel; open {
			if distChatroom.UserMap.Len() == 0 {
				cm.DeleteChatroom(distChatroom)
				return
			}
		}
		time.Sleep(time.Second)
	}

}

// 每秒检查聊天室数量的Logger
func (cm *ChatroomManager) LogPerCheckCurAllocChatroomNumber() {
	for {
		n := len(cm.IChatrooms)
		fmt.Println("The current chat room number is:", n)
		for i := 0; i < n; i++ {
			cr := cm.IChatrooms[i].(*chatroom.Chatroom)
			//log.Printf("第%d个聊天室的用户数量有%d位，用户分别是%v\n", i+1, cr.UserMap.Len(), cr.UserMap)
			log.Printf("第%d个聊天室的用户数量有%d位\n", i+1, cr.UserMap.Len())
		}
		time.Sleep(time.Second)
	}
}
