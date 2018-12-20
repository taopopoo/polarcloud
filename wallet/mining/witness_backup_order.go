package mining

import (
	"encoding/hex"
	"math/big"
	"polarcloud/core/utils"
	"sort"
)

func OrderWitness(ws []*Witness, random *[]byte) *Witness {
	witnesses := make(map[string]*Witness)
	addrs := make([]*utils.Multihash, 0)
	for i, one := range ws {
		addrs = append(addrs, one.Addr)
		witnesses[one.Addr.B58String()] = ws[i]
	}

	//	for {
	//		witnesses[startWitness.Addr.B58String()] = startWitness
	//		addrs = append(addrs, startWitness.Addr)
	//		if startWitness.NextWitness == nil {
	//			break
	//		}
	//		startWitness = startWitness.NextWitness
	//	}
	idasc := NewAddrASC(random, addrs)
	idsOrder := idasc.Sort()
	var start *Witness
	var last *Witness
	for _, one := range idsOrder {
		witness := witnesses[one.B58String()]
		if start == nil {
			start = witness
		} else {
			last.NextWitness = witness
			witness.PreWitness = last
		}
		last = witness
		last.NextWitness = nil
	}
	return start
}

/*
	收款地址排序算法
	从小到大排序
*/
type AddrASC struct {
	findNode *big.Int
	nodes    []*big.Int
	addrMap  map[string]*utils.Multihash
}

func (this AddrASC) Len() int {
	return len(this.nodes)
}

func (this AddrASC) Less(i, j int) bool {
	a := new(big.Int).Xor(this.findNode, this.nodes[i])
	b := new(big.Int).Xor(this.findNode, this.nodes[j])
	if a.Cmp(b) > 0 {
		return false
	} else {
		return true
	}
}

func (this AddrASC) Swap(i, j int) {
	this.nodes[i], this.nodes[j] = this.nodes[j], this.nodes[i]
}

func (this AddrASC) Sort() []*utils.Multihash {
	sort.Sort(this)
	result := make([]*utils.Multihash, 0)
	for _, one := range this.nodes {
		mhash := this.addrMap[hex.EncodeToString(one.Bytes())]
		result = append(result, mhash)
	}
	return result
}

/*
	创建一个收款地址排序算法
	不能有重复地址
*/
func NewAddrASC(random *[]byte, addrs []*utils.Multihash) *AddrASC {
	addrMap := make(map[string]*utils.Multihash)
	addrArray := make([]*big.Int, 0)
	for i, one := range addrs {
		oneBig := new(big.Int).SetBytes(*one)
		addrMap[hex.EncodeToString(oneBig.Bytes())] = addrs[i]
		addrArray = append(addrArray, oneBig)
	}
	findNode := new(big.Int).SetBytes(*random)

	return &AddrASC{
		findNode: findNode,
		nodes:    addrArray,
		addrMap:  addrMap,
	}
}
