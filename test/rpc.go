package main

import (
	"fmt"
	"polarcloud/rpc"
)

func main() {
	fmt.Println("start...")
	rpc.RegisterRpcServer()
}
