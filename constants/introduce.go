package constants

import (
	"fmt"
	"sync"
)

var (
	introduceStr string
	once         sync.Once
)

func init() {
	GetDynamicConstIntroduceStr()
}

// 认为动态定义，不能修改的伪常量
func GetDynamicConstIntroduceStr() string {
	once.Do(func() {
		introduceStr = fmt.Sprintf(
			"当前仅支持私、广播、查看当前聊天室成员、展示我的名字和退出.\n"+
				" eg,privateChat:  %d|<name>|<msgbody>\n"+
				" eg,Broad:  %d|<msgbody>\n"+
				" eg,ShowAllOnlineUsers:  %d\n"+
				" eg,MyName:  %d\n"+
				" eg,quit: %d\n"+
				"请再次输入\n",
			PrivateChatOption, BroadOption, ShowAllOnlineUsersOption, MyNameOption, QuitOption)
	})
	return introduceStr
}
