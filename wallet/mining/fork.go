/*
	区块分叉管理
*/
package mining

import (
	"encoding/hex"
	"fmt"
	"sync"
	"yunpan/wallet/db"
)

type forks struct {
	HeightBlock *Block    //最高区块引用
	HashMap     *sync.Map //保存每个分叉最高区块hash。key:string=区块hash;value:*Block=区块引用;
	//	blocks      []*Block  //保存着多个分叉，区块高度最高的排名靠前。
}

//func (this forks) Len() int {
//	return len(this)
//}

//func (this forks) Less(i, j int) bool {
//	return this[i].Height < this[j].Height
//}

//func (this forks) Swap(i, j int) {
//	this[i], this[j] = this[j], this[i]
//}

//func (this *forks) Order() {
//this.HashMap.Range()
//}

/*
	添加新的区块到分叉中
*/
func (this *forks) AddBlock(bh *BlockHead) *Block {
	newBlock := new(Block)
	newBlock.Id = bh.Hash
	newBlock.Height = bh.Height
	newBlock.NextBlock = make([]*Block, 0)

	//首个区块
	//TODO 首个区块的判断标准
	if this.HeightBlock == nil {
		newGroup := new(Group)
		newGroup.Height = bh.GroupHeight
		newGroup.Blocks = []*Block{newBlock}
		newBlock.Group = newGroup
		this.HashMap.Store(hex.EncodeToString(newBlock.Id), newBlock)
		this.HeightBlock = newBlock
		return newBlock
	}

	var beforeBlock *Block
	beforBlockHash := hex.EncodeToString(bh.Previousblockhash)
	blockItr, ok := this.HashMap.Load(beforBlockHash)
	if ok {
		beforeBlock = blockItr.(*Block)
	} else {
		//从数据库中其他节点建立的分叉
		bs, err := db.Find(bh.Previousblockhash)
		if err != nil {
			fmt.Println("添加的区块高度不连续")
			return nil
		}
		_, err = ParseBlockHead(bs)
		if err != nil {
			return nil
		}

		//TODO 产生了新的分叉，考虑加载这个分叉之前的所有块

	}

	newBlock.PreBlock = beforeBlock
	beforeBlock.NextBlock = append(beforeBlock.NextBlock, newBlock)
	this.HashMap.Store(hex.EncodeToString(newBlock.Id), newBlock)
	//TODO 暂时不删除旧区块，保存所有区块，保证所有区块能在内存中查到，避免从数据库中加载时候要加载之前的多个块.
	//	this.HashMap.Delete(beforBlockHash)

	//找出最高区块
	if newBlock.Height > this.HeightBlock.Height {
		this.HeightBlock = newBlock
	}

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

	return newBlock
}

/*
	创建一个新的分叉管理器
*/
func NewForks() *forks {
	return &forks{
		//			HeightBlock *Block    //最高区块引用
		HashMap: new(sync.Map), //区块hash对应的区块引用。key:string=区块hash;value:*Block=区块引用;
	}
}
