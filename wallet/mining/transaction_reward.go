/*
	矿工费交易
*/
package mining

import (
	"encoding/binary"
	"encoding/json"
	"polarcloud/config"
	"polarcloud/core/utils"
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
	检查交易是否合法
*/
func (this *Tx_reward) Check() bool {
	//	fmt.Println("开始验证交易合法性")
	//检查输入输出是否对等，还有手续费

	outTotal := uint64(0)
	for _, one := range this.Vout {
		outTotal = outTotal + one.Value
	}

	return true
}

/*
	构建hash值得到交易id
*/
func (this *Tx_reward) BuildHash() {
	bs := this.Serialize()
	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_mining)
	this.Hash = append(id, utils.Hash_SHA3_256(*bs)...)
}

/*
	对整个交易签名
*/
//func (this *Tx_reward) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
//	bs := this.SignSerialize()
//	return key.Sign(*bs, pwd)
//}

/*
	对本交易签名，用于支付
*/
//func (this *Tx_reward) SignForPay(prk *ecdsa.PrivateKey) (*[]byte, error) {
//	bs := this.SignSerialize()
//	sign, err := utils.Sign(prk, *bs)
//	return sign, err
//}

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
