/*
	预备矿工管理
	管理已经投票成功的预备矿工
*/
package mining

import (
	"encoding/json"
	"sync"
	"yunpan/config"
	"yunpan/core/utils"
)

var groupMinersLock = new(sync.RWMutex)
var groupMiners = make([]utils.Multihash, 0)

/*
	备用矿工
*/
type BackupMiners struct {
	Time   int64         //统计时间
	Miners []BackupMiner //预备矿工最多保存两组矿工最大数量(14个)
}

/*
	备用矿工选票计数器
*/
type BackupMiner struct {
	Miner utils.Multihash //矿工地址
	Count uint64          //票数
}

func (this *BackupMiners) JSON() *[]byte {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil
	}
	return &bs
}

/*
	解析预备矿工
*/
func ParseBackupMiners(bs *[]byte) (*BackupMiners, error) {
	bh := new(BackupMiners)
	err := json.Unmarshal(*bs, bh)
	if err != nil {
		return nil, err
	}
	return bh, nil
}

/*
	预备矿工组
*/
//type BackupGroupMiner struct {
//	Height uint64            `json:"height"` //组高度
//	Miners []utils.Multihash `json:"miners"` //预备矿工列表
//}

func AddGroupBackupMiner(miners ...*utils.Multihash) {
	groupMinersLock.Lock()
	//去掉重复的
	for _, one := range miners {
		find := false
		for _, two := range groupMiners {
			if one.B58String() == two.B58String() {
				find = true
				break
			}
		}
		if find {
			continue
		}
		groupMiners = append(groupMiners, *one)
	}
	groupMinersLock.Unlock()
}

/*
	获取预备矿工数量
*/
func TotalBackupMiner() (n uint64) {
	groupMinersLock.RLock()
	n = uint64(len(groupMiners))
	groupMinersLock.RUnlock()
	return
}

//func Get

func RemoveGroupBackupMiner(miners ...utils.Multihash) {
	newMiners := make([]utils.Multihash, 0)
	groupMinersLock.Lock()
	for _, one := range groupMiners {
		find := false
		for _, two := range miners {
			if one.B58String() == two.B58String() {
				find = true
				break
			}
		}
		if find {
		} else {
			newMiners = append(newMiners, one)
		}
	}
	groupMiners = newMiners
	groupMinersLock.Unlock()
}

/*
	加载预备矿工组
*/
func LoadGroupBackupMiner() error {

	group := chain.group
	//向前10组开始查找预备矿工
	for i := 0; i < 10; i++ {
		if group.PreGroup == nil {
			break
		}
		group = group.PreGroup
	}

	for {
		if group.NextGroup == nil && group.Height > 2 {
			break
		}

		//统计这个组中的见证人投票结果，并加入到备用矿工列表中
		//		group.BuildWitness()

		if group.NextGroup == nil {
			break
		} else {
			group = group.NextGroup
		}
	}

	//打印见证人
	//	chain.witnessChain.PrintWitnessList()

	return nil
}

/*
	计算矿工组投票结果
*/
func countBackupGroupVote(bm []*BackupMiners) uint64 {
	witness := chain.witnessChain.GetBackupWitness()
	m := make(map[string]int)
	count := uint64(0)
	for i := 0; i < config.Mining_group_max; i++ {
		//重复的备用见证人，归到下一组
		if _, ok := m[witness.Addr.B58String()]; ok {
			//要判断下下一组人数是否够最少人数
			break
		}
		count++
		m[witness.Addr.B58String()] = 0
		if witness.NextWitness == nil {
			break
		}
		witness = witness.NextWitness
	}
	return count

	//TODO 去掉最高票，去掉最低票，求平均

}
