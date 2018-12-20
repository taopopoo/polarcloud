package mining

import (
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"sync/atomic"
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

type Group struct {
	PreGroup  *Group   //前置组
	NextGroup *Group   //下一个组
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
	添加一个区块
	只能连续添加区块高度更高的区块
*/
func AddBlock(bh *BlockHead, txs *[]TxItr) bool {

	chain := forks.AddBlock(bh)
	if chain == nil {
		//		fmt.Println("111111111111添加一个区块不连续")
		//1.区块不连续.
		//2.产生了分叉.
		//3.本节点内存不同步.
		return false
	}
	fmt.Println("添加一个区块", bh.Height, hex.EncodeToString(bh.Hash))

	//计算余额
	bhvo := &BlockHeadVO{BH: bh, Txs: *txs}
	chain.balance.CountBalanceForBlock(bhvo)

	//	depositTxs := make([]TxItr, 0)
	for _, one := range *txs {
		switch one.Class() {
		//过滤见证人押金交易，添加见证人
		case config.Wallet_tx_type_deposit_in:
			//			depositTxs = append(depositTxs, one)
			addr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
			if err != nil {
				continue
			}
			score := (*one.GetVout())[0].Value
			chain.witnessBackup.addWitness(addr, score)
		case config.Wallet_tx_type_deposit_out:
			addr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
			if err != nil {
				continue
			}
			chain.witnessBackup.DelWitness(addr)
			//			//如果是自己，则删除
			//			if chain.balance.depositin == nil {
			//				continue
			//			}
			//			if !bytes.Equal(*addr, *chain.balance.depositin.Addr) {
			//				continue
			//			}
			//			chain.balance.depositin = nil
		case config.Wallet_tx_type_vote_in:
			//			depositTxs = append(depositTxs, one)
			voteAddr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
			if err != nil {
				continue
			}
			score := (*one.GetVout())[0].Value
			votein := one.(*Tx_vote_in)

			chain.witnessBackup.addVote(&votein.Vote.Address, voteAddr, score)
		case config.Wallet_tx_type_vote_out:

			for _, oneVin := range *one.GetVin() {

				bs, err := db.Find(oneVin.Txid)
				if err != nil {
					//TODO 不能找到上一个交易，程序出错退出
					continue
				}
				txItr, err := ParseTxBase(bs)
				if err != nil {
					//TODO 不能解析上一个交易，程序出错退出
					continue
				}
				votein := txItr.(*Tx_vote_in)
				score := votein.Vout[oneVin.Vout].Value
				voteAddr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
				if err != nil {
					continue
				}
				chain.witnessBackup.DelVote(&votein.Vote.Address, voteAddr, score)
			}

			//			//如果是自己，则删除
			//			if chain.balance.depositin == nil {
			//				continue
			//			}
			//			if !bytes.Equal(*addr, *chain.balance.depositin.Addr) {
			//				continue
			//			}
			//			chain.balance.depositin = nil
		}
	}

	chain.witnessChain.SetWitnessBlock(chain.GetLastBlock())

	//删除已经打包了的交易
	chain.transactionManager.DelTx(bhvo.Txs)

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
