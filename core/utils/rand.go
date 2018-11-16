package utils

import (
	"math/rand"
	"time"
)

/*
	字符串中随机选择一个
*/
func RandString(strs ...string) string {
	//	timens := int64(time.Now().Nanosecond())
	rand.Seed(int64(time.Now().Nanosecond()))
	r := rand.Intn(len(strs))
	return strs[r]
}
