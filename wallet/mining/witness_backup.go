package mining

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"sort"
	"sync"
)

//var witnessesLock = new(sync.RWMutex)
//var witnesses = make(WitnessBackup, 0)

type WitnessBackup struct {
	chain        *Chain          //
	lock         sync.RWMutex    //
	witnesses    []BackupWitness //
	witnessesMap sync.Map        //key:string=备用见证人地址;value:*BackupWitness=备用见证人;
	Vote         sync.Map        //注意：投票押金要和见证人分开，因为区块回滚的时候，恢复见证人就不方便恢复投票押金。投票押金 key:string=投票人地址;value:*VoteScore=投票人和押金;
}

func (this WitnessBackup) Len() int {
	return len(this.witnesses)
}

func (this WitnessBackup) Less(i, j int) bool {
	return this.witnesses[i].Score < this.witnesses[j].Score
}

func (this WitnessBackup) Swap(i, j int) {
	this.witnesses[i], this.witnesses[j] = this.witnesses[j], this.witnesses[i]
}

/*
	统计备用见证人和见证人投票
*/
func (this *WitnessBackup) CountWitness(txs *[]TxItr) {
	//	depositTxs := make([]TxItr, 0)
	for _, one := range *txs {
		switch one.Class() {
		//过滤见证人押金交易，添加见证人
		case config.Wallet_tx_type_deposit_in:
			//			addr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
			//			if err != nil {
			//				continue
			//			}
			//这里决定了交易输出地址才是见证人地址。
			vout := (*one.GetVout())[0]
			score := vout.Value
			this.addWitness(&vout.Address, score)
		case config.Wallet_tx_type_deposit_out:
			for _, two := range *one.GetVin() {
				addr, err := keystore.ParseHashByPubkey(two.Puk)
				if err != nil {
					continue
				}
				this.DelWitness(addr)
			}
		case config.Wallet_tx_type_vote_in:
			//			voteAddr, err := keystore.ParseHashByPubkey((*one.GetVin())[0].Puk)
			//			if err != nil {
			//				continue
			//			}
			//这里决定了交易输出地址才是见证人地址。
			//只有下标为0的输出才是押金。
			vout := (*one.GetVout())[0]
			score := vout.Value
			votein := one.(*Tx_vote_in)

			this.addVote(&votein.Vote, &vout.Address, score)
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
				//因为有可能退回金额不够手续费，所以输入中加入了其他类型交易
				if txItr.Class() != config.Wallet_tx_type_vote_in {
					continue
				}
				vout := (*txItr.GetVout())[oneVin.Vout]
				votein := txItr.(*Tx_vote_in)
				this.DelVote(&votein.Vote, &vout.Address, vout.Value)
			}

		}
	}

}

/*
	回滚统计备用见证人和见证人投票
*/
func (this *WitnessBackup) RollbackCountWitness(txs *[]TxItr) {
	//	depositTxs := make([]TxItr, 0)
	for _, one := range *txs {
		switch one.Class() {
		//过滤见证人押金交易，添加见证人
		case config.Wallet_tx_type_deposit_in:
			//这里决定了交易输出地址才是见证人地址。
			vout := (*one.GetVout())[0]
			// score := vout.Value
			// this.addWitness(&vout.Address, score)
			this.DelWitness(&vout.Address)
		case config.Wallet_tx_type_deposit_out:
			//恢复之前的押金交易，把之前的见证人添加回去
			for _, two := range *one.GetVin() {

				class := ParseTxClass(two.Txid)
				//查找见证人押金输入类型的交易
				if class == config.Wallet_tx_type_deposit_in {
					//找到了这个交易
					txbs, err := db.Find(two.Txid)
					if err != nil {
						fmt.Println("回滚见证人失败-恢复见证人押金输入交易错误", err)
						return
					}
					txItr, err := ParseTxBase(txbs)
					if err != nil {
						fmt.Println("回滚见证人失败-解析并恢复见证人押金输入交易错误", err)
						return
					}
					vout := (*txItr.GetVout())[0]
					this.addWitness(&vout.Address, vout.Value)
					break
				}

			}

		case config.Wallet_tx_type_vote_in:
			//这里决定了交易输出地址才是见证人地址。
			//只有下标为0的输出才是押金。
			vout := (*one.GetVout())[0]
			score := vout.Value
			votein := one.(*Tx_vote_in)

			// this.addVote(&votein.Vote, &vout.Address, score)
			this.DelVote(&votein.Vote, &vout.Address, score)
		case config.Wallet_tx_type_vote_out:
			//恢复之前的押金交易，把之前的投票押金添加回去
			for _, two := range *one.GetVin() {

				class := ParseTxClass(two.Txid)
				//查找见证人押金输入类型的交易
				if class == config.Wallet_tx_type_vote_in {
					//找到了这个交易
					txbs, err := db.Find(two.Txid)
					if err != nil {
						fmt.Println("回滚见证人失败-恢复投票押金输入交易错误", err)
						return
					}
					txItr, err := ParseTxBase(txbs)
					if err != nil {
						fmt.Println("回滚见证人失败-解析并恢复投票押金输入交易错误", err)
						return
					}
					vout := (*txItr.GetVout())[0]
					votein := txItr.(*Tx_vote_in)
					this.addVote(&votein.Vote, &vout.Address, vout.Value)
					break
				}

			}

		}
	}

}

/*
	添加一个见证人到投票列表
*/
func (this *WitnessBackup) addWitness(witnessAddr *utils.Multihash, score uint64) {
	fmt.Println("添加一个见证人", witnessAddr.B58String())
	_, ok := this.witnessesMap.Load(witnessAddr.B58String())
	if ok {
		fmt.Println("见证人已经存在")
		return
	}
	witness := BackupWitness{
		Addr:  witnessAddr, //见证人地址
		Score: score,       //押金
		// Vote:  new(sync.Map), //投票押金
	}
	this.lock.Lock()
	this.witnesses = append(this.witnesses, witness)
	this.lock.Unlock()
	this.witnessesMap.Store(witnessAddr.B58String(), witness)
}

/*
	删除一个见证人
*/
func (this *WitnessBackup) DelWitness(witnessAddr *utils.Multihash) {
	this.lock.Lock()
	//	fmt.Println("++++++删除备用见证人前", len(this.witnesses), witnessAddr.B58String())
	//TODO 有待提高速度
	for i, one := range this.witnesses {
		if !bytes.Equal(*witnessAddr, *one.Addr) {
			continue
		}
		temp := this.witnesses[:i]
		this.witnesses = append(temp, this.witnesses[i+1:]...)
		break
	}
	//	fmt.Println("++++++删除备用见证人后", len(this.witnesses))
	this.lock.Unlock()
	this.witnessesMap.Delete(witnessAddr.B58String())
}

/*
	添加一个投票
*/
func (this *WitnessBackup) addVote(witnessAddr, voteAddr *utils.Multihash, score uint64) {
	// //	fmt.Println("+++++++++++添加一个投票", witnessAddr.B58String(), voteAddr.B58String(), score)
	// v, ok := this.witnessesMap.Load(witnessAddr.B58String())
	// if !ok {
	// 	//		fmt.Println("++++++++添加失败")
	// 	return
	// }
	// bw := v.(*BackupWitness)
	v, ok := this.Vote.Load(voteAddr.B58String())
	if ok {
		vs := v.(*VoteScore)
		vs.Score = vs.Score + score
	} else {
		vs := new(VoteScore)
		vs.Addr = voteAddr
		vs.Score = score
		this.Vote.Store(voteAddr.B58String(), vs)
	}
}

/*
	添加一个见证人到投票列表
*/
func (this *WitnessBackup) DelVote(witnessAddr, voteAddr *utils.Multihash, score uint64) {
	// //	fmt.Println("------------删除一个投票", witnessAddr.B58String(), voteAddr.B58String(), score)
	// v, ok := this.witnessesMap.Load(witnessAddr.B58String())
	// if !ok {
	// 	//		fmt.Println("---------不存在，所以删除失败1111")
	// 	return
	// }
	// bw := v.(*BackupWitness)
	v, ok := this.Vote.Load(voteAddr.B58String())
	if !ok {
		fmt.Println("---------不存在，所以删除失败2222")
		return
	}
	vs := v.(*VoteScore)
	vs.Score = vs.Score - score
	//如果押金为0，则删除这个投票
	if vs.Score == 0 {
		this.Vote.Delete(voteAddr.B58String())
	}
}

/*
	查找备用见证人列表中是否有查询的见证人
*/
func (this *WitnessBackup) haveWitness(witnessAddr *utils.Multihash) (have bool) {
	this.lock.RLock()
	for _, one := range this.witnesses {
		have = bytes.Equal(*witnessAddr, *one.Addr)
		if have {
			break
		}
	}
	this.lock.RUnlock()
	return
}

/*
	参加选举的备用见证人
*/
type BackupWitness struct {
	Addr  *utils.Multihash //见证人地址
	Score uint64           //评分
	// Vote  *sync.Map        //投票押金 key:string=投票人地址;value:*VoteScore=投票人和押金;
}

/*
	根据这一时刻见证人投票排序，生成见证人链
	@return    *Witness    备用见证人链中的一个见证人指针
*/
func (this *WitnessBackup) CreateWitnessGroup() *Witness {
	if len(this.witnesses) < config.Witness_backup_min {
		return nil
	}

	this.lock.Lock()
	sort.Sort(this)
	this.lock.Unlock()

	ws := make([]*Witness, 0)
	for i := 0; i < config.Witness_backup_max && i < len(this.witnesses); i++ {
		newWitness := new(Witness)
		newWitness.Addr = this.witnesses[i].Addr
		newWitness.Score = this.witnesses[i].Score
		newWitness.Votes = make([]*VoteScore, 0)
		this.Vote.Range(func(k, v interface{}) bool {
			vs := v.(*VoteScore)
			newvs := new(VoteScore)
			newvs.Addr = vs.Addr
			newvs.Score = vs.Score
			newWitness.Votes = append(newWitness.Votes, newvs)
			return true
		})
		ws = append(ws, newWitness)
	}
	//	for i, one := range this.witnesses {
	//		if i == 0 {
	//			lastWitness.Addr = one.Addr
	//			startWitness = lastWitness
	//		} else if i >= config.Witness_backup_max {
	//			//只获取排名靠前的n个备用见证人
	//			break
	//		} else {
	//			newWitness := new(Witness)
	//			newWitness.Addr = one.Addr
	//			newWitness.PreWitness = lastWitness
	//			lastWitness.NextWitness = newWitness
	//			lastWitness = newWitness
	//		}

	//	}
	random := this.chain.HashRandom()
	fmt.Println("前n个块hash", hex.EncodeToString(*random))
	start := OrderWitness(ws, random)
	last := start
	for {
		if last == nil {
			break
		}
		fmt.Println("备用见证人排序", last.Addr)
		last = last.NextWitness
	}
	return start
}

/*
	打印备用见证人列表
*/
func (this *WitnessBackup) PrintWitnessBackup() {
	fmt.Println("打印备用见证人")
	this.lock.Lock()
	sort.Sort(this)
	this.lock.Unlock()

	for i, one := range this.witnesses {
		if i >= config.Witness_backup_max {
			//只获取排名靠前的n个备用见证人
			break
		} else {
			fmt.Println("备用见证人", i, one.Addr.B58String())
		}
	}
}

/*
	创建备用见证人列表
*/
func NewWitnessBackup(chain *Chain) *WitnessBackup {
	wb := WitnessBackup{
		chain:        chain, //
		lock:         *new(sync.RWMutex),
		witnesses:    make([]BackupWitness, 0),
		witnessesMap: *new(sync.Map),
		Vote:         *new(sync.Map),
	}
	return &wb
}

/*
	投票押金，作为股权分红
*/
type VoteScore struct {
	Addr  *utils.Multihash //投票人地址
	Score uint64           //押金
}
