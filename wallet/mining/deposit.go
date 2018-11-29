/*
	管理缴纳押金
*/
package mining

import (
	"fmt"
	"polarcloud/config"
	"polarcloud/wallet/keystore"
)

//var haveDeposit uint32 = 1 //是否交了押金 1=已交;2=未交;
//func SetHaveDepositYes() {
//	atomic.StoreUint32(&haveDeposit, 2)
//}
//func SetHaveDepositNo() {
//	atomic.StoreUint32(&haveDeposit, 1)
//}

//func GetHaveDeposit() bool {
//	ok := atomic.LoadUint32(&haveDeposit)
//	if ok == 1 {
//		return true
//	} else {
//		return false
//	}
//}

/*
	交押金
*/
func Deposit() {
	if !config.Miner {
		return
	}

	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}

	//交了押金就不再交了
	if hashWitness(addr.Hash) {
		fmt.Println("交了押金就不用再交了")
		return
	}

	//缴纳备用见证人押金交易
	err = chain.witnessChain.PayDeposit(config.Mining_deposit)
	if err != nil {
		fmt.Println("缴纳押金失败", err)
	}
	fmt.Println("缴纳押金完成")

	//判断自己的押金交易是否被打包到块中，已打包则给自己投票
	//	chain.CheckVote(addr)
	//	fmt.Println("投票完成")
}
