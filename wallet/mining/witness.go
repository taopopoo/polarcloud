package mining

import (
	"bytes"
	"fmt"
	"math/big"
	"polarcloud/config"
	"polarcloud/core/utils"
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
	Addr        *utils.Multihash //见证人地址
	Block       *Block           //见证人生产的块
	Score       uint64           //押金
	Votes       []*VoteScore     //投票人和押金
	//	DepositId   []byte           //押金交易id
	//	ElectionMap *sync.Map        //本块中交易押金投票数 key:string=投票者地址;value:*BallotTicket=选票;
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
	组人数乘以每块奖励，再分配给实际出块的人
*/
func (this *WitnessGroup) CountReward(txs []TxItr) *Tx_reward {
	//计算交易手续费
	allGas := uint64(0)
	for _, one := range txs {
		allGas = allGas + one.GetGas()
	}

	//统计本组股权
	allPos := uint64(0)    //股权数量
	allReward := uint64(0) //本组奖励数量
	for _, one := range this.Witness {
		if one.Block == nil {
			continue
		}
		allPos = allPos + (one.Score * 2) //计算股权的时候，见证人的股权要乘以2
		for _, vote := range one.Votes {
			allPos = allPos + vote.Score
		}

		//第一个块产出80个币
		//每增加一定块数，产出减半，直到为0
		//最多减半9次，第10次减半后产出为0
		oneReward := uint64(config.Mining_reward)
		n := one.Block.Height / config.Mining_block_cycle
		if n < 10 {
			for i := uint64(0); i < n; i++ {
				oneReward = oneReward / 2
			}
		} else {
			oneReward = 0
		}
		allReward = allReward + oneReward
	}
	allReward = allReward + allGas
	//	fmt.Println("本组所有股权占股数", allPos)
	//	onePos := allReward / allPos

	//开始分配奖励
	countReward := uint64(0)
	vouts := make([]Vout, 0)
	for _, one := range this.Witness {
		if one.Block == nil {
			continue
		}
		//见证人自己的股权和投票者的股权的总和
		groupPos := one.Score * 2
		for _, two := range one.Votes {
			groupPos = groupPos + two.Score
		}
		//计算见证人和投票者的收益总和
		temp := new(big.Int).Mul(big.NewInt(int64(allReward)), big.NewInt(int64(groupPos)))
		groupReward := new(big.Int).Div(temp, big.NewInt(int64(allPos)))

		voteAllReward := uint64(0)
		for _, two := range one.Votes {
			temp = new(big.Int).Mul(groupReward, big.NewInt(int64(two.Score)))
			value := new(big.Int).Div(temp, big.NewInt(int64(groupPos)))
			//			fmt.Println("投票者收益", temp.Uint64(), two.Score, value.Uint64())
			//奖励为0的矿工交易不写入区块
			if uint64(value.Int64()) <= 0 {
				continue
			}
			vout := Vout{
				Value:   uint64(value.Int64()),
				Address: *two.Addr,
			}
			vouts = append(vouts, vout)
			voteAllReward = voteAllReward + uint64(value.Int64())
		}

		//		value := new(big.Int).Div(temp, big.NewInt(int64(one.Score*2)))

		//平均数不能被整除时候，剩下的给出块的见证人
		value := uint64(groupReward.Int64()) - voteAllReward
		//奖励为0的矿工交易不写入区块
		if value <= 0 {
			countReward = countReward + voteAllReward
			continue
		}
		vout := Vout{
			Value:   value,
			Address: *one.Addr,
		}
		vouts = append(vouts, vout)
		countReward = countReward + voteAllReward + value
	}
	//平均数不能被整除时候，剩下的给最后一个出块的见证人
	vouts[len(vouts)-1].Value = vouts[len(vouts)-1].Value + (allReward - countReward)

	for _, one := range vouts {
		fmt.Println("查看矿工结算信息", one.Value)
	}

	//	//统计本组的出块奖励
	//	//	allReward := uint64(0)
	//	vouts := make([]Vout, 0)
	//	for _, one := range this.Witness {
	//		if one.Block == nil {
	//			continue
	//		}
	//		vout := Vout{
	//			Value:   config.Mining_reward,
	//			Address: *one.Addr,
	//		}
	//		vouts = append(vouts, vout)

	//		oneReward := uint64(config.Mining_reward)
	//		n := one.Block.Height / config.Mining_block_cycle
	//		for i := uint64(0); i < n; i++ {
	//			oneReward = oneReward / 2
	//		}
	//		allReward = allReward + oneReward
	//		fmt.Println("打印投票股权", one.Votes)
	//	}

	//	//平均分给每一个出块的人，保留整数，去掉小数点
	//	oneReward := allReward / uint64(len(vouts))
	//	for i, _ := range vouts {
	//		vouts[i].Value = uint64(oneReward)
	//	}
	//	//平均数不能被整除时候，剩下的给最后一个出块的见证人
	//	lastReward := uint64(allReward - (oneReward * uint64(len(vouts)-1)))
	//	vouts[len(vouts)-1].Value = lastReward

	base := TxBase{
		Type:       config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vout_total: uint64(len(vouts)),           //输出交易数量
		Vout:       vouts,                        //交易输出
		CreateTime: time.Now().Unix(),            //创建时间
	}

	txReward := Tx_reward{
		TxBase: base,
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
