package main

import (
	"chatroom/utils"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

var serverIp string        // 链接聊天室的IP地址
var serverPort string      // 链接聊天室的端口号
const NumberOfUser = 10000 // 测试用户数量

func init() {
	flag.StringVar(&serverIp, "i", "127.0.0.1", "链接聊天室的IP地址")
	flag.StringVar(&serverPort, "p", "4096", "链接聊天室的端口号")
}

// 测试 NumberOfUser 个用户，测试并发
func main() {
	flag.Parse()
	var cnt atomic.Int32

	for i := 0; i < NumberOfUser; i++ {
		go func(ix int) {
			localPort := strconv.Itoa(ix + 20000)
			localAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%s", localPort))
			remoteAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", serverIp, serverPort))
			conn, err := net.DialTCP("tcp", localAddr, remoteAddr)
			utils.CheckError(err, "Client connect")
			MsgContext := generateRandomString(16)
			randv := rand.Intn(2)
			if randv == 0 { // 私聊
				conn.Write([]byte(fmt.Sprintf("%d|%s|", randv, conn.LocalAddr().String(), MsgContext)))
			} else { // 广播
				conn.Write([]byte(fmt.Sprintf("%d|%s", randv, MsgContext)))
			}
			log.Printf("第%d个用户发送消息成功, 端口为:%v\n", ix, localPort)
			cnt.Add(1)
			//time.Sleep(100 * time.Millisecond)
			//conn.Write([]byte(fmt.Sprintf("%d|%s", 4, MsgContext)))
		}(i)
	}
	time.Sleep(3 * time.Second)
	log.Println("一共测试连接数量为:", cnt.Load())
	time.Sleep(6 * 60 * time.Second)
}

// 生成指定长度的随机字符串
func generateRandomString(length int) string {
	tStr := "qwertyuioplkjhgfdsazxcvbnmQWERTYUIOPLKJHGFDSAZXCVBNM"
	var resStr string
	for i := 0; i < length; i++ {
		index := rand.Intn(len(tStr))
		resStr += string(tStr[index])
	}
	return resStr
}
