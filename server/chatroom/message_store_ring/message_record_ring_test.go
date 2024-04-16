package message_store_ring

import (
	"fmt"
	"strconv"
	"testing"
)

func TestMsgRing(t *testing.T) {
	r := NewMsgRing()
	for i := 0; i < 70; i++ {
		r.AddCoverMsg(strconv.Itoa(i))
	}
	fmt.Println(r.GetSeqMsg(21))
}
