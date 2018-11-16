package nodeStore

import (
	"encoding/hex"
	"math/big"
	"sync"
	gconfig "polarcloud/config"
	"polarcloud/core/utils"
)

type Ids struct {
	root           *utils.Multihash
	ids            []*utils.Multihash
	count          int64
	logicNumBuider *LogicNumBuider
	lock           *sync.RWMutex
}

/*
	添加一个id
*/
func (this *Ids) AddId(id *utils.Multihash) (ok bool, removeIDs []*utils.Multihash) {

	if this.count <= 0 {
		for i := 0; i < len(this.ids); i++ {
			this.ids[i] = id
		}
		this.count++
		ok = true
		return
	}

	//非逻辑节点不要添加
	netIDs := this.logicNumBuider.GetNodeNetworkNum()

	delId := make([]*utils.Multihash, 0)
	for i, one := range this.ids {
		kl := NewKademlia()
		kl.Add(new(big.Int).SetBytes(one.Data()))
		kl.Add(new(big.Int).SetBytes(id.Data()))
		nearId := kl.Get(new(big.Int).SetBytes(netIDs[i].Data()))
		//		fmt.Println(hex.EncodeToString(nearId[0].Bytes()))
		if hex.EncodeToString(one.Data()) == hex.EncodeToString(nearId[0].Bytes()) {
			continue
		}
		//		fmt.Println("删除的节点id", i, one, "替换", node.IdInfo.Id)
		delId = append(delId, one)
		//		netNodes[i] = node.IdInfo.Id
		this.ids[i] = id
		ok = true
	}
	//找到删除的节点
	removeIDs = make([]*utils.Multihash, 0)
	for _, one := range delId {
		find := false
		for _, netOne := range this.ids {
			if one.B58String() == netOne.B58String() {
				find = true
				break
			}
		}
		if !find {
			removeIDs = append(removeIDs, one)
		}
	}
	if ok {
		this.count++
	}
	return
}

/*
	删除一个id
*/
func (this *Ids) RemoveId(id *utils.Multihash) {
	have := false

	netIDs := this.logicNumBuider.GetNodeNetworkNum()
	for i, one := range this.ids {
		if one.B58String() != id.B58String() {
			continue
		}
		ids := this.GetIds()

		kl := NewKademlia()

		for _, one := range ids {
			kl.Add(new(big.Int).SetBytes(one.Data()))
		}

		//		nodes.Range(func(k, v interface{}) bool {
		//			value := v.(*Node)
		//			kl.Add(new(big.Int).SetBytes(value.IdInfo.Id.Data()))
		//			return true
		//		})
		nearId := kl.Get(new(big.Int).SetBytes(netIDs[i].Data()))
		//		if len(nearId) == 0 {
		//			continue
		//		}
		mhbs, _ := utils.Encode(nearId[1].Bytes(), gconfig.HashCode)
		idmh := utils.Multihash(mhbs)
		this.ids[i] = &idmh

		have = true

		//		if hex.EncodeToString(one.Data()) == hex.EncodeToString(nearId[0].Bytes()) {
		//			continue
		//		}

		//		bs, _ := utils.Encode(nearId[0].Bytes(), utils.SHA1)
		//		mhbs := utils.Multihash(bs)
		//		netNodes[i] = &mhbs
	}
	if have {
		this.count--
	}
}

/*
	获取所有id
*/
func (this *Ids) GetIds() []*utils.Multihash {
	m := make(map[string]*utils.Multihash)
	if this.count <= 0 {
		return make([]*utils.Multihash, 0)
	}
	for _, one := range this.ids {
		m[one.B58String()] = one
	}
	ids := make([]*utils.Multihash, 0)
	for _, v := range m {
		ids = append(ids, v)
	}
	return ids
}

/*
	通过下标获取id
*/
func (this *Ids) GetIndex(index int) *utils.Multihash {
	return this.ids[index]
}

func NewIds(id *utils.Multihash, level uint) *Ids {
	lb := NewLogicNumBuider(id, level)
	return &Ids{
		root:           id,
		ids:            make([]*utils.Multihash, level),
		logicNumBuider: lb,
		lock:           new(sync.RWMutex),
	}
}
