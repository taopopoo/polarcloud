package main

import (
	"fmt"
	"yunpan/rpc"
)

func main() {
	fmt.Println("start...")
	rpc.RegisterRpcServer()
}
