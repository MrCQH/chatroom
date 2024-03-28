package main

import (
	"chatroom/server"
	"flag"
)

var serverIp string
var serverPort string

func init() {
	flag.StringVar(&serverIp, "i", "127.0.0.1", "聊天室的IP地址")
	flag.StringVar(&serverPort, "p", "12345", "聊天室的端口号")
}

func main() {
	flag.Parse()
	//log.Println("serverIp:", serverIp, "serverPort:", serverPort)
	chatServer := server.NewChatServer(serverIp, serverPort)
	chatServer.Start()
}
