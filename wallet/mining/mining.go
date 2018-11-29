package mining

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
	"strconv"
	"time"
)

/*
	开始挖矿
	当每个组见证人选出来之后，启动挖矿程序，按顺序定时出块
*/
func Mining() {
	if !config.Miner {
		return
	}
	//判断是否同步完成
	if GetHighestBlock() <= 0 {
		fmt.Println("开始挖矿，但是区块未同步完成", GetHighestBlock())
		return
	}

	fmt.Println("启动挖矿程序")

	//交押金
	//	Deposit()

	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}
	//判断用什么方式出块
	if chain.witnessChain.group == nil {
		//用工作量证明方式出块
		fmt.Println("用工作量证明方式出块")
		BuildBlockForPOW()
	} else {
		//用见证人方式出块
		fmt.Println("用见证人方式出块")
		//判断是否已经安排了任务
		if chain.witnessChain.group.Task {
			return
		}
		chain.witnessChain.group.Task = true

		//判断自己出块顺序的时间
		for i, one := range chain.witnessChain.group.Witness {
			//自己是见证人才能出块，否则自己出块了，其他节点也不会承认
			if bytes.Equal(*one.Addr, *addr.Hash) {
				fmt.Println("多少秒钟后出块", config.Mining_block_time*(i+1))
				utils.AddTimetask(time.Now().Unix()+int64(config.Mining_block_time*(i+1)),
					TaskBuildBlock, Task_class_buildBlock, "")
			}
		}
		fmt.Println("本节点没有在备用见证人中")
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
	for _, one := range chain.witnessChain.group.Witness {
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

	lastBlock := chain.GetLastBlock()

	if this.PreWitness != nil && this.PreWitness.Block != nil {
		fmt.Println("===准备出块" + strconv.Itoa(int(lastBlock.Height+1)) + "===")
	}

	//打包交易
	tx := make([]TxItr, 0)
	txids := make([][]byte, 0)
	//判断是否是该组第一个块
	//判断上一个组是否是见证人方式出块，是见证人方式出块，计算上一组出块奖励。
	if chain.witnessChain.beforeGroup != nil {
		reward := chain.witnessChain.beforeGroup.CountReward()
		tx = append(tx, reward)
		txids = append(txids, reward.Hash)
	}

	//打包10秒内的所有交易
	unpackedTransactions.Range(func(k, v interface{}) bool {
		txItr := v.(TxItr)
		tx = append(tx, txItr)
		txids = append(txids, *txItr.GetHash())
		return true
	})

	//准备块中的交易
	//	fmt.Println("准备块中的交易")
	coinbase, err := keystore.GetCoinbase()
	if err != nil {
		return
	}

	//开始生成块
	bh := BlockHead{
		Height:            lastBlock.Height + 1,    //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       this.Group.Height,       //矿工组高度
		Previousblockhash: chain.GetLastBlock().Id, //上一个区块头hash
		NTx:               uint64(len(tx)),         //交易数量
		Tx:                txids,                   //本区块包含的交易id
		Time:              time.Now().Unix(),       //unix时间戳
		//		BackupMiner:       bmId,                            //备用矿工选举结果hash
		//		DepositId: this.DepositId, //预备矿工组高度
		Witness: *coinbase.Hash, //此块矿工地址
	}
	bh.BuildMerkleRoot()
	bh.BuildHash()

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

	//打包交易
	txs := make([]TxItr, 0)
	txids := make([][]byte, 0)

	//构造出块奖励
	vouts := make([]Vout, 0)
	vouts = append(vouts, Vout{
		Value:   config.Mining_reward, //输出金额 = 实际金额 * 100000000
		Address: *addr.Hash,           //钱包地址
	})
	base := TxBase{
		Type:       config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vout_total: 1,                            //输出交易数量
		Vout:       vouts,                        //交易输出
	}
	reward := Tx_reward{
		TxBase:     base,
		CreateTime: time.Now().Unix(), //创建时间
	}
	txs = append(txs, &reward)
	reward.BuildHash()
	txids = append(txids, reward.Hash)

	//打包10秒内的所有交易
	unpackedTransactions.Range(func(k, v interface{}) bool {
		txItr := v.(TxItr)
		txs = append(txs, txItr)
		txids = append(txids, *txItr.GetHash())
		return true
	})

	//准备块中的交易
	//	fmt.Println("准备块中的交易")
	coinbase, err := keystore.GetCoinbase()
	if err != nil {
		return
	}
	lastBlock := chain.GetLastBlock()

	//开始生成块
	bh := BlockHead{
		Height:            lastBlock.Height + 1,       //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       lastBlock.Group.Height + 1, //矿工组高度
		Previousblockhash: lastBlock.Id,               //上一个区块头hash
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
	缴纳押金，并广播
*/
func (this *WitnessChain) PayDeposit(amount uint64) error {
	key, err := keystore.GetCoinbase()
	if err != nil {
		return err
	}

	deposiIn := CreateTxDepositIn(key, amount)
	if deposiIn == nil {
		//		fmt.Println("33333333333333 22222")
		return errors.New("交押金失败")
	}
	deposiIn.BuildHash()
	bs, err := deposiIn.Json()
	if err != nil {
		//		fmt.Println("33333333333333 33333")
		return err
	}
	//	fmt.Println("4444444444444444")
	MulticastTx(bs)
	//	fmt.Println("5555555555555555")
	txbase, err := ParseTxBase(bs)
	if err != nil {
		return err
	}
	txbase.BuildHash()
	//	fmt.Println("66666666666666")
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return errors.New("交易不合法，则不发送出去")
	}
	unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
	//	fmt.Println("7777777777777777")
	return nil
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

///*
//	开始挖矿
//*/
//func mining() {
//	var countPow uint64 = 0

//	//计算上一组旷工中股权分配
//	vouts := make([]Vout, 0)
//	miners := FindLastGroupMiner()
//	for _, one := range miners {
//		surplus := FindSurplus(one)
//		vout := Vout{
//			Value:   surplus,
//			Address: one,
//		}
//		vouts = append(vouts, vout)
//		countPow = countPow + surplus
//	}
//	//本次块旷工收益
//	countBlanas := 25
//	for i, one := range vouts {
//		rat, _ := new(big.Rat).Mul(big.NewRat(int64(countBlanas), 1), big.NewRat(int64(one.Value), int64(countPow))).Float64()
//		vouts[i].Value = uint64(rat * Unit)
//	}

//	//挖矿成功，广播区块头
//	//	MulticastBlock()

//}

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
