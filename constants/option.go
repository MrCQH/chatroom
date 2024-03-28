package constants

// 所有用户消息的标识符
//const (
//	PrivateChatOption        = "to"      // 私聊标识符
//	BroadOption              = "broad"   // 广播标识符
//	ShowAllOnlineUsersOption = "show"    // 展示当前所有用户标识符
//	MyNameOption             = "my_name" // 查看我的名字标识符
//	QuitOption               = "quit"    // 退出标识符
//)

// 重构的用户消息标识符
const (
	PrivateChatOption        = iota // 私聊标识符
	BroadOption                     // 广播标识符
	ShowAllOnlineUsersOption        // 展示当前所有用户标识符
	MyNameOption                    // 查看我的名字标识符
	QuitOption                      // 退出标识符
)
