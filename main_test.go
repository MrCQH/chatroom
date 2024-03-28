package main

import (
	"chatroom/utils"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"testing"
)

// 测试 NumberOfUser 个用户，测试并发
func TestSever(t *testing.T) {
	go main()
	const NumberOfUser = 10000
	for i := 0; i < NumberOfUser; i++ {
		go func() {
			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", serverIp, serverPort))
			utils.CheckError(err, "Client connect")
			randv := rand.Intn(2)
			context, _ := generateRandomString(16)
			if randv == 0 { // 私聊
				conn.Write([]byte(fmt.Sprintf("%d|%s|", randv, conn.LocalAddr().String(), context)))
			} else { // 广播
				conn.Write([]byte(fmt.Sprintf("%d|%s", randv, context)))
			}
		}()
	}
}

// 生成指定长度的随机字符串 by GPT
func generateRandomString(length int) (string, error) {
	// 计算生成随机字符串所需的字节数
	byteLength := (length * 6) / 8 // base64编码后的字节数

	// 创建一个字节数组来存储随机数据
	randBytes := make([]byte, byteLength)

	// 使用crypto/rand包生成随机数据填充数组
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", err
	}

	// 将随机字节流进行base64编码
	randomString := base64.URLEncoding.EncodeToString(randBytes)

	// 截取指定长度的随机字符串
	return randomString[:length], nil
}
