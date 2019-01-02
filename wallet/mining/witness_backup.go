package mining

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"sort"
	"sync"
)

//var witnessesLock = new(sync.RWMutex)
//var witnesses = make(WitnessBackup, 0)

type WitnessBackup struct {
	chain        *Chain           //
	lock         *sync.RWMutex    //
	witnesses    []*BackupWitness //
	witnessesMap *sync.Map        //key:string=备用见证人地址;value:*BackupWitness=备用见证人;
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
	添加一个见证人到投票列表
*/
func (this *WitnessBackup) addWitness(witnessAddr *utils.Multihash, score uint64) {
	fmt.Println("添加一个见证人", witnessAddr.B58String())
	_, ok := this.witnessesMap.Load(witnessAddr.B58String())
	if ok {
		fmt.Println("见证人已经存在")
		return
	}
	witness := &BackupWitness{
		Addr:  witnessAddr,   //见证人地址
		Score: score,         //押金
		Vote:  new(sync.Map), //投票押金
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
	//	fmt.Println("+++++++++++添加一个投票", witnessAddr.B58String(), voteAddr.B58String(), score)
	v, ok := this.witnessesMap.Load(witnessAddr.B58String())
	if !ok {
		//		fmt.Println("++++++++添加失败")
		return
	}
	bw := v.(*BackupWitness)
	v, ok = bw.Vote.Load(voteAddr.B58String())
	if ok {
		vs := v.(*VoteScore)
		vs.Score = vs.Score + score
	} else {
		vs := new(VoteScore)
		vs.Addr = voteAddr
		vs.Score = score
		bw.Vote.Store(voteAddr.B58String(), vs)
	}
}

/*
	添加一个见证人到投票列表
*/
func (this *WitnessBackup) DelVote(witnessAddr, voteAddr *utils.Multihash, score uint64) {
	//	fmt.Println("------------删除一个投票", witnessAddr.B58String(), voteAddr.B58String(), score)
	v, ok := this.witnessesMap.Load(witnessAddr.B58String())
	if !ok {
		//		fmt.Println("---------不存在，所以删除失败1111")
		return
	}
	bw := v.(*BackupWitness)
	v, ok = bw.Vote.Load(voteAddr.B58String())
	if !ok {
		fmt.Println("---------不存在，所以删除失败2222")
		return
	}
	vs := v.(*VoteScore)
	vs.Score = vs.Score - score
	//如果押金为0，则删除这个投票
	if vs.Score == 0 {
		bw.Vote.Delete(voteAddr.B58String())
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
	Vote  *sync.Map        //投票押金 key:string=投票人地址;value:*VoteScore=投票人和押金;
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
		this.witnesses[i].Vote.Range(func(k, v interface{}) bool {
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
		lock:         new(sync.RWMutex),
		witnesses:    make([]*BackupWitness, 0),
		witnessesMap: new(sync.Map),
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
