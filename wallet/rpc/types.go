package rpc

import (
	"encoding/json"
)

//统一输出结构
type Result struct {
	Result interface{} `json:"result"`
}

//详情
type Getinfo struct {
	Balance       uint64 `json:"balance"`
	Testnet       bool   `json:"testnet"`
	Blocks        uint64 `json:"blocks"`
	Group         uint64 `json:"group"`         //区块组高度
	StartingBlock uint64 `json:"StartingBlock"` //区块开始高度
	HighestBlock  uint64 `json:"HighestBlock"`  //所链接的节点的最高高度
	CurrentBlock  uint64 `json:"CurrentBlock"`  //已经同步到的区块高度
	PulledStates  uint64 `json:"PulledStates"`  //正在同步的区块高度
}

//新地址
type GetNewAddress struct {
	Address string `json:"address"`
}

//帐号余额
type GetAccount struct {
	Balance float64 `json:"Balance"`
}

//成功
//type Success struct {
//	Success string `json:"Success"`
//}

func tojson(data interface{}) ([]byte, error) {
	res, err := json.Marshal(Result{Result: data})
	return res, err
}
