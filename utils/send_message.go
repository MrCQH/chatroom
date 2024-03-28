package utils

import (
	"fmt"
	"net"
)

// 将 msg 发送给对应的 conn
func SendMessage(conn net.Conn, msg string) {
	_, err := conn.Write([]byte(msg))
	CheckError(err, fmt.Sprintf("%s write", conn.RemoteAddr()))
}
