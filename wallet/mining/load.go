package mining

import (
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"polarcloud/wallet/db"
)

/*
	节点启动前，检查数据库中的区块是否合法，区块是否损坏、篡改、连续
	1.先从高度由低到高检查区块头
	2.再从高度由高到低检查交易合法性
	@return    start    起始块高度
	@return    end      数据库中最高块高度
*/
func CheckBlockDB() bool {

	headid, err := db.Find(config.Key_block_start)
	if err != nil {
		//认为这是一个空数据库
		fmt.Println("这是一个空数据库")
		return true
	}

	bs, err := db.Find(*headid)
	if err != nil {
		fmt.Println("1111", err)
		return false
	}

	hB, err := ParseBlockHead(bs)
	if err != nil {
		fmt.Println("2222", err)
		return false
	}

	for {
		if hB.Nextblockhash == nil {
			fmt.Println("没有下一个块了")
			break
		}
		fmt.Println("开始验证下一个区块", hB.Height+1)
		bs, err = db.Find(hB.Nextblockhash[0])
		if err != nil {
			//数据库中的区块头查找错误，需要重新下载区块
			fmt.Println("区块头查找错误", hB.Height+1, hex.EncodeToString(*bs))
			return false
		}
		hB, err = ParseBlockHead(bs)
		if err != nil {
			//数据库中的区块头解析错误，需要重新下载区块
			fmt.Println("本区块解析错误", hB.Height)
			return false
		}
		if !hB.Check() {
			fmt.Println("本区块不合法", hB.Height)
			return false
		}
		//检查交易是否正确
		for _, one := range hB.Tx {
			bs, err = db.Find(one)
			if err != nil {
				//数据库中的交易查找错误，需要重新下载
				fmt.Println("查找交易错误", hB.Height, hex.EncodeToString(*bs))
				return false
			}
			txItr, err := ParseTxBase(bs)
			if err != nil {
				fmt.Println("解析交易错误", hB.Height, hex.EncodeToString(*bs))
				return false
			}
			if !txItr.Check() {
				fmt.Println("验证交易失败，交易不合法")
				return false
			}
		}
	}
	return true
}

/*
	从数据库中加载区块
	先找到内存中最高区块，从区块由低到高开始加载
*/
func LoadBlockChain() error {

	fmt.Println("----开始加载区块到内存")
	var bh *BlockHead
	var txItrs []TxItr
	chain := forks.GetLongChain()
	if chain == nil {
		//首次同步区块
		var err error
		headid, err := db.Find(config.Key_block_start)
		if err != nil {
			//认为这是一个空数据库
			fmt.Println("这是一个空数据库")
			return nil
		}
		bh, txItrs, err = loadBlockForDB(headid)
		if err != nil {
			return err
		}
		fmt.Println("设置起始区块高度", bh.Height)
		SetStartingBlock(bh.Height)
		forks.AddBlock(bh, &txItrs)
	} else {
		var err error
		headid := &chain.GetLastBlock().Id
		bh, txItrs, err = loadBlockForDB(headid)
		if err != nil {
			return err
		}
	}
	//	chain.AddBlock(bh, &txItrs)

	if bh.Nextblockhash == nil || len(bh.Nextblockhash) == 0 {
		//		fmt.Println("因Nextblockhash为空退出")
		return nil
	}
	for i, _ := range bh.Nextblockhash {
		deepCycleLoadBlock(&bh.Nextblockhash[i])
	}
	return nil
}

/*
	深度循环加载区块，包括分叉的链的加载
	加载到出错或者加载完成为止
*/
func deepCycleLoadBlock(bhash *[]byte) {
	bh, txItrs, err := loadBlockForDB(bhash)
	if err != nil {
		return
	}
	forks.AddBlock(bh, &txItrs)
	//	chain.AddBlock(bh, &txItrs)
	if bh.Nextblockhash == nil {
		return
	}
	for i, _ := range bh.Nextblockhash {
		deepCycleLoadBlock(&bh.Nextblockhash[i])
	}
	return
}

/*
	从数据库中加载一个区块
*/
func loadBlockForDB(bhash *[]byte) (*BlockHead, []TxItr, error) {
	head, err := db.Find(*bhash)
	if err != nil {
		return nil, nil, err
	}
	hB, err := ParseBlockHead(head)
	if err != nil {
		return nil, nil, err
	}
	txItrs := make([]TxItr, 0)
	for _, one := range hB.Tx {
		txBs, err := db.Find(one)
		if err != nil {
			fmt.Println("3333", err)
			return nil, nil, err
		}
		txItr, err := ParseTxBase(txBs)
		txItrs = append(txItrs, txItr)
	}

	return hB, txItrs, nil
}
