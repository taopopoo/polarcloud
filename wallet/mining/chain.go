package mining

import (
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"sync/atomic"
	//	"polarcloud/wallet/keystore"
)

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
	witnessBackup      *WitnessBackup      //备用见证人
	witnessChain       *WitnessChain       //见证人组链
	lastBlock          *Block              //最新块
	balance            *BalanceManager     //
	transactionManager *TransactionManager //交易管理器
}

func NewChain(block *Block) *Chain {
	chain := &Chain{}
	wb := NewWitnessBackup(chain)
	wc := NewWitnessChain(wb)
	tm := NewTransactionManager(wb)
	b := NewBalanceManager(wb, tm)

	chain.lastBlock = block
	chain.witnessBackup = wb
	chain.witnessChain = wc
	chain.balance = b
	chain.transactionManager = tm
	return chain
}

/*
	克隆一个链
*/
//func (this *Chain) Clone() *Chain {
//Chain{
//	witnessBackup : this.witnessBackup ,     *WitnessBackup      //备用见证人
//	witnessChain       *WitnessChain       //见证人组链
//	lastBlock          *Block              //最新块
//	balance            *BalanceManager     //
//	transactionManager *TransactionManager //交易管理器
//}
//}

type Group struct {
	PreGroup  *Group   //前置组
	NextGroup []*Group //下一个组，有分叉，下标为0的是最长链
	Height    uint64   //组高度
	Blocks    []*Block //组中的区块
}

type Block struct {
	PreBlock  []*Block //前置区块高度
	NextBlock []*Block //下一个区块高度
	Group     *Group   //所属组
	Height    uint64   //区块高度
	Id        []byte   //区块id

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

	blockHead, err := ParseBlockHead(bh)
	if err != nil {
		return nil, err
	}
	return blockHead, nil
}

/*
	加载本区块的所有交易
*/
func (this *Block) LoadTxs() (*BlockHead, *[]TxItr, error) {
	bh, err := this.Load()
	if err != nil {
		return nil, nil, err
	}
	txs := make([]TxItr, 0)
	for _, one := range bh.Tx {
		bs, err := db.Find(one)
		if err != nil {
			return nil, nil, err
		}
		txItr, err := ParseTxBase(bs)
		if err != nil {
			return nil, nil, err
		}
		txs = append(txs, txItr)
	}
	return bh, &txs, nil
}

/*
	修改本区块的下一个区块中最长区块下标为0
*/
func (this *Block) UpdateNextIndex(bhash []byte) error {

	return nil
}

/*
	添加一个区块
	只能连续添加区块高度更高的区块
*/
func (this *Chain) CountBlock(bh *BlockHead, txs *[]TxItr) bool {

	//计算余额
	bhvo := &BlockHeadVO{BH: bh, Txs: *txs}
	this.balance.CountBalanceForBlock(bhvo)

	//统计交易中的备用见证人以及见证人投票
	this.witnessBackup.CountWitness(txs)

	//把见证人设置为已出块
	this.witnessChain.SetWitnessBlock(this.GetLastBlock())

	//删除已经打包了的交易
	this.transactionManager.DelTx(bhvo.Txs)

	//跟新同步高度
	//	if GetCurrentBlock()+1 == bhvo.BH.Height {
	//		atomic.StoreUint64(&forks.CurrentBlock, bhvo.BH.Height)
	//	}
	if bhvo.BH.Height > GetCurrentBlock() {
		atomic.StoreUint64(&forks.CurrentBlock, bhvo.BH.Height)
	}

	go Mining()
	return true
}

/*
	统计分叉链上的区块
	@bh    *BlockHead    分叉点区块
	@hs    [][]byte      分叉链hash路径
*/
func (this *Chain) CountForkBlock(block *Block, hs [][]byte) bool {
	block.Load()
}

/*
	回滚一个区块
	@height    uint64    要回滚的区块高度
*/
func (this *Chain) RollbackBlock(height uint64) {
	block := this.GetLastBlock()
	for height < block.Height {
		block = block.PreBlock[0]
	}
	bh, txs, err := block.LoadTxs()
	if err != nil {
		return
	}

	bhvo := &BlockHeadVO{BH: bh, Txs: *txs}
	//回滚余额
	this.balance.RollbackBalance(bhvo)

	//统计交易中的备用见证人以及见证人投票
	this.witnessBackup.RollbackCountWitness(txs)

	//把见证人设置为已出块
	//	this.witnessChain.SetWitnessBlock(this.GetLastBlock())

	//回滚已经打包了的交易
	this.transactionManager.AddTxs(bhvo.Txs)

}

/*
	获取本链最高的区块
*/
func (this *Chain) GetLastBlock() *Block {
	return this.lastBlock
}

/*
	打印块列表
*/
func (this *Chain) PrintBlockList() {

	//	start := this.GetLastBlock()
	//	for {
	//		if start.PreBlock == nil {
	//			break
	//		}
	//		start = start.PreBlock
	//	}
	//	for {
	//		fmt.Println("打印块列表", start.Height)
	//		if start.NextBlock == nil {
	//			break
	//		}
	//		start = start.NextBlock
	//	}
}

/*
	依次获取前n个区块的hash，连接起来做一次hash
*/
func (this *Chain) HashRandom() *[]byte {
	bs := make([]byte, 0)
	lastBlock := this.lastBlock
	for i := 0; i < config.Mining_block_hash_count; i++ {
		if lastBlock == nil {
			break
		}
		bs = append(bs, lastBlock.Id...)
		if lastBlock.PreBlock == nil || len(lastBlock.PreBlock) <= 0 {
			break
		}
		lastBlock = lastBlock.PreBlock[0]
	}
	bs = utils.Hash_SHA3_256(bs)
	return &bs
}

/*
	获取区块开始高度
*/
func SetStartingBlock(n uint64) {
	atomic.StoreUint64(&forks.StartingBlock, n)
}

/*
	获取所链接的节点的最高高度
*/
func SetHighestBlock(n uint64) {
	atomic.StoreUint64(&forks.HighestBlock, n)
}
