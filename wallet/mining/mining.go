package mining

import (
	"bytes"
	"fmt"
	"log"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
	"time"
)

/*
	开始挖矿
	当每个组见证人选出来之后，启动挖矿程序，按顺序定时出块
*/
func Mining() {
	//判断是否同步完成
	if GetHighestBlock() <= 0 {
		fmt.Println("区块未同步完成，不能挖矿 GetHighestBlock", GetHighestBlock())
		return
	}
	if GetHighestBlock() > GetCurrentBlock() {
		fmt.Println("区块未同步完成，不能挖矿 GetCurrentBlock", GetCurrentBlock(), GetHighestBlock())
		return
	}
	if !config.Miner {
		return
	}

	fmt.Println("启动挖矿程序")

	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}
	//判断用什么方式出块
	if forks.GetLongChain().witnessChain.group == nil {
		//用工作量证明方式出块
		fmt.Println("用工作量证明方式出块")
		BuildBlockForPOW()
	} else {
		//用见证人方式出块
		fmt.Println("用见证人方式出块")
		//判断是否已经安排了任务
		if forks.GetLongChain().witnessChain.group.Task {
			fmt.Println("已经安排了任务，退出")
			return
		}
		forks.GetLongChain().witnessChain.group.Task = true

		//判断自己出块顺序的时间
		for i, one := range forks.GetLongChain().witnessChain.group.Witness {
			//自己是见证人才能出块，否则自己出块了，其他节点也不会承认
			if bytes.Equal(*one.Addr, *addr.Hash) {
				fmt.Println("多少秒钟后出块", config.Mining_block_time*(i+1))
				utils.AddTimetask(time.Now().Unix()+int64(config.Mining_block_time*(i+1)),
					TaskBuildBlock, Task_class_buildBlock, "")
			}
		}
	}

}

/*
	见证人方式出块
	出块并广播
	@gh    uint64    出块的组高度
	@id    []byte    押金id
*/
func BuildBlock() {
	var this *Witness
	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}

	//判断自己出块顺序的时间
	for _, one := range forks.GetLongChain().witnessChain.group.Witness {
		//自己是见证人才能出块，否则自己出块了，其他节点也不会承认
		if one.Addr.B58String() == addr.Hash.B58String() {
			this = one
			break
		}
	}

	//自己是见证人才能出块，否则自己出块了，其他节点也不会承认
	if this.Addr.B58String() != addr.Hash.B58String() {
		return
	}

	fmt.Println("===准备出块===")

	<-time.NewTimer(time.Second * 10).C

	chain := forks.GetLongChain()
	lastBlock := chain.GetLastBlock()

	blockHeight := lastBlock.Height + 1
	groupHieght := this.Group.Height

	if this.PreWitness != nil && this.PreWitness.Block != nil {
		fmt.Println("===准备出块=== 块高度", blockHeight, "组高度", groupHieght)
	}

	//打包交易
	tx := make([]TxItr, 0)
	txids := make([][]byte, 0)

	//打包10秒内的所有交易
	txs, ids := chain.transactionManager.Package()

	//判断是否是该组第一个块
	//判断上一个组是否是见证人方式出块，是见证人方式出块，计算上一组出块奖励。
	if chain.witnessChain.beforeGroup != nil &&
		chain.witnessChain.group.FirstWitness() {
		reward := chain.witnessChain.beforeGroup.CountReward(txs)
		tx = append(tx, reward)
		txids = append(txids, reward.Hash)
	}

	tx = append(tx, txs...)
	txids = append(txids, ids...)

	//准备块中的交易
	//	fmt.Println("准备块中的交易")
	coinbase, err := keystore.GetCoinbase()
	if err != nil {
		return
	}

	//开始生成块
	bh := BlockHead{
		Height:            blockHeight,                       //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       groupHieght,                       //矿工组高度
		Previousblockhash: [][]byte{chain.GetLastBlock().Id}, //上一个区块头hash
		NTx:               uint64(len(tx)),                   //交易数量
		Tx:                txids,                             //本区块包含的交易id
		Time:              time.Now().Unix(),                 //unix时间戳
		Witness:           *coinbase.Hash,                    //此块矿工地址
	}
	bh.BuildMerkleRoot()
	bh.BuildHash()
	bh.BuildSign(coinbase)

	bhvo := CreateBlockHeadVO(&bh, tx)

	//广播区块
	MulticastBlock(bhvo)

	fmt.Println("========出块完成=======")
	AddBlockHead(bhvo)

}

/*
	POW方式出块并广播
	@gh    uint64    出块的组高度
	@id    []byte    押金id
*/
func BuildBlockForPOW() {
	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}

	fmt.Println("===准备pow方式出块===")

	chain := forks.GetLongChain()
	lastBlock := chain.GetLastBlock()

	//打包交易
	txs := make([]TxItr, 0)
	txids := make([][]byte, 0)

	//打包10秒内的所有交易
	txss, ids := chain.transactionManager.Package()
	fmt.Println("打包的交易", len(txss))

	allGas := uint64(0)
	for _, one := range txss {
		allGas = allGas + one.GetGas()
	}

	//第一个块产出80个币
	//每增加一定块数，产出减半，直到为0
	//最多减半9次，第10次减半后产出为0
	oneReward := uint64(config.Mining_reward)
	n := (lastBlock.Height + 1) / config.Mining_block_cycle
	if n < 10 {
		for i := uint64(0); i < n; i++ {
			oneReward = oneReward / 2
		}
	} else {
		oneReward = 0
	}
	allReward := oneReward + allGas

	//构造出块奖励
	if allReward > 0 {
		vouts := make([]Vout, 0)
		vouts = append(vouts, Vout{
			Value:   allReward,  //输出金额 = 实际金额 * 100000000
			Address: *addr.Hash, //钱包地址
		})
		base := TxBase{
			Type:       config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
			Vout_total: uint64(len(vouts)),           //输出交易数量
			Vout:       vouts,                        //交易输出
			LockHeight: lastBlock.Height + 100,       //锁定高度
			//			CreateTime: time.Now().Unix(),            //创建时间
		}
		reward := Tx_reward{
			TxBase: base,
		}
		txs = append(txs, &reward)
		reward.BuildHash()
		txids = append(txids, reward.Hash)
	}

	//判断上一个组是否是见证人方式出块，是见证人方式出块，计算上一组出块奖励。
	if chain.witnessChain.beforeGroup != nil {
		reward := chain.witnessChain.beforeGroup.CountReward(txss)
		txs = append(txs, reward)
		txids = append(txids, reward.Hash)
	}

	txs = append(txs, txss...)
	txids = append(txids, ids...)

	//准备块中的交易
	//	fmt.Println("准备块中的交易")
	coinbase, err := keystore.GetCoinbase()
	if err != nil {
		return
	}

	//开始生成块
	bh := BlockHead{
		Height:            lastBlock.Height + 1,       //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       lastBlock.Group.Height + 1, //矿工组高度
		Previousblockhash: [][]byte{lastBlock.Id},     //上一个区块头hash
		NTx:               uint64(len(txs)),           //交易数量
		Tx:                txids,                      //本区块包含的交易id
		Time:              time.Now().Unix(),          //unix时间戳
		//		BackupMiner:       bmId,                            //备用矿工选举结果hash
		//		DepositId: this.DepositId, //预备矿工组高度
		Witness: *coinbase.Hash, //此块矿工地址
	}
	bh.BuildMerkleRoot()

	if !findNonce(&bh) {
		fmt.Println("因中断而退出")
		return
	}

	bhvo := CreateBlockHeadVO(&bh, txs)

	//广播区块
	MulticastBlock(bhvo)

	fmt.Println("========出块完成======= 高度为", bhvo.BH.Height)
	AddBlockHead(bhvo)
}

/*
	发起投票，广播
*/
func Seekvote() {
	//	log.Println("发起投票")
	//	engine.NLog.Debug(engine.LOG_console, "发起投票")
	if nodeStore.NodeSelf.IsSuper {
		//		engine.NLog.Debug(engine.LOG_console, "是超级节点发起投票")
		log.Println("是超级节点发起投票")

		//		coinbase := "1234567890"

		//		ele := NewElection(coinbase)
		//		content := ele.JSON()
		//		if content == nil {
		//			return
		//		}

		//添加自己为竞选
		//		AddElection(ele)

		ele := NewElection(nodeStore.NodeSelf.IdInfo.Id)

		//		content := []byte(*nodeStore.NodeSelf.IdInfo.Id)
		head := mc.NewMessageHead(nil, nil, false)
		body := mc.NewMessageBody(ele.JSON(), "", nil, 0)
		message := mc.NewMessage(head, body)
		message.BuildHash()

		//广播给其他节点
		//		ids := nodeStore.GetIdsForFar(message.Content)
		for _, one := range nodeStore.GetAllNodes() {
			log.Println("发送给", one.B58String())
			if ss, ok := engine.GetSession(one.B58String()); ok {
				ss.Send(config.MSGID_multicast_vote_recv, head.JSON(), body.JSON(), false)
			} else {
				engine.NLog.Debug(engine.LOG_console, "发送消息失败")
			}
		}
	} else {
		//非超级节点不需要广播
	}
}

/*
	广播挖到的区块
*/
func MulticastBlock(bhVO *BlockHeadVO) {
	if nodeStore.NodeSelf.IsSuper {
		//		engine.NLog.Debug(engine.LOG_console, "是超级节点发起投票")
		//		log.Println("是超级节点发起投票")
		head := mc.NewMessageHead(nil, nil, false)
		//		content := []byte(*nodeStore.NodeSelf.IdInfo.Id)
		bs, err := bhVO.Json()
		if err != nil {
			return
		}
		body := mc.NewMessageBody(bs, "", nil, 0)
		message := mc.NewMessage(head, body)
		message.BuildHash()

		//广播给其他节点
		//		ids := nodeStore.GetIdsForFar(message.Content)
		for _, one := range nodeStore.GetAllNodes() {
			//			log.Println("区块广播发送给", one.B58String())
			if ss, ok := engine.GetSession(one.B58String()); ok {
				ss.Send(config.MSGID_multicast_blockhead, head.JSON(), body.JSON(), false)
			} else {
				engine.NLog.Debug(engine.LOG_console, "发送消息失败")
			}
		}
	} else {
		//非超级节点不需要广播
	}
}
