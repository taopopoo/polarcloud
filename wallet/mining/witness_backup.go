package mining

import (
	"bytes"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"sort"
	"sync"
)

//var witnessesLock = new(sync.RWMutex)
//var witnesses = make(WitnessBackup, 0)

type WitnessBackup struct {
	lock      *sync.RWMutex
	witnesses []BackupWitness
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
	witness := BackupWitness{
		Addr:  witnessAddr, //见证人地址
		Score: 0,           //评分
	}
	this.lock.Lock()
	this.witnesses = append(this.witnesses, witness)
	this.lock.Unlock()
}

/*
	添加一个见证人到投票列表
*/
func (this *WitnessBackup) DelWitness(witnessAddr *utils.Multihash) {
	this.lock.Lock()
	fmt.Println("++++++删除备用见证人前", len(this.witnesses), witnessAddr.B58String())
	for i, one := range this.witnesses {
		if !bytes.Equal(*witnessAddr, *one.Addr) {
			continue
		}
		temp := this.witnesses[:i]
		this.witnesses = append(temp, this.witnesses[i+1:]...)
		break
	}
	fmt.Println("++++++删除备用见证人后", len(this.witnesses))
	this.lock.Unlock()
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

	var startWitness *Witness
	lastWitness := new(Witness)
	//	var lastWitness *Witness
	for i, one := range this.witnesses {
		if i == 0 {
			lastWitness.Addr = one.Addr
			startWitness = lastWitness
		} else if i >= config.Witness_backup_max {
			//只获取排名靠前的n个备用见证人
			break
		} else {
			newWitness := new(Witness)
			newWitness.Addr = one.Addr
			newWitness.PreWitness = lastWitness
			lastWitness.NextWitness = newWitness
			lastWitness = newWitness
		}

	}
	return startWitness
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
func NewWitnessBackup() *WitnessBackup {
	wb := WitnessBackup{
		lock:      new(sync.RWMutex),
		witnesses: make([]BackupWitness, 0),
	}
	return &wb
}
