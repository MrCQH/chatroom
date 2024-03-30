package main

import (
	"chatroom/server/server"
	"flag"
)

var serverIp string   // 聊天室的IP地址
var serverPort string // 聊天室的端口号

func init() {
	flag.StringVar(&serverIp, "i", "127.0.0.1", "聊天室的IP地址")
	flag.StringVar(&serverPort, "p", "4096", "聊天室的端口号")
}

func main() {
	flag.Parse()
	chatServer := server.NewChatServer(serverIp, serverPort)
	chatServer.Start()
}
