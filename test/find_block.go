package main

import (
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"polarcloud/wallet/db"
	"polarcloud/wallet/mining"
)

func main() {
	find("D:/temp/root/wallet/data")
	fmt.Println("finish!")
}

func find(dir string) {
	nums := []uint64{}
	for i := uint64(1); i < 1000; i++ {
		nums = append(nums, i)
	}

	db.InitDB(dir)
	beforBlockHash, err := db.Find(config.Key_block_start)
	if err != nil {
		fmt.Println("111 查询起始块id错误", err)
		return
	}
	maxBlock := uint64(0)
	for _, one := range nums {
		if one > maxBlock {
			maxBlock = one
		}
	}

	for i := uint64(1); i <= maxBlock; i++ {
		bs, err := db.Find(*beforBlockHash)
		if err != nil {
			fmt.Println("查询第", i, "个块错误", err)
			return
		}
		bh, err := mining.ParseBlockHead(bs)
		if err != nil {
			fmt.Println("解析第", i, "个块错误", err)
			return
		}
		beforBlockHash = &bh.Nextblockhash[0]
		isPrint := false
		for _, one := range nums {
			if one == i {
				isPrint = true
				break
			}
		}
		if isPrint {
			fmt.Println("第", i, "个块\n", hex.EncodeToString(bh.Hash), "\n", string(*bs), "\n")
			for _, one := range bh.Tx {
				tx, err := db.Find(one)
				if err != nil {
					fmt.Println("查询第", i, "个块的交易错误", err)
					return
				}
				txBase, err := mining.ParseTxBase(tx)
				if err != nil {
					fmt.Println("解析第", i, "个块的交易错误", err)
					return
				}

				txid := txBase.GetHash()
				//				if txBase.Class() == config.Wallet_tx_type_deposit_in {
				//					deposit := txBase.(*mining.Tx_deposit_in)
				//					txid = deposit.Hash
				//				}
				fmt.Println(string(hex.EncodeToString(*txid)), "\n", string(*tx), "\n")
			}
		}
	}
}
