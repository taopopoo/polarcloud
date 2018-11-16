package mining

import (
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
)

var chain = new(Chain)

func init() {
	chain.witnessChain = new(WitnessChain)
}

/*
	查找一个地址的余额
*/
func FindSurplus(addr utils.Multihash) uint64 {
	return 0
}

/*
	查找最后一组旷工地址
*/
func FindLastGroupMiner() []utils.Multihash {
	return []utils.Multihash{}
}

type Chain struct {
	Init          bool          //是否是创世节点
	group         *Group        //已经产出的区块链，只保留最后10个组，如果保留1年的块，内存占用超200M
	witnessChain  *WitnessChain //见证人组链
	StartingBlock uint64        //区块开始高度
	HighestBlock  uint64        //网络节点的最高高度
	CurrentBlock  uint64        //已经同步到的区块高度
	PulledStates  uint64        //正在同步的区块高度
}

type Group struct {
	PreGroup  *Group   //前置组
	NextGroup *Group   //下一个组
	Height    uint64   //组高度
	Blocks    []*Block //组中的区块
	//	Witness   []*Witness //这个组选出的备用见证人列表排序结果
}

/*
	将这个组中每个块投票结果统计，统计结果加入预备见证人组中
*/
//func (this *Group) BuildWitness() error {
//	//	fmt.Println("++++构建见证人组")

//	//第一个组第一个块要特殊处理
//	if this.Height == 1 {

//		//构建备用见证人
//		bh, err := this.Blocks[0].Load()
//		if err != nil {
//			return err
//		}
//		for _, one := range bh.Tx {
//			class, err := binary.ReadUvarint(bytes.NewBuffer(one[:8]))
//			if err != nil {
//				return err
//			}
//			//			fmt.Println("--交易类型", class)
//			if class != config.Wallet_tx_type_deposit_in {
//				continue
//			}

//			//
//			bs, err := db.Find(one)
//			if err != nil {
//				return err
//			}
//			txitr, err := ParseTxBase(bs)
//			if err != nil {
//				return err
//			}
//			depositIn := txitr.(*Tx_deposit_in)
//			addr, err := keystore.ParseHashByPubkey(depositIn.Vin[0].Puk)
//			if err != nil {
//				return err
//			}
//			witness := Witness{
//				DepositId: one,
//				Addr:      addr,
//			}
//			chain.witnessChain.AddWitness(&witness)
//		}
//		chain.witnessChain.BuildWitnessGroupForNum(1, 2)

//		return nil
//	}

//	//找出这个组出块的见证人
//	witnes := make([]Witness, 0)
//	//找出这个组投票出来的见证人列表
//	miners := make([]*BackupMiners, 0) //当前组投票结果
//	for _, one := range this.Blocks {
//		bm, _, err := one.LoadBackminer()
//		if err != nil {
//			return err
//		}
//		//		fmt.Println("++++====投票结果", len(bm.Miners))
//		miners = append(miners, bm)
//		//		chain.witnessChain.SetWitnessBlock(addr, one)

//		//TODO 只保留已经交了押金的投票，删除未交押金的投票
//		bh, err := one.Load()
//		if err != nil {
//			return err
//		}
//		for _, txid := range bh.Tx {
//			t, err := binary.ReadUvarint(bytes.NewBuffer(txid[:8]))
//			if err != nil {
//				return err
//			}
//			//			fmt.Println("这个交易类型为", t, config.Wallet_tx_type_deposit_in)
//			if t != config.Wallet_tx_type_deposit_in {
//				continue
//			}
//			//找到交押金的交易
//			bs, err := db.Find(txid)
//			if err != nil {
//				return err
//			}
//			txitr, err := ParseTxBase(bs)
//			if err != nil {
//				return err
//			}
//			depositIn := txitr.(*Tx_deposit_in)
//			addr, err := keystore.ParseHashByPubkey(depositIn.Vin[0].Puk)
//			if err != nil {
//				return err
//			}
//			witness := Witness{
//				DepositId: txid,
//				Addr:      addr,
//			}
//			witnes = append(witnes, witness)
//			//			fmt.Println("添加一个见证人", witness.Addr.B58String())
//			chain.witnessChain.AddWitness(&witness)
//		}
//	}
//	//	fmt.Println("查看待统计的见证人", len(miners[0].Miners))
//	//计算备用矿工
//	count := countBackupGroupVote(miners)
//	//TODO 每组见证人投票最多保留3组
//	//TODO 当备用见证人组超过60组，则去掉多余的见证人

//	//addr循环在外面，保证添加进列表的顺序
//	//	n := uint64(0)
//	//	for _, one := range addrs {
//	//		for _, two := range witnes {
//	//			if one.B58String() == two.Addr.B58String() {
//	//				n = n + 1
//	//				chain.witnessChain.AddWitness(&two)
//	//				break
//	//			}
//	//		}
//	//	}
//	//	fmt.Println("++++本次构建见证人组人数", count)
//	//TODO 有多组矿工，可以同时构建多组

//	chain.witnessChain.BuildWitnessGroupForNum(count, 0)
//	//	chain.witnessChain.BuildWitnessGroup() //确定备用见证人
//	return nil
//}

//func (this *Group)

type Block struct {
	PreBlock  *Block  //前置区块高度
	NextBlock *Block  //下一个区块高度
	Group     *Group  //所属组
	Height    uint64  //区块高度
	Id        []byte  //区块id
	DepositTx []TxItr //押金交易，每个组里的押金交易不能重复

}

func (this *Block) Load() (*BlockHead, error) {
	bh, err := db.Find(this.Id)
	if err != nil {
		//		if err == leveldb.ErrNotFound {
		//			return
		//		} else {
		//		}
		return nil, err
	}

	//		bs := (bh).(*[]byte)
	blockHead, err := ParseBlockHead(bh)
	if err != nil {
		return nil, err
	}
	return blockHead, nil
}

//func (this *Block) LoadTx() (*BlockHead, error) {
//	bh, err := db.Find(this.Id)
//	if err != nil {
//		//		if err == leveldb.ErrNotFound {
//		//			return
//		//		} else {
//		//		}
//		return nil, err
//	}

//	//		bs := (bh).(*[]byte)
//	blockHead, err := ParseBlockHead(bh)
//	if err != nil {
//		return nil, err
//	}
//	return blockHead, nil
//}

/*
	加载这个块上的见证人投票结果
*/
//func (this *Block) LoadBackminer() (*BackupMiners, *utils.Multihash, error) {
//	blockHead, err := this.Load()
//	if err != nil {
//		return nil, nil, err
//	}

//	minerBS, err := db.Find(blockHead.BackupMiner)
//	if err != nil {
//		return nil, nil, err
//	}
//	//	fmt.Println("==========\n", string(*minerBS))
//	bm, err := ParseBackupMiners(minerBS)
//	if err != nil {
//		return nil, nil, err
//	}
//	return bm, &blockHead.Witness, nil
//}

/*
	获得链上最后一个区块
*/
func (this *Block) GetLastBlock() *Block {
	block := this
	for {
		if block.NextBlock == nil {
			return block
		}
		block = block.NextBlock
	}
}

/*
	同步下一个区块
*/
//func (this *Block) SyncNextBlock() *Block {
//}

/*
	设置见证人
*/
//func (this *Group) SetWitness(addrs []*utils.Multihash) {

//	this.Miners = addrs
//}

/*
	获得这个组中的最后一个区块
*/
//func (this *Group) GetLastBlock() *Block {

//}

/*
	添加一个区块
	只能连续添加区块高度更高的区块
*/
func (this *Chain) AddBlock(bh *BlockHead, txs *[]TxItr) bool {
	fmt.Println("添加一个区块", bh.Height, hex.EncodeToString(bh.Hash))

	CountBalanceForBlock(&BlockHeadVO{BH: bh, Txs: *txs})

	newBlock := new(Block)
	newBlock.Id = bh.Hash
	newBlock.Height = bh.Height

	depositTxs := make([]TxItr, 0)
	for _, one := range *txs {
		switch one.Class() {
		//过滤见证人押金交易
		case config.Wallet_tx_type_deposit_in:
			depositTxs = append(depositTxs, one)
			addr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
			if err != nil {
				continue
			}
			score := (*one.GetVout())[0].Value
			addWitness(addr, score)
		}
	}
	newBlock.DepositTx = depositTxs

	//添加首个区块
	if this.group == nil {
		newGroup := new(Group)
		newGroup.Blocks = make([]*Block, 0)
		newBlock.Group = newGroup
		newGroup.Blocks = append(newGroup.Blocks, newBlock)
		newGroup.Height = bh.GroupHeight
		this.group = newGroup
		//		this.group.BuildWitness()
		//		atomic.StoreUint64(&this.CurrentBlock, newBlock.Height)
		fmt.Println("添加首个区块")

	} else {
		lastBlock := this.group.Blocks[0].GetLastBlock()
		//判断区块高度是否连续
		if (lastBlock.Height + 1) != bh.Height {
			fmt.Println("添加的区块高度不连续")
			return false
		}

		//TODO 赋值考虑异步安全 atomic
		lastBlock.NextBlock = newBlock
		newBlock.PreBlock = lastBlock

		if bh.GroupHeight == lastBlock.Group.Height {
			lastBlock.Group.Blocks = append(lastBlock.Group.Blocks, newBlock)
			newBlock.Group = lastBlock.Group
		} else {
			//新的组
			newGroup := new(Group)
			newGroup.Blocks = make([]*Block, 0)
			newBlock.Group = newGroup
			newGroup.Blocks = append(newGroup.Blocks, newBlock)
			newGroup.Height = bh.GroupHeight
			newGroup.PreGroup = lastBlock.Group
			lastBlock.Group.NextGroup = newGroup
			this.group = newGroup
		}
	}

	//	ok, last := this.witnessChain.SetWitnessBlock(newBlock)
	//	fmt.Println("是否设置成功", ok, "是不是这个组的最后一个块", last)
	//	if last {
	//		//已经是这个组的最后一个块，可以统计投票结果
	//		fmt.Println("已经是这个组的最后一个块，可以统计投票结果")
	//		this.group.BuildWitness()
	//	}
	Mining()
	return true
}

/*
	获得首个区块
*/
func (this *Chain) GetFirstBlock() *Block {
	if this.group == nil {
		return nil
	}
	group := this.group
	for group.PreGroup != nil {
		group = group.PreGroup
	}
	return group.Blocks[0]
}

/*
	获得最后一个区块
*/
func (this *Chain) GetLastBlock() *Block {
	if this.group == nil {
		return nil
	}
	return this.group.Blocks[len(this.group.Blocks)-1]
}

/*
	检查投票
*/
func (this *Chain) CheckVote(key *keystore.Address) {
	//	this.witnessChain.PrintWitnessList()
	next := this.witnessChain.GetBackupWitness()
	for {
		if next == nil {
			break
		}
		//		fmt.Println("对比备用见证人投票地址", hex.EncodeToString(next.DepositId), next.Addr.B58String())
		if next.Addr.B58String() == key.Hash.B58String() {
			MulticastBallotTicket(&next.DepositId, next.Addr)
		}
		if next.NextWitness == nil {
			break
		}
		next = next.NextWitness
	}
	//	this.witnessChain.PrintWitnessList()
}

/*
	打印块列表
*/
func (this *Chain) PrintBlockList() {

	start := this.GetLastBlock()
	for {
		if start.PreBlock == nil {
			break
		}
		start = start.PreBlock
	}
	for {
		fmt.Println("打印块列表", start.Height)
		if start.NextBlock == nil {
			break
		}
		start = start.NextBlock
	}
}

/*
	获取区块开始高度
*/
func SetStartingBlock(n uint64) {
	atomic.StoreUint64(&chain.StartingBlock, n)
}

/*
	获取所链接的节点的最高高度
*/
func SetHighestBlock(n uint64) {
	atomic.StoreUint64(&chain.HighestBlock, n)
}
