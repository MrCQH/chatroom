package parameter

import "time"

// Mongo连接的相关的参数
const (
	DatabaseUrl             = "mongodb://localhost:27017" // 数据连接的url
	DatabaseName            = "runoob"                    // 对应数据库名称
	DatabaseConnectPoolSize = 500                         // 数据库连接池大小
	Timeout                 = 20 * time.Second            // 连接的超时时间
)

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
	MaxNumberOfRetries = 100 //概率采样的重试次数 每一层随机找房间的最大重试次数
	// Deprecated: 关联到 tryGetMutexSign 方法，该方法已废弃
	//BlockBufferChannelSize = 10000 // 阻塞Channel的容量，也是整个聊天系统存储User的最大的容量
)
