package mining

import (
	"math/big"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
)

func OrderWitness(startWitness *Witness, random *[]byte) *Witness {
	findId := new(big.Int).SetBytes(*random)
	ids := make([]*big.Int, 0)
	witnesses := make(map[string]*Witness)
	for {
		witnesses[startWitness.Addr.B58String()] = startWitness
		idOne := new(big.Int).SetBytes(*startWitness.Addr)
		ids = append(ids, idOne)
		if startWitness.NextWitness == nil {
			break
		}
		startWitness = startWitness.NextWitness
	}
	idasc := nodeStore.NewIdASC(findId, ids)
	idsOrder := idasc.Sort()
	var start *Witness
	var last *Witness
	for _, one := range idsOrder {
		multId := utils.Multihash(one.Bytes())
		witness := witnesses[multId.B58String()]
		if start == nil {
			start = witness
		} else {
			last.NextWitness = witness
			witness.PreWitness = last
		}
		last = witness
	}
	return start
}
