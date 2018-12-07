package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"polarcloud/wallet/mining"
)

var prk *ecdsa.PrivateKey
var coinbase *keystore.Address

func Register() {
	mining.RegisteMSG()

	_, err := keystore.Load()
	if err != nil {
		//TODO 应该让用户主动生成
		fmt.Println("开始生成keystory")
		bs := utils.Hash_SHA3_256([]byte(utils.GetRandomDomain()))
		err := keystore.NewLoad(hex.EncodeToString(bs), "123456")
		if err != nil {
			fmt.Println(err)
		}
	}
	//	keystore.GetAddr()
	addrs := keystore.GetAddr()
	if len(addrs) <= 0 {
		coinbase, err = keystore.GetNewAddr("123456")
	}
	addr, err := keystore.GetCoinbase()
	//fmt.Println("xxxx", addr, err)
	if err != nil {
		fmt.Println("获取见证人地址失败", err)
		return
	}
	fmt.Println("coinbase", addr.Hash.B58String())

	//	BuildPrkSead()

	fmt.Println("------------123--------------")

	//检查目录是否存在，不存在则创建
	utils.CheckCreateDir(filepath.Join(config.Wallet_path))

	err = db.InitDB(config.DB_path)
	if err != nil {
		panic(err)
	}

	//如果是创世节点，不用同步区块，直接开始挖矿
	if config.InitNode {
		mining.SetHighestBlock(1)
	}

	//当本地数据库为空时，需要先同步第一个区块，这个只有初始3个矿工需要这个操作

	//	fmt.Println("检查区块是否合法")
	//检查区块是否被篡改，中间是否有不连续的块。
	//	mining.CheckBlockDB()

	fmt.Println("开始加载数据库中的区块")
	//加载数据库中的区块
	err = mining.LoadBlockChain()
	if err != nil {
		fmt.Println("加载数据库中的区块错误", err)
	}
	fmt.Println("完成加载数据库中的区块")

	//如果是创世节点，不用同步区块
	if config.InitNode {
		return
	}

	//一边同步块，一边加载新块
	//开始同步区块
	err = mining.SyncBlockHead()
	if err != nil {
		fmt.Println("同步区块错误", err)
		return
	}
	fmt.Println("节点启动完成")
	//	mining.Mining()

}
