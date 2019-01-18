/*
	区块分叉管理
*/
package mining

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"sync"
)

var forks = new(Forks)

func init() {
	forks.chainss = new(sync.Map)
}

type Forks struct {
	Init          bool      //是否是创世节点
	StartingBlock uint64    //区块开始高度
	HighestBlock  uint64    //网络节点广播的区块最高高度
	CurrentBlock  uint64    //内存中已经同步到的区块高度
	PulledStates  uint64    //正在同步的区块高度
	LongChain     *Chain    //最高区块引用
	chainss       *sync.Map //保存各个分叉链key:string=链最高块hash;value:*Block=各个分叉链引用;
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
//func (this *Forks) GetChain(beforeHash string) *Chain {
//	chainItr, ok := this.chains.Load(beforeHash)
//	if ok {
//		chain := chainItr.(*Chain)
//		return chain
//	}
//	return nil
//}

/*
	添加新的区块到分叉中
	返回(true,chain)  区块添加到分叉链上
	返回(false,chain) 区块添加到主链上的
	返回(true,nil)    区块分叉超过了区块确认数量
	@return    bool      是否有分叉
	@return    *Chain    区块添加到的链
*/
func (this *Forks) AddBlock(bh *BlockHead, txs *[]TxItr) {
	fmt.Println("加载区块到内存", bh.Height)
	newBlock := new(Block)
	newBlock.Id = bh.Hash
	newBlock.Height = bh.Height
	newBlock.PreBlock = make([]*Block, 0)
	newBlock.NextBlock = make([]*Block, 0)

	//系统中还没有链，创建首个链
	if this.LongChain == nil {
		if this.StartingBlock < bh.Height {
			fmt.Println("创建首个链", this.StartingBlock, bh.Height)
			return
		}
		fmt.Println("初始化链端")
		newGroup := new(Group)
		newGroup.Height = bh.GroupHeight
		newGroup.Blocks = []*Block{newBlock}
		newGroup.NextGroup = make([]*Group, 0)
		newBlock.Group = newGroup
		newChain := NewChain(newBlock)
		this.chainss.Store(hex.EncodeToString(newBlock.Id), newBlock)
		this.LongChain = newChain
		newChain.CountBlock(bh, txs)
		newChain.lastBlock = newBlock
		return
	}

	//获取本块所在的链
	beforeBlockHash := hex.EncodeToString(bh.Previousblockhash[0])
	chainItr, ok := this.chainss.Load(beforeBlockHash)
	if ok {
		fmt.Println("添加区块 1111111")
		beforeBlock := chainItr.(*Block)
		//		beforeBlock := chain.GetLastBlock()

		newBlock.PreBlock = append(newBlock.PreBlock, beforeBlock)
		beforeBlock.NextBlock = append(beforeBlock.NextBlock, newBlock)
		beforeBlock.FlashNextblockhash()

		//新的区块组
		if bh.GroupHeight > beforeBlock.Group.Height {

			fmt.Println("添加区块 222222222")
			newGroup := new(Group)
			newGroup.Height = bh.GroupHeight
			newGroup.Blocks = []*Block{newBlock}
			newGroup.NextGroup = make([]*Group, 0)
			newGroup.PreGroup = beforeBlock.Group
			beforeBlock.Group.NextGroup = append(beforeBlock.Group.NextGroup, newGroup)
			newBlock.Group = newGroup
		} else {

			fmt.Println("添加区块 3333333333")
			//保存到旧的区块组中
			beforeBlock.Group.Blocks = append(beforeBlock.Group.Blocks, newBlock)
			newBlock.Group = beforeBlock.Group
		}

		//		chain.lastBlock = newBlock
		this.chainss.Store(hex.EncodeToString(newBlock.Id), newBlock)
		this.chainss.Delete(beforeBlockHash)

		//找出最高区块
		//		if newBlock.Height > this.LongChain.lastBlock.Height {
		//			this.LongChain = chain
		//		}

		//判断是否保存到主链上，保存在主链上，则统计收益和见证人
		if bytes.Equal(bh.Previousblockhash[0], this.LongChain.GetLastBlock().Id) {

			fmt.Println("添加区块 444444444444")
			this.LongChain.lastBlock = newBlock
			this.LongChain.CountBlock(bh, txs)
		}

		//		return chain
		return
	}
	fmt.Println("添加区块 5555555555")
	//这里创建新的分叉，只能从未确认的块中分叉
	//	var chain *Chain
	this.chainss.Range(func(k, v interface{}) bool {
		//		chain := v.(*Chain)
		//		oneBlock := chain.GetLastBlock()
		oneBlock := v.(*Block)
		groupHeight := oneBlock.Group.Height
		for i := 0; i < config.Block_confirm; {

			if bytes.Equal(oneBlock.Id, bh.Previousblockhash[0]) {
				fmt.Println("添加区块 6666666666666")
				newBlock.PreBlock = append(newBlock.PreBlock, oneBlock)
				oneBlock.NextBlock = append(oneBlock.NextBlock, newBlock)
				oneBlock.FlashNextblockhash()

				newGroup := new(Group)
				newGroup.Height = bh.GroupHeight
				newGroup.NextGroup = make([]*Group, 0)
				newGroup.PreGroup = oneBlock.PreBlock[0].Group.PreGroup
				oneBlock.PreBlock[0].Group.NextGroup = append(oneBlock.PreBlock[0].Group.NextGroup, newGroup)
				newBlock.Group = newGroup
				//找到分叉的区块
				if bh.GroupHeight > oneBlock.Group.Height {
					fmt.Println("添加区块 777777777")
					//新的区块组
					newGroup.Blocks = []*Block{newBlock}

				} else {
					fmt.Println("添加区块 88888888888")
					//不是新的区块组，需要克隆区块组
					newGroup.Blocks = make([]*Block, 0)
					newGroup.Blocks = append(newGroup.Blocks, oneBlock.Group.Blocks...)
					newGroup.Blocks = append(newGroup.Blocks, newBlock)

				}
				this.chainss.Store(hex.EncodeToString(newBlock.Id), newBlock)
				return false
			}
			if oneBlock.PreBlock == nil || len(oneBlock.PreBlock) <= 0 {
				break
			}
			oneBlock = oneBlock.PreBlock[0]
			if oneBlock.Group.Height < groupHeight {
				i++
				groupHeight = oneBlock.Group.Height
			}
		}
		return true
	})

	return

	//TODO 将失效的分叉删除，将未确认的区块前的分叉删除
}

/*
	判断分叉链是否长于当前最长链
	如果分叉链长于当前链，则找出分叉链从主链上的分叉路径
	@chain        *Chain      最高链
	@forkChain    *Chain      分叉链
	@return       bool        是否分叉链最长
	@return       [][]byte    分叉链区块头hash
*/
func (this *Forks) ContrastLongBlock(chain *Chain) (ok bool, hs [][]byte) {
	//判断最新块是不是添加在最长链上
	if bytes.Equal(this.LongChain.GetLastBlock().Id, chain.GetLastBlock().Id) {
		//
		return false, nil
	} else {
		if this.LongChain.GetLastBlock().Height >= chain.GetLastBlock().Height {
			return false, nil
		}
		//保存主链所有未确认块hash
		hs := make([][]byte, 0)
		oneBlock := this.LongChain.GetLastBlock()
		groupHeight := oneBlock.Group.Height
		for i := 0; i < config.Block_confirm; {
			hs = append(hs, oneBlock.Id)
			oneBlock = oneBlock.PreBlock[0]
			if oneBlock.Group.Height < groupHeight {
				i++
				groupHeight = oneBlock.Group.Height
			}
		}
		//保存分叉链hash值
		forkBlockHashs := make([][]byte, 0)
		//找到主链和分叉链的分叉点
		oneBlock = this.LongChain.GetLastBlock()
		groupHeight = oneBlock.Group.Height
		//分叉链最多查找未确认的块，如果找完都未找到，则是应该是被删除的链，有问题
		for i := 0; i < config.Block_confirm; {
			forkBlockHashs = append(forkBlockHashs, oneBlock.Id)
			oneBlock = oneBlock.PreBlock[0]
			if len(oneBlock.NextBlock) > 1 {
				//找到分叉点，和主链上的块对比
				for _, one := range hs {
					if bytes.Equal(one, oneBlock.Id) {
						//找到了分叉点
						return true, forkBlockHashs
					}
				}
			}

			if oneBlock.Group.Height < groupHeight {
				i++
				groupHeight = oneBlock.Group.Height
			}
		}
		return true, nil
	}

}

/*
	查找目标链和主链的交叉点，返回分叉链区块
	@return    uint64      主链回滚区块数量
	@return    [][]byte    新分叉主链区块路径，从区块高度由高到低的顺序返回区块hash
*/
func (this *Forks) FindIntersection(forkBlock *Block) (uint64, [][]byte) {
	//保存主链所有未确认块hash
	hs := make([][]byte, 0)
	oneBlock := this.LongChain.GetLastBlock()
	groupHeight := oneBlock.Group.Height
	for i := 0; i < config.Block_confirm; {
		hs = append(hs, oneBlock.Id)
		oneBlock = oneBlock.PreBlock[0]
		if oneBlock.Group.Height < groupHeight {
			i++
			groupHeight = oneBlock.Group.Height
		}
	}
	//保存分叉链hash值
	forkBlockHashs := make([][]byte, 0)
	//找到主链和分叉链的分叉点
	oneBlock = forkBlock
	groupHeight = oneBlock.Group.Height
	//分叉链最多查找未确认的块，如果找完都未找到，则是应该是被删除的链，有问题
	for i := 0; i < config.Block_confirm; {
		forkBlockHashs = append(forkBlockHashs, oneBlock.Id)
		oneBlock = oneBlock.PreBlock[0]
		if len(oneBlock.NextBlock) > 1 {
			//找到分叉点，和主链上的块对比
			for j, one := range hs {
				if bytes.Equal(one, oneBlock.Id) {
					//找到了分叉点
					return uint64(j + 1), forkBlockHashs
				}
			}
		}

		if oneBlock.Group.Height < groupHeight {
			i++
			groupHeight = oneBlock.Group.Height
		}
	}
	return 0, forkBlockHashs
}

/*
	选择最长链，分叉链最长就回滚
*/
func (this *Forks) SelectLongChain() {
	var n uint64
	var hs [][]byte
	this.chainss.Range(func(k, v interface{}) bool {
		//		chain := v.(*Chain)
		block := v.(*Block)
		if bytes.Equal(this.LongChain.GetLastBlock().Id, block.Id) {
			return true
		}
		if block.Height > this.LongChain.GetLastBlock().Height {
			fmt.Println("选择最长区块", block.Height, this.LongChain.GetLastBlock().Height)
			n, hs = this.FindIntersection(block)
			//找到分叉点区块
			return false
		}

		return true
	})
	if n <= 0 {
		return
	}

	fmt.Println("开始回滚区块", hs)
	//找到分叉点区块
	forkBlock := this.LongChain.GetLastBlock()
	for i := uint64(0); i < n; i++ {
		forkBlock = forkBlock.PreBlock[0]
	}
	//验证分叉点区块
	if !bytes.Equal(forkBlock.Id, hs[len(hs)-1]) {
		//验证不通过
		fmt.Println("验证回滚的区块分叉点，不通过")
		return
	}
	//开始回滚
	fmt.Println("开始回滚区块")
	this.rollBackBlocks(n)
	//把分叉区块连接的下一个块排序，index为0的是最长链

	//回滚后重新加载新的区块，这些区块只统计见证人投票
	fmt.Println("开始加载分叉链区块")
	this.CountForkBlocks(n, hs)

}

/*
	区块回滚，当链分叉的时候，需要回滚区块，添加最长链的区块
	@n    uint64    回滚多少个区块
*/
func (this *Forks) rollBackBlocks(n uint64) {
	block := this.LongChain.GetLastBlock()
	for i := uint64(0); i < n; i++ {
		this.LongChain.RollbackBlock(block.Height)
	}

}

/*
	统计分叉块
	@bh    *BlockHead    最新区块
	@n    uint64    回滚多少个区块
*/
func (this *Forks) CountForkBlocks(n uint64, hs [][]byte) {
	block := this.LongChain.GetLastBlock()
	for i := uint64(0); i < n; i++ {
		block = block.PreBlock[0]
	}
	for _, hbs := range hs {
		has := false
		for _, one := range block.NextBlock {
			if bytes.Equal(one.Id, hbs) {
				//TODO 把本块hash修改排序，排在第一位.
				has = true
				bh, txs, err := one.LoadTxs()
				if err != nil {
					fmt.Println("回滚后重新统计分叉链出错-加载区块信息错误", err)
					return
				}
				this.LongChain.CountBlock(bh, txs)
				break
			}
		}
		if !has {
			fmt.Println("程序出错，没找到统计的区块")
		}
	}
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
