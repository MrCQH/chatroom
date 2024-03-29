package user

import (
	"log"
	"sync"
	"sync/atomic"
)

// 封装的sync.Map，其key为userName string ,value为User *User
type SafeUserMap struct {
	m   *sync.Map
	cnt atomic.Int64 //计数器
}

func (sm *SafeUserMap) GetUser(username string) (*User, bool) {
	u, isPresent := sm.m.Load(username)
	log.Println(isPresent)
	if !isPresent {
		return nil, false
	}
	user, success := u.(*User)
	if !success {
		panic("转化失败，这个值不是*User类型")
	}
	return user, true
}

func (sm *SafeUserMap) SetUser(username string, user *User) (*User, bool) {
	us, isPresent := sm.m.LoadOrStore(username, user)
	u, success := us.(*User)
	if !success {
		panic("转化失败，这个值不是*User类型")
	}
	if isPresent {
		return u, false
	}
	sm.cnt.Add(1)
	return u, true
}

func (sm *SafeUserMap) DeleteUser(username string) (*User, bool) {
	u, isPresent := sm.m.LoadAndDelete(username)
	if !isPresent {
		return nil, false
	}
	user, success := u.(*User)
	if !success {
		panic("转化失败，这个值不是*User类型")
	}
	sm.cnt.Add(-1)
	return user, true
}

func (sm *SafeUserMap) Len() int {
	return int(sm.cnt.Load())
}

func (sm *SafeUserMap) Range(fn func(key any, value any) bool) {
	sm.m.Range(fn)
}
