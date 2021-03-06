package addr_manager

import (
	"fmt"
	"io/ioutil"
)

func init() {
	registerFunc(LoadSuperPeerEntry)
}

/*
	读取并解析本地的超级节点列表文件
*/
func LoadSuperPeerEntry() {
	fileBytes, err := ioutil.ReadFile(Path_SuperPeerAddress)
	if err != nil {
		fmt.Println("读取超级节点地址列表失败", err)
		return
	}
	parseSuperPeerEntry(fileBytes)
}
