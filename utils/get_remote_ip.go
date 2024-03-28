package utils

import (
	"net"
	"strings"
)

func GetRemoteIp(conn net.Conn) string {
	return strings.Split(conn.RemoteAddr().String(), ":")[0]
}
