package utils

import (
	"io"
	"log"
)

// 错误处理方法
func CheckError(err error, info string, closers ...io.Closer) bool {
	if err != nil && err != io.EOF {
		defer func() {
			for _, closer := range closers {
				closer.Close()
			}
		}()
		log.Println(info, ":", err)
		return true
	}
	return false
}
