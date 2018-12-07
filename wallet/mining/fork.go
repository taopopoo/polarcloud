/*
	区块分叉管理
*/
package mining

import (
	"encoding/hex"
	"fmt"
	"sync"
)

var forks = new(Forks)

func init() {
	forks.chains = new(sync.Map)
}

type Forks struct {
	Init          bool      //是否是创世节点
	StartingBlock uint64    //区块开始高度
	HighestBlock  uint64    //网络节点广播的区块最高高度
	CurrentBlock  uint64    //内存中已经同步到的区块高度
	PulledStates  uint64    //正在同步的区块高度
	LongChain     *Chain    //最高区块引用
	chains        *sync.Map //保存各个分叉链key:string=链最高块hash;value:*Chain=各个分叉链引用;
}

/*
	获得最长链
*/
func (this *Forks) GetLongChain() *Chain {
	return this.LongChain
}

/*
	获得最长链
*/
func (this *Forks) GetChain(beforeHash string) *Chain {
	chainItr, ok := this.chains.Load(beforeHash)
	if ok {
		chain := chainItr.(*Chain)
		return chain
	}
	return nil
}

/*
	添加新的区块到分叉中
*/
func (this *Forks) AddBlock(bh *BlockHead) *Chain {

	newBlock := new(Block)
	newBlock.Id = bh.Hash
	newBlock.Height = bh.Height
	newBlock.NextBlock = make([]*Block, 0)

	//系统中还没有链，创建首个链
	if this.LongChain == nil {
		if this.StartingBlock < bh.Height {
			fmt.Println("创建首个链", this.StartingBlock, bh.Height)
			return nil
		}
		newGroup := new(Group)
		newGroup.Height = bh.GroupHeight
		newGroup.Blocks = []*Block{newBlock}
		newBlock.Group = newGroup
		newChain := NewChain(newBlock)
		this.chains.Store(hex.EncodeToString(newBlock.Id), newChain)
		this.LongChain = newChain
		return newChain
	}

	//获取本块所在的链
	beforeBlockHash := hex.EncodeToString(bh.Previousblockhash)
	chainItr, ok := this.chains.Load(beforeBlockHash)
	if !ok {
		//TODO 产生了新的分叉，考虑加载这个分叉之前的所有块
		//TODO 可能是脱离主网太久
		return nil
	}
	chain := chainItr.(*Chain)
	beforeBlock := chain.GetLastBlock()

	newBlock.PreBlock = beforeBlock
	beforeBlock.NextBlock = append(beforeBlock.NextBlock, newBlock)

	//新的区块组
	if bh.GroupHeight > beforeBlock.Group.Height {
		newGroup := new(Group)
		newGroup.Height = beforeBlock.Group.Height + 1
		newGroup.Blocks = []*Block{newBlock}
		newGroup.PreGroup = beforeBlock.Group
		beforeBlock.Group.NextGroup = newGroup
		newBlock.Group = newGroup
	} else {
		//保存到旧的区块组中
		beforeBlock.Group.Blocks = append(beforeBlock.Group.Blocks, newBlock)
		newBlock.Group = beforeBlock.Group
	}

	chain.lastBlock = newBlock
	this.chains.Store(hex.EncodeToString(newBlock.Id), chain)
	this.chains.Delete(beforeBlockHash)

	//找出最高区块
	if newBlock.Height > this.LongChain.lastBlock.Height {
		this.LongChain = chain
	}

	return chain
}

/*
	创建一个新的分叉管理器
*/
//func NewForks() *Forks {
//	return &Forks{
//		//			HeightBlock *Block    //最高区块引用
//		HashMap: new(sync.Map), //区块hash对应的区块引用。key:string=区块hash;value:*Block=区块引用;
//	}
//}
