package mining

import (
	"bytes"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"sort"
	"sync"
)

var witnessesLock = new(sync.RWMutex)
var witnesses = make(WitnessBackup, 0)

type WitnessBackup []BackupWitness

func (this WitnessBackup) Len() int {
	return len(this)
}

func (this WitnessBackup) Less(i, j int) bool {
	return this[i].Score < this[j].Score
}

func (this WitnessBackup) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

/*
	添加一个见证人到投票列表
*/
func addWitness(witnessAddr *utils.Multihash, score uint64) {
	witness := BackupWitness{
		Addr:  witnessAddr, //见证人地址
		Score: 0,           //评分
	}
	witnessesLock.Lock()
	witnesses = append(witnesses, witness)
	witnessesLock.Unlock()
}

/*
	查找备用见证人列表中是否有查询的见证人
*/
func hashWitness(witnessAddr *utils.Multihash) (have bool) {
	witnessesLock.RLock()
	for _, one := range witnesses {
		have = bytes.Equal(*witnessAddr, *one.Addr)
		if have {
			break
		}
	}
	witnessesLock.RUnlock()
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
func CreateWitnessGroup() *Witness {
	if len(witnesses) < config.Witness_backup_min {
		return nil
	}

	witnessesLock.Lock()
	sort.Sort(witnesses)
	witnessesLock.Unlock()

	var startWitness *Witness
	lastWitness := new(Witness)
	//	var lastWitness *Witness
	for i, one := range witnesses {
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
func PrintWitnessBackup() {
	fmt.Println("打印备用见证人")
	witnessesLock.Lock()
	sort.Sort(witnesses)
	witnessesLock.Unlock()

	for i, one := range witnesses {
		if i >= config.Witness_backup_max {
			//只获取排名靠前的n个备用见证人
			break
		} else {
			fmt.Println("备用见证人", i, one.Addr.B58String())
		}
	}
}
