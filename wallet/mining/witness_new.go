package mining

import (
	"bytes"
	"sync"
	"polarcloud/config"
	"polarcloud/core/utils"
)

var witnessesLock = new(sync.RWMutex)
var witnesses = make([]BackupWitness, 0)

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
*/
func CreateWitnessGroup() *Witness {
	if len(witnesses) < config.Mining_group_min {
		return nil
	}
	var startWitness *Witness
	lastWitness := new(Witness)
	//	var lastWitness *Witness
	for i, one := range witnesses {
		if i == 0 {
			lastWitness.Addr = one.Addr
			startWitness = lastWitness
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
