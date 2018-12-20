/*
	矿工费交易
*/
package mining

import (
	"encoding/binary"
	//	"encoding/hex"
	"encoding/json"
	//	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
)

/*
	矿工费交易
	没有输入，只有输出
*/
type Tx_reward struct {
	TxBase
}

/*
	格式化成json字符串
*/
func (this *Tx_reward) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	验证是否合法
*/
func (this *Tx_reward) Check() bool {
	return true
}

/*
	构建hash值得到交易id
*/
func (this *Tx_reward) BuildHash() {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return
	}

	ms, err := utils.ChangeMap(this)
	if err != nil {
		return
	}
	m["CreateTime"] = ms["CreateTime"]

	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_mining)
	this.Hash = append(id, utils.Hash_SHA3_256(bs)...)
	//	fmt.Println("构建的带交易类型的hash", hex.EncodeToString(this.Hash))
	//	fmt.Println("交易信息", string(bs))
}

/*
	对整个交易签名
*/
func (this *Tx_reward) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return nil, err
	}

	ms, err := utils.ChangeMap(this)
	if err != nil {
		return nil, err
	}
	m["CreateTime"] = ms["CreateTime"]

	bs, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return key.Sign(bs, pwd)
}

///*
//	这个交易输出被使用之后，需要把UTXO输出标记下
//*/
//func (this *Tx_reward) SetTxid(index uint64, txid *[]byte) error {
//	this.Vout[index].Tx = *txid
//	bs, err := this.Json()
//	if err != nil {
//		return err
//	}
//	err = db.Save(this.Hash, bs)
//	if err != nil {
//		return err
//	}
//	return nil
//}
