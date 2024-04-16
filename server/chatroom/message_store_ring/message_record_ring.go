package message_store_ring

import "chatroom/parameter"

// 一个环形的消息存储的数据结构
// 只有插入消息 AddCoverMsg 和返回一段消息 GetSeqMsg 两个公开方法
type MsgRecording struct {
	rIndex      int      // 当前存储消息的索引
	curSize     int      // 当前消息存储的大小（长度）
	maxCapacity int      // 整个环形消息数据结构消息存储的最大容量
	msgRing     []string // 消息存储内部的数据
}

func NewMsgRing() *MsgRecording {
	return &MsgRecording{
		rIndex:      0,
		curSize:     0,
		maxCapacity: parameter.RingMaxCapacity,
		msgRing:     make([]string, parameter.RingMaxCapacity),
	}
}

func (r *MsgRecording) isFull() bool {
	return r.curSize == r.maxCapacity
}

// 先插入，后增加Index,
func (r *MsgRecording) AddCoverMsg(msg string) {
	r.msgRing[r.rIndex] = msg
	r.rIndex = (r.rIndex + 1) % r.maxCapacity
	if !r.isFull() {
		r.curSize++
	}
}

// 从uIndex读取的消息，从r.index里返回消息
func (r *MsgRecording) GetSeqMsg(uIndex int) []string {
	if uIndex <= r.rIndex {
		return r.msgRing[uIndex:r.rIndex]
	} else {
		if r.isFull() {
			return append(r.msgRing[uIndex:r.maxCapacity], r.msgRing[0:r.rIndex]...)
		}
		panic("不应该出现环形数组没满，但user index 大于 ringIndex 的情况")
	}
}
