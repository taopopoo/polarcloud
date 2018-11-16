package main

import (
	"encoding/hex"
	"fmt"
	"yunpan/core/utils"
	"time"
)

/*
	map占用内存，10万key占用20M，100W占用199M
*/
func main() {
	data := make(map[string]int)
	fmt.Println("start")
	time.Sleep(time.Second * 20)
	//	time

	for i := 0; i < 1000000; i++ {
		time.Sleep(time.Nanosecond)
		one := hex.EncodeToString(utils.GetHashForDomain(time.Now().Format("2006-01-02 15:04:05.999999999")))
		data[one] = 0
	}
	fmt.Println("end", len(data))
	time.Sleep(time.Second * 60)
}
