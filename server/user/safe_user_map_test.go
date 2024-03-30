package user

import (
	"fmt"
	"testing"
)

func TestNewSafeUserMap(t *testing.T) {
	safeUserMap := NewSafeUserMap()
	fmt.Println("safeUserMap:", safeUserMap)
	safeUserMap.SetUser("nihao", NewUser("1", "2", "3", nil, nil))
	fmt.Println("safeUserMap set:", safeUserMap)
	user, _ := safeUserMap.GetUser("nihao")
	deleteUser, _ := safeUserMap.DeleteUser("nihao")
	fmt.Println("user:", user)
	fmt.Println("deleteUser:", deleteUser)
	fmt.Println("safeUserMap:", safeUserMap)
}
