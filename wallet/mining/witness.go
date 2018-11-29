package mining

import (
	"bytes"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"sync"
	"time"
)

/*
	见证人链
	投票竞选出来的见证人加入链中，但是没有加入组
	当交了押金后，被分配到见证人组
*/
type WitnessChain struct {
	beforeGroup          *WitnessGroup //上一组见证人
	group                *WitnessGroup //正在出块的见证人组
	firstWitnessNotGroup *Witness      //首个未分配组的见证人引用
	lastWitnessNotGroup  *Witness      //最后一个未分配组的见证人引用
}

/*
	见证人组
*/
type WitnessGroup struct {
	Task      bool          //是否已经定时出块
	PreGroup  *WitnessGroup //上一个组
	NextGroup *WitnessGroup //下一个组
	Height    uint64        //见证人组高度
	Witness   []*Witness    //本组见证人列表
}

/*
	见证人
*/
type Witness struct {
	Group       *WitnessGroup    //
	PreWitness  *Witness         //上一个见证人
	NextWitness *Witness         //下一个见证人
	DepositId   []byte           //押金交易id
	Addr        *utils.Multihash //见证人地址
	Block       *Block           //见证人生产的块
	ElectionMap *sync.Map        //本块中交易押金投票数 key:string=投票者地址;value:*BallotTicket=选票;
}

/*
	依次获取n个未分配组的见证人，构建一个新的见证人组
*/
func (this *WitnessChain) BuildGroupForNum() {
	//先检查人数是否足够
	witnessGroup := this.GetOneGroupWitness()

	//见证人太少，从备用见证人中评选出新的见证人
	if len(witnessGroup) < config.Mining_group_min {
		//从备用见证人中构建见证人组
		witness := CreateWitnessGroup()
		if witness == nil {
			//备用见证人数量不够
			fmt.Println("备用见证人数量不够")
			this.PrintWitnessList()
			//本组没有见证人，将当前见证人组设置为空
			this.beforeGroup = this.group
			this.group = nil
			return
		} else {
			this.AddWitness(witness)
		}
	}

	witnessGroup = this.GetOneGroupWitness()

	if this.group != nil {
		this.beforeGroup = this.group
	}

	blockHeight := chain.GetLastBlock().Group.Height + 1

	newGroup := &WitnessGroup{
		PreGroup: this.beforeGroup, //上一个组
		Height:   blockHeight,      //
		Witness:  witnessGroup,     //本组见证人列表
	}
	for i, _ := range witnessGroup {
		witnessGroup[i].Group = newGroup
	}
	this.group = newGroup
}

/*
	获取一组新的见证人组
	从未分配组的见证人中按顺序获取一个组的见证人
*/
func (this *WitnessChain) GetOneGroupWitness() []*Witness {
	witnessGroup := make([]*Witness, 0)
	if this.firstWitnessNotGroup != nil {
		tempWitness := this.firstWitnessNotGroup
		for i := 0; i < config.Mining_group_max; i++ {
			witnessGroup = append(witnessGroup, tempWitness)
			if tempWitness.NextWitness == nil {
				break
			}
			tempWitness = tempWitness.NextWitness
		}
	}
	return witnessGroup
}

/*
	获取
*/
//func (this *WitnessChain) BuildWitnessGroupForNum(n, height uint64) bool {

//}

/*
	按新组数量构建一个新的见证人组
	@n         uint64    组人数
	@height    uint64    组高度
*/
//func (this *WitnessChain) BuildWitnessGroupForNum(n, height uint64) bool {
//	if height == 0 {
//		//首个组需要指定组高度，后面的组不需要指定
//		if this.group == nil {
//			return false
//		}
//		group := this.group
//		for {
//			if group.NextGroup == nil {
//				break
//			}
//			group = group.NextGroup
//		}
//		height = group.Height + 1
//	}
//	fmt.Println("本次见证人组 人数", n, "组高度", height)
//	start := this.lastWitness
//	if this.group == nil {
//		for {
//			if start.PreWitness == nil {
//				break
//			}
//			start = start.PreWitness
//		}
//	} else {
//		if this.group.Witness[len(this.group.Witness)-1].NextWitness == nil {
//			return false
//		}
//		start = this.group.Witness[len(this.group.Witness)-1].NextWitness
//	}
//	witness := make([]*Witness, 0)
//	for i := 0; i < int(n); i++ {
//		witness = append(witness, start)
//		if start.NextWitness == nil {
//			break
//		} else {
//			start = start.NextWitness
//		}
//	}
//	if len(witness) < int(n) {
//		return false
//	}
//	newGroup := WitnessGroup{
//		Height:  height,  //
//		Witness: witness, //本组见证人列表
//	}
//	for i, _ := range newGroup.Witness {
//		newGroup.Witness[i].Group = &newGroup
//	}
//	if this.group == nil {
//		this.group = &newGroup
//		fmt.Println("构建新的见证人组")

//	} else {
//		//把新的组保存链接到见证人组后面
//		group := this.group
//		for {
//			if group.NextGroup == nil {
//				break
//			}
//			group = group.NextGroup
//		}
//		group.NextGroup = &newGroup
//		newGroup.PreGroup = group

//		//切换到下一个组
//		this.group = this.group.NextGroup

//	}

//	Mining()

//	return true
//}

///*
//	构建一个新的见证人组
//	从备用见证人中抽出最少的个见证人组成见证人组
//*/
//func (this *WitnessChain) BuildWitnessGroup() bool {
//	//	groupHeight := uint64(0)
//	if this.group == nil {
//		return false
//	}
//	//	start := this.lastWitness
//	start := this.group.Witness[len(this.group.Witness)-1].NextWitness
//	groupHeight := this.group.Height + 1

//	witness := make([]*Witness, 0)
//	for i := 0; i < config.Mining_group_max; i++ {
//		//		fmt.Println("+++构建本组见证人", start.Addr.B58String())
//		witness = append(witness, start)
//		if start.NextWitness == nil {
//			break
//		} else {
//			start = start.NextWitness
//		}
//	}
//	if len(witness) < config.Mining_group_min {
//		return false
//	}
//	newGroup := WitnessGroup{
//		Height:  groupHeight,
//		Witness: witness, //本组见证人列表
//	}
//	for i, _ := range newGroup.Witness {
//		newGroup.Witness[i].Group = &newGroup
//	}
//	if this.group == nil {
//		this.group = &newGroup
//		return true
//	}
//	//	this.group.NextGroup = &newGroup
//	//	newGroup.PreGroup = this.group
//	//	this.group = &newGroup
//	//把新的组保存链接到见证人组后面
//	group := this.group
//	for {
//		if group.NextGroup == nil {
//			break
//		}
//		group = group.NextGroup
//	}
//	group.NextGroup = &newGroup
//	newGroup.PreGroup = group

//	//切换到下一个组
//	this.group = this.group.NextGroup
//	return true
//}

/*
	添加见证人，依次添加
*/
func (this *WitnessChain) AddWitness(newwitness *Witness) {
	//	fmt.Println("++添加备用见证人", newwitness.Addr.B58String())
	//	if this.lastWitness == nil {
	//		this.lastWitness = newwitness
	//		return
	//	}
	//	this.lastWitness.NextWitness = newwitness
	//	newwitness.PreWitness = this.lastWitness
	//	this.lastWitness = newwitness
	if this.firstWitnessNotGroup == nil {
		this.firstWitnessNotGroup = newwitness
		this.lastWitnessNotGroup = newwitness
	} else {
		this.lastWitnessNotGroup.NextWitness = newwitness
	}

	//让this.lastWitnessNotGroup保持引用最后一个见证人
	for {
		if this.lastWitnessNotGroup.NextWitness == nil {
			break
		}
		this.lastWitnessNotGroup = this.lastWitnessNotGroup.NextWitness
	}
}

/*
	获得正在出块的见证人组
*/
//func (this *WitnessChain) GetWitness() *WitnessGroup {
//	return this.group
//}

/*
	获取所有准备出块的见证人
*/
//func (this *WitnessChain) GetBackupWitness() *Witness {
//	next := this.lastWitness
//	for {
//		if next == nil || next.PreWitness == nil {
//			break
//		}
//		if next.PreWitness.Block != nil {
//			break
//		}
//		next = next.PreWitness
//	}
//	return next
//}

/*
	打印见证人列表
*/
func (this *WitnessChain) PrintWitnessList() {
	//打印未分组的见证人列表
	PrintWitnessBackup()

	//	start := this.firstWitnessNotGroup
	//	for {
	//		if start.PreWitness == nil {
	//			break
	//		}
	//		start = start.PreWitness
	//	}
	//	for {
	//		groupHeight := uint64(0)
	//		if start.Group != nil {
	//			groupHeight = start.Group.Height
	//		}

	//		if start.Block == nil {
	//			fmt.Println("打印见证人列表", hex.EncodeToString(start.DepositId), start.Addr.B58String(), "组高度", groupHeight)
	//		} else {
	//			fmt.Println("打印见证人列表", hex.EncodeToString(start.DepositId), start.Addr.B58String(), "组高度", groupHeight, "块高度", start.Block.Height)
	//		}
	//		if start.NextWitness == nil {
	//			break
	//		}
	//		start = start.NextWitness
	//	}
}

//func (this *WitnessChain) GetWitness() []*Witness {
//	group := this.group
//	for {
//		if group.PreGroup == nil {
//			break
//		}
//		if group.PreGroup.Witness[0].Block != nil {
//			break
//		}
//		group = group.PreGroup
//	}
//	return group.Witness
//}

/*
	设置见证人生成的块
	只能设置当前组，不能设置其他组
	当本组所有见证人都出块了，将当前组见证人的变量指针修改为下一组见证人
	@return    bool    是否设置成功
	@return    bool    是否是本组的最后一个块
*/
func (this *WitnessChain) SetWitnessBlock(block *Block) (bool, bool) {
	if this.group == nil {
		this.BuildGroupForNum()
		return false, false
	}
	group := this.group
	if group.Height != block.Group.Height {
		return false, false
	}
	for i, one := range group.Witness {
		bh, err := block.Load()
		if err != nil {
			return false, false
		}
		if bytes.Equal(*one.Addr, bh.Witness) {
			one.Block = block
			if len(group.Witness) == i+1 {
				//是该组最后一个出块
				this.BuildGroupForNum()
				return true, true
			}
			return true, false
		}
	}
	return false, false
}

/*
	统计组中见证人投票数量
	满足最少见证人组中的人数，才能出块
	从备用见证人中获取第一个备用分组，并且分组中只要有3个见证人被投票
	@return    map[string]uint64    每个见证人的票数,key:string=见证人地址;value:uint64=票数;
*/
//func (this *WitnessChain) CountWitness() (map[string]uint64, map[string]uint64) {
//	//	fmt.Println("----开始统计见证人数\n")
//	countGroup := make(map[string]uint64)
//	witness := this.GetBackupWitness()
//	for i := 0; i < config.Mining_group_max; i++ {
//		//		fmt.Println("----开始统计见证人数 1111111")
//		if witness == nil {
//			break
//		}
//		if witness.ElectionMap == nil {
//			continue
//		}
//		count := uint64(0)
//		witness.ElectionMap.Range(func(k, v interface{}) bool { count++; return true })
//		countGroup[witness.Addr.B58String()] = count
//		witness = witness.NextWitness
//	}
//	rest := make(map[string]uint64)
//	//剩下的
//	for {
//		//		fmt.Println("----开始统计见证人数 222222222222")
//		if witness == nil {
//			break
//		}
//		if witness.ElectionMap == nil {
//			if witness.NextWitness != nil {
//				witness = witness.NextWitness
//				continue
//			}
//			break
//		}
//		count := uint64(0)
//		witness.ElectionMap.Range(func(k, v interface{}) bool { count++; return true })
//		rest[witness.Addr.B58String()] = count
//		witness = witness.NextWitness
//	}
//	return countGroup, rest

//	//	count := new(sync.Map)
//	//	for _, one := range this.Blocks {
//	//		if one.ElectionMap == nil {
//	//			continue
//	//		}
//	//		one.ElectionMap.Range(func(k, v interface{}) bool {
//	//			count.Store(k, v)
//	//			//			fmt.Println("----统计见证人数", k, v)
//	//			return true
//	//		})
//	//	}
//	//	result := make(map[string]uint64) // uint64(0)
//	//	count.Range(func(k, v interface{}) bool {
//	//		witness := k.(string)
//	//		bts := v.(*sync.Map)
//	//		total := uint64(0)
//	//		bts.Range(func(k, v interface{}) bool { total++; return true })
//	//		result[witness] = total
//	//		return true
//	//	})
//	//	return result
//}

/*
	在当前组中查找见证人
*/
//func (this *WitnessGroup) FindWitness(witness *utils.Multihash) bool {
//	for _, one := range this.Witness {
//		if one.Block == nil {
//			continue
//		}
//		for _, tx := range one.Block.DepositTx {
//			deposit := tx.(*Tx_deposit_in)
//			if witness.B58String() == deposit.GetWitness().B58String() {
//				return true
//			}
//		}
//	}
//	return false
//}

/*
	构建本组中的见证人出块奖励
	按股权分配
	只有见证人方式出块才统计
*/
func (this *WitnessGroup) CountReward() *Tx_reward {
	//统计本组的出块奖励
	vouts := make([]Vout, 0)
	for _, one := range this.Witness {
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
		Type:       config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vout_total: uint64(len(vouts)),           //输出交易数量
		Vout:       vouts,                        //交易输出
	}

	txReward := Tx_reward{
		TxBase:     base,
		CreateTime: time.Now().Unix(), //创建时间
	}
	txReward.BuildHash()
	return &txReward
}
