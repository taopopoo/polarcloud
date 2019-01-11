package mining

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"sync"
	"sync/atomic"

	//	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	//	"github.com/syndtr/goleveldb/leveldb"
)

//同步保存区块头队列
var syncSaveBlockHead = make(chan *BlockHeadVO, 1)

//保存区块不连续信号
var syncForNeighborChain = make(chan bool, 1)

//var syncHeightBlock = new(sync.Map)
//var heightBlockGroup = new(sync.WaitGroup)

func init() {
	//	go saveBlockHead()

	go func() {
		for range syncForNeighborChain {
			LoadBlockChain()
		}
	}()
}

/*
	通知加载区块到内存的信号
*/
func NoticeLoadBlockForDB() {
	select {
	case syncForNeighborChain <- true:
	default:
	}
}

/*
	添加一个区块到添加队列中去
*/
func AddBlockHead(bhvo *BlockHeadVO) {
	//TODO 检查本节点是否是挖矿节点
	//别人先出块，自己停止出块
	stopFindNonce(bhvo.BH)

	ok := saveBlockHead(bhvo)
	if !ok {
		fmt.Println("------添加一个区块不连续")
		//区块不连续有两种情况
		//1.恶意区块
		//2.本节点未同步完整所有区块
		if (GetHighestBlock() <= 0 || GetHighestBlock() <= 0) ||
			GetHighestBlock() > GetCurrentBlock() {
			//更新网络广播块高度
			if bhvo.BH.Height > GetHighestBlock() {
				atomic.StoreUint64(&forks.HighestBlock, bhvo.BH.Height)
			}
			//同步内存，从数据库加载到内存
			NoticeLoadBlockForDB()
		}
		//TODO
		//恶意区块，或者链已经分叉
		//如果自己所在链长期不更新，则自己处在小众分叉链上

		return
	}
	//更新网络广播块高度
	if bhvo.BH.Height > GetHighestBlock() {
		atomic.StoreUint64(&forks.HighestBlock, bhvo.BH.Height)
	}

	//添加到内存
	forks.AddBlock(bhvo.BH, &bhvo.Txs)

	forks.SelectLongChain()

	//判断节点和内存不同步，重新加载区块到内存
	if bhvo.BH.Height > GetCurrentBlock() {
		//同步内存，从数据库加载到内存
		NoticeLoadBlockForDB()
	}

	//	if chain == nil {

	//		return
	//	} else {
	//		//统计余额，备用见证人，以及备用见证人投票
	//		//		chain.AddBlock(bhvo.BH, &bhvo.Txs)
	//		fmt.Println("添加一个区块", bhvo.BH.Height, hex.EncodeToString(bhvo.BH.Hash))
	//		ok, hs := forks.ContrastLongBlock(chain)
	//		if ok {
	//			if hs == nil {
	//				fmt.Println("出错了,分叉了，分叉路径却为空")
	//			} else {
	//				//分叉了，回滚
	//				fmt.Println("分叉了，需要回滚区块")
	//			}
	//		} else {
	//			//未分叉
	//		}
	//	}

}

/*
	保存一个区块，返回这个区块是否连续，判断依据是能够找到并修改前置区块
	@return    bool    是否连续
*/
func saveBlockHead(bhvo *BlockHeadVO) (ok bool) {
	ok = true
	//保存区块中的交易
	for i, _ := range bhvo.Txs {
		//			fmt.Println("改变前", hex.EncodeToString(*bhvo.Txs[i].GetHash()))
		bhvo.Txs[i].BuildHash()
		bhvo.Txs[i].SetBlockHash(bhvo.BH.Hash)
		//			fmt.Println("改变后", hex.EncodeToString(*bhvo.Txs[i].GetHash()))
		bs, err := bhvo.Txs[i].Json()
		if err != nil {
			//TODO 严谨的错误处理
			fmt.Println("严重错误1", err)
			ok = false
			return
		}
		db.Save(*bhvo.Txs[i].GetHash(), bs)

		//		for _, two := range *one.GetVin() {
		//			txbs, err := db.Find(two.Txid)
		//			if err != nil {
		//				//TODO 区块未同步完整，查找不到交易
		//				continue
		//			}
		//			txItr, err := ParseTxBase(txbs)
		//			if err != nil {
		//				fmt.Println("严重错误3", err)
		//				return
		//			}
		//			//这个交易输出被使用之后，需要把UTXO输出标记下
		//			err = txItr.SetTxid(txbs, two.Vout, one.GetHash())
		//			if err != nil {
		//				fmt.Println("严重错误4", err)
		//				return
		//			}
		//		}
	}
	//		fmt.Println("222222222222222")

	//修改前一个区块的next
	bs, err := db.Find(bhvo.BH.Previousblockhash[0])
	if err == nil {
		bh, err := ParseBlockHead(bs)
		if err != nil {
			fmt.Println("查询前置区块错误", err)
			ok = false
		}
		if bh.Nextblockhash == nil {
			bh.Nextblockhash = make([][]byte, 0)
		}
		bh.Nextblockhash = append(bh.Nextblockhash, bhvo.BH.Hash)
		bs, err = bh.Json()
		if err != nil {
			fmt.Println("格式化前置区块错误", err)
			ok = false
		}
		err = db.Save(bh.Hash, bs)
		if err != nil {
			//TODO 严谨的错误处理
			fmt.Println("保存前置区块错误", err)
			ok = false
		}
	}

	//保存区块
	bs, err = bhvo.BH.Json()
	if err != nil {
		//TODO 严谨的错误处理
		fmt.Println("格式化新区块错误", err)
		ok = false
	}
	err = db.Save(bhvo.BH.Hash, bs)
	if err != nil {
		//TODO 严谨的错误处理
		fmt.Println("保存新区块错误", err)
		ok = false
	}

	//		if bhvo.BH.Height == atomic.LoadUint64(&forks.StartingBlock) {
	//			db.Save(config.Key_block_start, &bhvo.BH.Hash)
	//		}
	return
}

/*
	查询邻居节点区块高度
	从邻居节点中查找最高区块高度
*/
func FindBlockHeight() {
	syncHeightBlock := new(sync.Map)

	//	heightBlockGroup = new(sync.WaitGroup)
	//	count := 0

	for _, key := range nodeStore.GetAllNodes() {
		mhead := mc.NewMessageHead(key, key, false)
		mbody := mc.NewMessageBody(nil, "", nil, 0)
		message := mc.NewMessage(mhead, mbody)
		message.BuildHash()

		session, ok := engine.GetSession(key.B58String())
		if ok {
			err := session.Send(config.MSGID_heightBlock, mhead.JSON(), mbody.JSON(), false)
			if err == nil {
				//				heightBlockGroup.Add(1)
				//				count = count + 1

				bs := mc.WaitRequest(mc.CLASS_findHeightBlock, message.Body.Hash.B58String())
				//		fmt.Println("有消息返回了啊")
				if bs == nil {
					fmt.Println("11111 发送共享文件消息失败，可能超时")
					continue
				}

				//				startHeight := binary.LittleEndian.Uint64((*bs)[:8])
				heightBlock := binary.LittleEndian.Uint64((*bs)[8:])
				//收到的区块高度比自己低，则不保存
				if GetCurrentBlock() > heightBlock {
					continue
				}

				syncHeightBlock.Store(session.GetName(), heightBlock)

			}
		}
	}
	//以下情况都返回
	//1.没有邻居节点。
	//2.查询邻居节点全部失败。
	//3.邻居节点都未同步完成。
	count := 0
	syncHeightBlock.Range(func(key, value interface{}) bool {
		count++
		return false //只要有数据就够了
	})
	if count <= 0 {
		return
	}

	//	done := make(chan bool, 1)
	//	go func() {
	//		heightBlockGroup.Wait()
	//		done <- true
	//	}()

	//	//超时时间
	//	overtime := time.NewTimer(time.Second * 5)

	//	select {
	//	case <-done:
	//		overtime.Stop()
	//	case <-overtime.C:
	//		for i := 0; i < count; i++ {
	//			heightBlockGroup.Done()
	//		}
	//	}

	//统计区块高度结果，给结果投票
	heightBlockVote := new(sync.Map)
	syncHeightBlock.Range(func(key, value interface{}) bool {
		//		fmt.Println("统计投票", key, value)
		height := value.(uint64)
		v, ok := heightBlockVote.Load(height)
		if ok {
			total := v.(uint64)
			heightBlockVote.Store(height, uint64(total+1))
		} else {
			heightBlockVote.Store(height, uint64(1))
		}
		return true
	})

	//查看区块高度投票结果，采用票数高的，票数都一样，采用区块高度最高的。
	heightBlockMax := uint64(0) //区块最高高度
	heightBlock := uint64(0)    //票数最高的区块高度
	heightTotal := uint64(0)    //最高票数
	isEqual := false            //最高票数是否有不同的区块高度
	heightBlockVote.Range(func(k, v interface{}) bool {
		//		fmt.Println("投票结果", k, v)
		height := k.(uint64)
		if height == 0 {
			return true
		}
		if height > heightBlockMax {
			heightBlockMax = height
		}
		total := v.(uint64)
		if total > heightTotal {
			heightTotal = total
			heightBlock = height
		} else if total == heightTotal {
			isEqual = true
		}
		return true
	})
	//TODO 考虑相同票数该选哪个
	//直接使用票数最多的区块高度
	//	atomic.StoreUint64(&chain.StartingBlock, 1)
	atomic.StoreUint64(&forks.HighestBlock, heightBlock)
	fmt.Println("收到的区块高度", heightBlock, "自己的高度", atomic.LoadUint64(&forks.CurrentBlock))

}

/*
	从邻居节点同步区块
*/
func SyncBlockHead() error {

	fmt.Println("+从邻居节点同步区块")
	//	var isStart bool //是否是起始区块
	var bhash *[]byte
	//获得本节点的最新块hash
	chain := forks.GetLongChain()
	//	lastBlock := forks.GetLongChain().GetLastBlock()
	if chain == nil {
		fmt.Println("+获取起始区块")
		//获得本节点的最新块失败，重新同步
		//从令居节点同步起始区块hash值
		bhash = FindStartBlockForNeighbor()
		if bhash == nil {
			return errors.New("同步起始区块hash失败")
		}
		fmt.Println("同步到的创世区块hash", hex.EncodeToString(*bhash))
		db.Save(config.Key_block_start, bhash)
	} else {
		bhash = &chain.GetLastBlock().Id
	}
	//最新块一定要去邻居节点拉取一次，更新next
	//	bhvo := FindBlockForNeighbor(bhash)
	bh := syncBlockFlashDB(bhash)
	if bh == nil {
		return nil
	}

	bh.BuildHash()
	fmt.Println("打印同步到的区块", hex.EncodeToString(bh.Hash))

	if bh.Nextblockhash == nil {
		return nil
	}

	for _, one := range bh.Nextblockhash {
		deepCycleSyncBlock(&one)
	}

	fmt.Println("同步区块完成")
	return nil
}

/*
	深度循环同步区块，包括分叉的链的同步
	加载到出错或者加载完成为止
*/
func deepCycleSyncBlock(bhash *[]byte) {
	fmt.Println("本次同步hash", hex.EncodeToString(*bhash))
	bh, err := syncBlockForDBAndNeighbor(bhash)
	if err != nil {
		return
	}
	fmt.Println("区块的next个数", len(bh.Nextblockhash), "高度", bh.Height)
	for _, one := range bh.Nextblockhash {
		deepCycleSyncBlock(&one)
	}
}

/*
	从数据库查询区块，如果数据库没有，从网络邻居节点查询区块
	查询到区块后，修改他们的指向hash值和UTXO输出的指向
*/
func syncBlockForDBAndNeighbor(bhash *[]byte) (*BlockHead, error) {
	//先查询数据库
	head, err := db.Find(*bhash)
	if err == nil {
		hB, err := ParseBlockHead(head)
		if err == nil {
			return hB, nil
		}
	}
	//再查找邻居节点
	bhvo := FindBlockForNeighbor(bhash)
	if bhvo == nil {
		return nil, errors.New("同步失败，未找到区块")
	}
	//保存区块中的交易
	for i, one := range bhvo.Txs {
		bhvo.Txs[i].BuildHash()
		bhvo.Txs[i].SetBlockHash(*bhash)
		bs, err := bhvo.Txs[i].Json()
		if err != nil {
			//TODO 严谨的错误处理
			fmt.Println("严重错误1", err)
			return nil, err
		}
		//			fmt.Println("保存交易", hex.EncodeToString(*bhvo.Txs[i].GetHash()))
		db.Save(*bhvo.Txs[i].GetHash(), bs)

		//将之前的UTXO输出标记为已使用
		//		if one.Class() != config.Wallet_tx_type_deposit_in &&
		//			one.Class() != config.Wallet_tx_type_pay {
		//			continue
		//		}

		for _, two := range *one.GetVin() {
			txbs, err := db.Find(two.Txid)
			if err != nil {
				//TODO 区块未同步完整可以查找不到交易
				return nil, err
			}
			txItr, err := ParseTxBase(txbs)
			if err != nil {
				fmt.Println("严重错误3", err)
				return nil, err
			}
			err = txItr.SetTxid(txbs, two.Vout, one.GetHash())
			if err != nil {
				fmt.Println("严重错误4", err)
				return nil, err
			}
		}
	}

	//先将前一个区块修改next
	if GetStartingBlock() > 0 {
		bs, err := db.Find(bhvo.BH.Previousblockhash[0])
		if err != nil {
			//TODO 区块未同步完整可以查找不到之前的区块
			return nil, err
		}
		bh, err := ParseBlockHead(bs)
		if err != nil {
			fmt.Println("严重错误5", err)
			return nil, err
		}
		if bh.Nextblockhash == nil {
			bh.Nextblockhash = make([][]byte, 0)
		}
		bh.Nextblockhash = append(bh.Nextblockhash, bhvo.BH.Hash)
		bs, err = bh.Json()
		if err != nil {
			fmt.Println("严重错误6", err)
			return nil, err
		}
		db.Save(bh.Hash, bs)
	}

	//保存区块
	bs, err := bhvo.BH.Json()
	if err != nil {
		//TODO 严谨的错误处理
		fmt.Println("严重错误7", err)
		return nil, err
	}
	db.Save(bhvo.BH.Hash, bs)

	return bhvo.BH, nil
}

/*
	同步区块并刷新本地数据库
*/
func syncBlockFlashDB(bhash *[]byte) *BlockHead {
	bhvo := FindBlockForNeighbor(bhash)
	if bhvo == nil {
		return nil
	}
	bhvo.BH.BuildHash()
	bs, err := bhvo.BH.Json()
	if err != nil {
		return nil
	}
	db.Save(*bhash, bs)
	for _, one := range bhvo.Txs {
		one.SetBlockHash(*bhash)
		bs, err := one.Json()
		if err != nil {
			return nil
		}
		db.Save(*one.GetHash(), bs)
	}
	return bhvo.BH
}
