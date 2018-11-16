package mining

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	//	"math/big"
	"strconv"
	"time"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
)

/*
	开始挖矿
	当每个组见证人选出来之后，启动挖矿程序，按顺序定时出块
*/
func Mining() {
	fmt.Println("Mining()")
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
	Deposit()

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
		//判断自己出块顺序的时间
		for i, one := range chain.witnessChain.group.Witness {
			//		if one.Addr.B58String() ==
			//自己是见证人才能出块，否则自己出块了，其他节点也不会承认
			if one.Addr.B58String() == addr.Hash.B58String() {
				fmt.Println("多少秒钟后出块", config.Mining_block_time*(i+1))
				utils.AddTimetask(time.Now().Unix()+int64(config.Mining_block_time*(i+1)),
					TaskBuildBlock, Task_class_buildBlock, "")
			}
		}
		fmt.Println("本节点没有在备用见证人中")
	}

}

/*
	启动挖矿程序
	在每一组出块完成时候开始挖矿
*/
func StartMining() error {
	if !config.Miner {
		return nil
	}

	fmt.Println("启动挖矿程序")
	//只有超级节点才能成为旷工，普通节点发起投票时，其他节点联系不上，是不会给他投票的。
	//	if !nodeStore.NodeSelf.IsSuper {
	//		fmt.Println("本节点不是超级节点，不能挖矿")
	//		return nil
	//	}

	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return err
	}

	//缴纳备用见证人押金交易
	err = chain.witnessChain.PayDeposit()
	if err != nil {
		fmt.Println("缴纳押金失败", err)
	}
	fmt.Println("缴纳押金完成")

	//判断自己的押金交易是否被打包到块中，已打包则给自己投票
	chain.CheckVote(addr)
	//	fmt.Println("投票完成")

	//让下一个见证人出块
	witness := chain.witnessChain.GetBackupWitness()
	if witness == nil {
		fmt.Println("没有备用见证人")
		return errors.New("没有备用见证人")
	}
	go witness.BuildBlock()

	//保证最少个矿工参加竟票才能出块
	fmt.Println("挖矿执行完成")
	return nil

}

/*
	出块并广播
	@gh    uint64    出块的组高度
	@id    []byte    押金id
*/
func (this *Witness) BuildBlock() {
	addr, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}
	//自己是见证人才能出块，否则自己出块了，其他节点也不会承认
	if this.Addr.B58String() != addr.Hash.B58String() {
		return
	}
	if this.PreWitness == nil {
		fmt.Println("===准备出块===")
	} else {
		fmt.Println("===准备出块" + strconv.Itoa(int(this.PreWitness.Block.Height+1)) + "===")
	}

	<-time.NewTimer(time.Second * 10).C

	//统计投票数量
	miners := make([]BackupMiner, 0)
	groupM, restM := chain.witnessChain.CountWitness()
	for k, v := range groupM {
		//				fmt.Println("轮询", k, v)
		addr, err := utils.FromB58String(k)
		if err != nil {
			fmt.Println("解析地址错误", err)
			continue
		}
		bm := BackupMiner{
			Miner: addr,
			Count: v, //票数
		}
		miners = append(miners, bm)
		//				fmt.Println("添加一个投票", bm)
	}

	for k, v := range restM {
		addr, err := utils.FromB58String(k)
		if err != nil {
			continue
		}
		bm := BackupMiner{
			Miner: addr,
			Count: v, //票数
		}
		miners = append(miners, bm)
	}

	//统计本组的所有押金，不能有重复
	existDeposit := make([]TxItr, 0)
	if this.Group.Witness != nil && len(this.Group.Witness) >= 1 &&
		this.Group.Witness[0].Addr.B58String() != this.Addr.B58String() { //不是第一个块
		for _, one := range this.Group.Witness {
			if one.Block == nil { //这里出块可以不用连续
				continue
			}
			bh, err := one.Block.Load()
			if err != nil {
				fmt.Println("本地数据库损坏了", err)
				return
			}
			for _, two := range bh.Tx {

				class, err := binary.ReadUvarint(bytes.NewBuffer(two[:8]))
				if class != config.Wallet_tx_type_deposit_in {
					continue
				}

				txbs, err := db.Find(two)
				if err != nil {
					return
				}
				txBase, err := ParseTxBase(txbs)
				if err != nil {
					return
				}
				existDeposit = append(existDeposit, txBase)
			}
		}
	}

	//打包交易
	tx := make([]TxItr, 0)
	txids := make([][]byte, 0)
	//判断是否是该组第一个块
	if this.Group.Witness != nil && len(this.Group.Witness) >= 1 &&
		this.Group.Witness[0].Addr.B58String() == this.Addr.B58String() &&
		this.Group.PreGroup != nil {

		fmt.Println("====+++++是该组第一个块")

		//该组第一个块，统计上一组的出块奖励
		vouts := make([]Vout, 0)
		group := this.Group.PreGroup
		for _, one := range group.Witness {
			if one.Block == nil {
				continue
			}
			vout := Vout{
				Value:   config.Mining_reward,
				Address: *one.Addr,
			}
			vouts = append(vouts, vout)
		}
		base := TxBase{
			Type: config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
			//	Vin_total  uint64 `json:"vin_total"`  //输入交易数量
			//	Vin        []Vin  `json:"vin"`        //交易输入
			Vout_total: uint64(len(vouts)), //输出交易数量
			Vout:       vouts,              //交易输出
			//	BlockHash  []byte `json:"blockhash"`  //自己被打包到的块hash
		}

		txReward := Tx_reward{
			TxBase:     base,
			CreateTime: time.Now().Unix(), //创建时间
		}
		txReward.BuildHash()
		txids = append(txids, *txReward.GetHash())
		tx = append(tx, &txReward)

		//该组第一个块，退还上一组见证人押金
		vinsReward := make([]Vin, 0)
		voutsReward := make([]Vout, 0)
		group = this.Group.PreGroup
		for _, one := range group.Witness {
			if one.Block == nil {
				continue
			}

			bs, err := db.Find(one.DepositId)
			if err != nil {
				return
			}
			txItr, err := ParseTxBase(bs)
			if err != nil {
				return
			}
			vin := Vin{
				Txid: one.DepositId, //UTXO 前一个交易的id
				Vout: 0,             //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（从零开始）
				//	Puk      []byte `json:"puk"`      //公钥
				//	Sign     []byte `json:"sign"`     //对上一个交易的输出签名
				//	VoutSign []byte `json:"voutsign"` //对本交易的输出签名
			}
			vinsReward = append(vinsReward, vin)

			//			fmt.Println("GetVout", txItr.Class(), len(*txItr.GetVin()), len(*txItr.GetVout()))

			vout := Vout{
				Value:   (*txItr.GetVout())[0].Value,
				Address: (*txItr.GetVout())[0].Address,
			}
			voutsReward = append(voutsReward, vout)
		}
		base = TxBase{
			Type:       config.Wallet_tx_type_deposit_out, //交易类型
			Vin_total:  uint64(len(vinsReward)),           //输入交易数量
			Vin:        vinsReward,                        //交易输入
			Vout_total: uint64(len(voutsReward)),          //输出交易数量
			Vout:       voutsReward,                       //交易输出
			//	BlockHash  []byte `json:"blockhash"`  //自己被打包到的块hash
		}

		txOut := Tx_deposit_out{
			TxBase:     base,
			CreateTime: time.Now().Unix(), //创建时间
		}
		txOut.BuildHash()
		txids = append(txids, *txOut.GetHash())
		tx = append(tx, &txOut)
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

	//	bm := BackupMiners{
	//		Time:   time.Now().Unix(), //统计时间
	//		Miners: miners,            //预备矿工最多保存两组矿工最大数量(14个)
	//	}
	//	bmId := utils.Hash_SHA3_256(*bm.JSON())
	//	fmt.Println("=====", bm.Miners)
	//开始生成块
	bh := BlockHead{
		Height:            chain.GetLastBlock().Height + 1, //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       this.Group.Height,               //矿工组高度
		Previousblockhash: chain.GetLastBlock().Id,         //上一个区块头hash
		NTx:               uint64(len(tx)),                 //交易数量
		Tx:                txids,                           //本区块包含的交易id
		Time:              time.Now().Unix(),               //unix时间戳
		//		BackupMiner:       bmId,                            //备用矿工选举结果hash
		//		DepositId: this.DepositId, //预备矿工组高度
		Witness: *coinbase.Hash, //此块矿工地址
	}
	bh.BuildMerkleRoot()
	bh.BuildHash()

	bhvo := CreateBlockHeadVO(&bh, tx)

	//	for _, one := range bhvo.Txs {
	//		if one.Class() == 0 {
	//			fmt.Println("生成区块中的交易", one)
	//		}
	//	}

	//广播区块
	MulticastBlock(bhvo)

	fmt.Println("========出块完成", bhvo.BH.Height, "=======")
	AddBlockHead(bhvo)

	//	err = ImportBlock(bhvo)
	//	if err != nil {
	//		fmt.Println("导入块错误", err)
	//	}

}

/*
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

	if this.PreWitness != nil && this.PreWitness.Block != nil {
		fmt.Println("===准备出块" + strconv.Itoa(int(this.PreWitness.Block.Height+1)) + "===")
	}

	//统计投票数量
	miners := make([]BackupMiner, 0)
	groupM, restM := chain.witnessChain.CountWitness()
	for k, v := range groupM {
		//				fmt.Println("轮询", k, v)
		addr, err := utils.FromB58String(k)
		if err != nil {
			fmt.Println("解析地址错误", err)
			continue
		}
		bm := BackupMiner{
			Miner: addr,
			Count: v, //票数
		}
		miners = append(miners, bm)
		//				fmt.Println("添加一个投票", bm)
	}

	for k, v := range restM {
		addr, err := utils.FromB58String(k)
		if err != nil {
			continue
		}
		bm := BackupMiner{
			Miner: addr,
			Count: v, //票数
		}
		miners = append(miners, bm)
	}

	//统计本组的所有押金，不能有重复
	existDeposit := make([]TxItr, 0)
	if this.Group.Witness != nil && len(this.Group.Witness) >= 1 &&
		this.Group.Witness[0].Addr.B58String() != this.Addr.B58String() { //不是第一个块
		for _, one := range this.Group.Witness {
			if one.Block == nil { //这里出块可以不用连续
				continue
			}
			bh, err := one.Block.Load()
			if err != nil {
				fmt.Println("本地数据库损坏了", err)
				return
			}
			for _, two := range bh.Tx {

				class, err := binary.ReadUvarint(bytes.NewBuffer(two[:8]))
				if class != config.Wallet_tx_type_deposit_in {
					continue
				}

				txbs, err := db.Find(two)
				if err != nil {
					return
				}
				txBase, err := ParseTxBase(txbs)
				if err != nil {
					return
				}
				existDeposit = append(existDeposit, txBase)
			}
		}
	}

	//打包交易
	tx := make([]TxItr, 0)
	txids := make([][]byte, 0)
	//判断是否是该组第一个块
	if this.Group.Witness != nil && len(this.Group.Witness) >= 1 &&
		this.Group.Witness[0].Addr.B58String() == this.Addr.B58String() &&
		this.Group.PreGroup != nil {

		fmt.Println("====+++++是该组第一个块")

		//该组第一个块，统计上一组的出块奖励
		vouts := make([]Vout, 0)
		group := this.Group.PreGroup
		for _, one := range group.Witness {
			if one.Block == nil {
				continue
			}
			vout := Vout{
				Value:   config.Mining_reward,
				Address: *one.Addr,
			}
			vouts = append(vouts, vout)
		}
		base := TxBase{
			Type: config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
			//	Vin_total  uint64 `json:"vin_total"`  //输入交易数量
			//	Vin        []Vin  `json:"vin"`        //交易输入
			Vout_total: uint64(len(vouts)), //输出交易数量
			Vout:       vouts,              //交易输出
			//	BlockHash  []byte `json:"blockhash"`  //自己被打包到的块hash
		}

		txReward := Tx_reward{
			TxBase:     base,
			CreateTime: time.Now().Unix(), //创建时间
		}
		txReward.BuildHash()
		txids = append(txids, *txReward.GetHash())
		tx = append(tx, &txReward)

		//该组第一个块，退还上一组见证人押金
		vinsReward := make([]Vin, 0)
		voutsReward := make([]Vout, 0)
		group = this.Group.PreGroup
		for _, one := range group.Witness {
			if one.Block == nil {
				continue
			}

			bs, err := db.Find(one.DepositId)
			if err != nil {
				return
			}
			txItr, err := ParseTxBase(bs)
			if err != nil {
				return
			}
			vin := Vin{
				Txid: one.DepositId, //UTXO 前一个交易的id
				Vout: 0,             //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（从零开始）
				//	Puk      []byte `json:"puk"`      //公钥
				//	Sign     []byte `json:"sign"`     //对上一个交易的输出签名
				//	VoutSign []byte `json:"voutsign"` //对本交易的输出签名
			}
			vinsReward = append(vinsReward, vin)

			//			fmt.Println("GetVout", txItr.Class(), len(*txItr.GetVin()), len(*txItr.GetVout()))

			vout := Vout{
				Value:   (*txItr.GetVout())[0].Value,
				Address: (*txItr.GetVout())[0].Address,
			}
			voutsReward = append(voutsReward, vout)
		}
		base = TxBase{
			Type:       config.Wallet_tx_type_deposit_out, //交易类型
			Vin_total:  uint64(len(vinsReward)),           //输入交易数量
			Vin:        vinsReward,                        //交易输入
			Vout_total: uint64(len(voutsReward)),          //输出交易数量
			Vout:       voutsReward,                       //交易输出
			//	BlockHash  []byte `json:"blockhash"`  //自己被打包到的块hash
		}

		txOut := Tx_deposit_out{
			TxBase:     base,
			CreateTime: time.Now().Unix(), //创建时间
		}
		txOut.BuildHash()
		txids = append(txids, *txOut.GetHash())
		tx = append(tx, &txOut)
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

	//	bm := BackupMiners{
	//		Time:   time.Now().Unix(), //统计时间
	//		Miners: miners,            //预备矿工最多保存两组矿工最大数量(14个)
	//	}
	//	bmId := utils.Hash_SHA3_256(*bm.JSON())
	//	fmt.Println("=====", bm.Miners)
	//开始生成块
	bh := BlockHead{
		Height:            chain.GetLastBlock().Height + 1, //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       this.Group.Height,               //矿工组高度
		Previousblockhash: chain.GetLastBlock().Id,         //上一个区块头hash
		NTx:               uint64(len(tx)),                 //交易数量
		Tx:                txids,                           //本区块包含的交易id
		Time:              time.Now().Unix(),               //unix时间戳
		//		BackupMiner:       bmId,                            //备用矿工选举结果hash
		//		DepositId: this.DepositId, //预备矿工组高度
		Witness: *coinbase.Hash, //此块矿工地址
	}
	bh.BuildMerkleRoot()
	bh.BuildHash()

	bhvo := CreateBlockHeadVO(&bh, tx)

	//	for _, one := range bhvo.Txs {
	//		if one.Class() == 0 {
	//			fmt.Println("生成区块中的交易", one)
	//		}
	//	}

	//广播区块
	MulticastBlock(bhvo)

	fmt.Println("========出块完成=======")
	AddBlockHead(bhvo)

	//	err = ImportBlock(bhvo)
	//	if err != nil {
	//		fmt.Println("导入块错误", err)
	//	}

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

	fmt.Println("===准备出块===")

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

	//	txs := make([]*Tx_reward, 0)
	//	vouts := make([]Vout, 0)
	//	vouts = append(vouts, Vout{
	//		Value:   config.Mining_reward, //输出金额 = 实际金额 * 100000000
	//		Address: *addr.Hash,           //钱包地址
	//	})
	//	base := TxBase{
	//		Type:       config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
	//		Vout_total: 1,                            //输出交易数量
	//		Vout:       vouts,                        //交易输出
	//	}
	//	reward := Tx_reward{
	//		TxBase:     base,
	//		CreateTime: time.Now().Unix(), //创建时间
	//	}
	//	txs = append(txs, &reward)
	//	reward.BuildHash()
	//	txHashes = append(txHashes, reward.Hash)
	//	//区块头
	//	blockHead1 := BlockHead{
	//		Height:      1, //区块高度(每秒产生一个块高度，也足够使用上千亿年)
	//		GroupHeight: 1, //
	//		//	MerkleRoot        string   //交易默克尔树根hash
	//		//	Previousblockhash string   //上一个区块头hash
	//		//	Nextblockhash     string   //下一个区块头hash
	//		NTx:     uint64(len(txHashes)), //交易数量
	//		Tx:      txHashes,              //本区块包含的交易id
	//		Time:    time.Now().Unix(),     //unix时间戳
	//		Witness: *witness[0].GetAddr()[0].Hash,
	//	}
	//	blockHead1.BuildMerkleRoot()
	//	blockHead1.FindNonce(20)
	//	//	db.Save(blockHead1.BackupMiner, backupMiner1.JSON())
	//	bhbs, _ := blockHead1.Json()

	//开始生成块
	bh := BlockHead{
		Height:            chain.GetLastBlock().Height + 1,       //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
		GroupHeight:       chain.GetLastBlock().Group.Height + 1, //矿工组高度
		Previousblockhash: chain.GetLastBlock().Id,               //上一个区块头hash
		NTx:               uint64(len(txs)),                      //交易数量
		Tx:                txids,                                 //本区块包含的交易id
		Time:              time.Now().Unix(),                     //unix时间戳
		//		BackupMiner:       bmId,                            //备用矿工选举结果hash
		//		DepositId: this.DepositId, //预备矿工组高度
		Witness: *coinbase.Hash, //此块矿工地址
	}
	bh.BuildMerkleRoot()
	bh.FindNonce(20)

	bhvo := CreateBlockHeadVO(&bh, txs)

	//广播区块
	MulticastBlock(bhvo)

	fmt.Println("========出块完成=======")
	AddBlockHead(bhvo)

	//	err = ImportBlock(bhvo)
	//	if err != nil {
	//		fmt.Println("导入块错误", err)
	//	}

}

/*
	缴纳押金
*/
func (this *WitnessChain) PayDeposit() error {
	key, err := keystore.GetCoinbase()
	if err != nil {
		return err
	}
	//	fmt.Println("111111111111111")
	//	group := this.group
	//	for {
	//		if group.NextGroup == nil {
	//			break
	//		}
	//		group = group.NextGroup
	//	}
	//	fmt.Println("22222222222222222")
	//	fmt.Println("缴纳押金方法", len(group.Witness), group.Witness, group.Height)
	//	last := group.Witness[len(group.Witness)-1]
	//	//查询最后一组备用见证人是否包含自己，不包含则可以缴押金
	//	find := false
	//PayDeposit_out_for:
	//	for {
	//		find = false
	//		for i := 0; i < config.Mining_group_max; i++ {
	//			if last.NextWitness == nil {
	//				break PayDeposit_out_for
	//			}
	//			last = last.NextWitness
	//			//			fmt.Println(last.Addr.B58String())
	//			if last.Addr.B58String() == key.Hash.B58String() {
	//				find = true
	//			}
	//		}
	//	}
	//	fmt.Println("33333333333333")
	//	if find {
	//		fmt.Println("33333333333333 11111")
	//		return nil
	//	}

	deposiIn := CreateTxDepositIn(key)
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
