package mining

import (
	"fmt"
	"sync/atomic"
	"yunpan/config"
	"yunpan/wallet/db"
)

/*
	节点启动前，检查数据库中的区块是否合法，区块是否损坏、篡改、连续
	1.先从高度由低到高检查区块头
	2.再从高度由高到低检查交易合法性
	@return    start    起始块高度
	@return    end      数据库中最高块高度
*/
func CheckBlockDB() (start, end uint64) {
	defer func() {
		atomic.StoreUint64(&chain.StartingBlock, start)
		atomic.StoreUint64(&chain.CurrentBlock, end)
	}()

	headid, err := db.Find(config.Key_block_start)
	if err != nil {
		//认为这是一个空数据库
		fmt.Println("这是一个空数据库")
		return
	}

	head, err := db.Find(*headid)
	if err != nil {
		fmt.Println("1111", err)
		return
	}

	hB, err := ParseBlockHead(head)
	if err != nil {
		fmt.Println("2222", err)
		return
	}
	start = hB.Height
	end = hB.Height

	for {
		if hB.Nextblockhash == nil {
			break
		}
		head, err = db.Find(hB.Nextblockhash[0])
		if err != nil {
			//数据库中的区块头查找错误，需要重新下载区块
			break
		}
		hB, err = ParseBlockHead(head)
		if err != nil {
			//数据库中的区块头解析错误，需要重新下载区块
			break
		}
		if !hB.Check() {
			break
		}
		start = hB.Height
		end = hB.Height

	}

	return
}

/*
	启动第一次加载区块
	从数据库中加载区块
	先找到起始区块，从区块高度由低到高开始加载
*/
func LoadBlockChain() error {
	headid, err := db.Find(config.Key_block_start)
	if err != nil {
		//认为这是一个空数据库
		fmt.Println("这是一个空数据库")
		return nil
	}

	bh, err := loadBlockForDB(headid)
	if err != nil {
		return err
	}
	SetStartingBlock(bh.Height)
	if bh.Nextblockhash == nil || len(bh.Nextblockhash) == 0 {
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
	bh, err := loadBlockForDB(bhash)
	if err != nil {
		return
	}
	if bh.Nextblockhash != nil {
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
func loadBlockForDB(bhash *[]byte) (*BlockHead, error) {
	head, err := db.Find(*bhash)
	if err != nil {
		return nil, err
	}
	hB, err := ParseBlockHead(head)
	if err != nil {
		return nil, err
	}
	//	fmt.Println("CurrentBlock保存的高度", hB.Height)
	txItrs := make([]TxItr, 0)
	for _, one := range hB.Tx {
		txBs, err := db.Find(one)
		if err != nil {
			fmt.Println("3333", err)
			return nil, err
		}
		txItr, err := ParseTxBase(txBs)
		txItrs = append(txItrs, txItr)
	}
	chain.AddBlock(hB, &txItrs)
	return hB, nil
}
