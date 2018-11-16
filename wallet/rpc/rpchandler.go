package rpc

import (
	"fmt"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
	"polarcloud/wallet/mining"
)

type serverHandler func(rj *rpcJson) ([]byte, error)

//访问接口，统一header里传username,password
var rpcHandler = map[string]serverHandler{
	"getinfo":         handleGetInfo,         //获取基本信息{"method":"getinfo","params":{"id":10}}
	"getnewaddress":   handleGetNewAddress,   //创建新地址 {"method":"getnewaddress","params":{"password":"123456"}}
	"listaccounts":    handleListAccounts,    //帐号列表{"method":"listaccounts"}
	"getaccount":      handleGetAccount,      //获取某一帐号余额{"method":"getaccount","params":{"address":"1AX9mfCRZkdEg5Ci3G5SLcyGgecj6GTzLo"}}
	"validateaddress": handleValidateAddress, //验证地址{"method":"validateaddress","params":{"address":"12EUY1EVnLJe4Ejb1VaL9NbuDQbBEV"}}
	"sendtoaddress":   sendToAddress,
}

//获取基本信息
//{
//    "jsonrpc": "2.0",
//    "code": 2000,
//    "result": {
//        "balance": 0,
//        "testnet": false,
//        "blocks": 0
//    }
//}
func handleGetInfo(rj *rpcJson) ([]byte, error) {
	value := mining.GetBalance()

	info := Getinfo{
		Balance:       value,
		Testnet:       true,
		Blocks:        mining.GetCurrentBlock(),
		Group:         mining.GetGroupHeight(),
		StartingBlock: mining.GetStartingBlock(), //区块开始高度
		HighestBlock:  mining.GetHighestBlock(),  //所链接的节点的最高高度
		CurrentBlock:  mining.GetCurrentBlock(),  //已经同步到的区块高度
		PulledStates:  mining.GetPulledStates(),  //正在同步的区块高度
	}
	res, err := tojson(info)
	return res, err
}

//创建新地址
//{
//    "jsonrpc": "2.0",
//    "code": 2000,
//    "result": {
//        "address": "12Hixu5fzDrVoQt1fDL5vHw2Aahw1q"
//    }
//}
func handleGetNewAddress(rj *rpcJson) (res []byte, err error) {
	if !rj.VerifyType("password", "string") {
		res, err = errcode(TypeWrong, "password")
		return
	}
	password, ok := rj.Get("password")
	if !ok {
		res, err = errcode(NoField, "password")
		return
	}
	addr, err := keystore.GetNewAddr(password.(string))
	if err != nil {
		res, _ = errcode(Nomarl)
		return
	}
	getnewadress := GetNewAddress{Address: addr.Hash.B58String()}
	res, err = tojson(getnewadress)
	return
}

//帐号列表
//{
//    "jsonrpc": "2.0",
//    "code": 2000,
//    "result": {
//        "12FRzz2xrVtEm9cwzgFArrLE7VA7ks": 0,
//        "12GJJknncS2MmbXh26ZHAMbd3CjCHy": 0,
//        "12Hixu5fzDrVoQt1fDL5vHw2Aahw1q": 0
//    }
//}
func handleListAccounts(rj *rpcJson) (res []byte, err error) {
	list := make(map[string]float64)
	addr := keystore.GetAddr()
	for _, val := range addr {
		list[val.Hash.B58String()] = float64(mining.GetBalanceForAddr(val.Hash)) / float64(mining.Unit)
	}
	res, err = tojson(list)
	return res, err
}

//获取某一帐号余额
//{
//    "jsonrpc": "2.0",
//    "code": 2000,
//    "result": {
//        "Balance": 0
//    }
//}
func handleGetAccount(rj *rpcJson) (res []byte, err error) {
	if !rj.VerifyType("address", "string") {
		res, err = errcode(TypeWrong, "address")
		return
	}
	addr, ok := rj.Get("address")
	if !ok {
		res, err = errcode(NoField, "address")
		return
	}
	fmt.Println(addr)
	getaccount := GetAccount{}
	res, err = tojson(getaccount)
	return
}

//验证地址
//{
//    "jsonrpc": "2.0",
//    "code": 2000,
//    "result": {
//        "IsVerify": true,
//        "IsMine": false,
//        "IsType": 1,
//        "Version": 0,
//        "ExpVersion": 0,
//        }
//    }
//}
func handleValidateAddress(rj *rpcJson) (res []byte, err error) {
	if !rj.VerifyType("address", "string") {
		res, err = errcode(TypeWrong, "address")
		return
	}
	addr, ok := rj.Get("address")
	if !ok {
		res, err = errcode(NoField, "address")
		return
	}
	vali := keystore.ValidateByAddress(addr.(string))
	res, err = tojson(vali)
	return
}

func sendToAddress(rj *rpcJson) (res []byte, err error) {
	addrItr, ok := rj.Get("address")
	if !ok {
		res, err = errcode(5002, "address")
		return
	}
	addr := addrItr.(string)
	amountItr, ok := rj.Get("amount")
	if !ok {
		res, err = errcode(5002, "amount")
		return
	}
	amount := uint64(amountItr.(float64))

	commentItr, ok := rj.Get("comment")
	if !ok {
		res, err = errcode(5002, "comment")
		return
	}
	comment := commentItr.(string)
	fmt.Println("转账到地址", addr, amount, comment)

	dst, e := utils.FromB58String(addr)
	if err != nil {
		err = e
		res, _ = errcode(5003, "error")
		return
	}

	txpay, e := mining.SendToAddress(&dst, amount, comment)
	if err != nil {
		err = e
		res, _ = errcode(5003, "error")
		return
	}
	res, err = tojson(txpay)
	return
}
