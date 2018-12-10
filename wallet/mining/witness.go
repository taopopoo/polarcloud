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
	witnessBackup        *WitnessBackup //
	beforeGroup          *WitnessGroup  //上一组见证人
	group                *WitnessGroup  //正在出块的见证人组
	firstWitnessNotGroup *Witness       //首个未分配组的见证人引用
	lastWitnessNotGroup  *Witness       //最后一个未分配组的见证人引用
}

func NewWitnessChain(wb *WitnessBackup) *WitnessChain {
	return &WitnessChain{
		witnessBackup: wb,
	}
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
	//	fmt.Println("获取的备用见证人数", len(witnessGroup))

	//见证人太少，从备用见证人中评选出新的见证人
	if len(witnessGroup) < config.Mining_group_min {
		//从备用见证人中构建见证人组
		witness := this.witnessBackup.CreateWitnessGroup()
		if witness == nil {
			//备用见证人数量不够
			//			fmt.Println("-----------备用见证人数量不够")
			this.PrintWitnessList()
			//本组没有见证人，将当前见证人组设置为空
			this.beforeGroup = this.group
			this.group = nil
			return
		} else {
			//			fmt.Println("------------备用见证人数量足够", len(witnessGroup))
			this.AddWitness(witness)
		}
	}

	witnessGroup = this.GetOneGroupWitness()
	//	fmt.Println("---------再次获取见证人数量", len(witnessGroup))

	if this.group != nil {
		this.beforeGroup = this.group
	}

	blockHeight := forks.GetLongChain().GetLastBlock().Group.Height + 1

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
			tempWitness = tempWitness.NextWitness
			if tempWitness == nil {
				break
			}
		}
		this.firstWitnessNotGroup = tempWitness
	}
	return witnessGroup
}

/*
	添加见证人，依次添加
*/
func (this *WitnessChain) AddWitness(newwitness *Witness) {

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
	打印见证人列表
*/
func (this *WitnessChain) PrintWitnessList() {
	//打印未分组的见证人列表
	this.witnessBackup.PrintWitnessBackup()

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
				fmt.Println("=========是该组最后一个块，需要从新分组")
				this.BuildGroupForNum()
				return true, true
			}
			return true, false
		}
	}
	return false, false
}

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

/*
	判断是否是本组首个见证人出块
*/
func (this *WitnessGroup) FirstWitness() bool {
	for _, one := range this.Witness {
		if one.Block != nil {
			return false
		}
	}
	return true
}
