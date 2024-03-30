package parameter

// ChatroomManager 相关参数
const (
	ChatroomMaxCapacity = 100 // 一个 ChatroomManager 管理的最大聊天室容量

)

// Chatroom 相关参数
const (
	UsersMaxCapacity = 100 // 一个 Chatroom 容纳 User 的最大容量
)

// 聊天室进入时，并发控制的参数
const (
	MaxNumberOfRetries     = 100   //概率采样的重试次数 每一层随机找房间的最大重试次数
	BlockBufferChannelSize = 10000 // 阻塞Channel的容量，也是整个聊天系统存储User的最大的容量
)
