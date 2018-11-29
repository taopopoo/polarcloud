/*
	参与挖矿投票
	参与挖矿的节点发出投票请求，
*/
package mining

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
)

/*
	竟票见证人成功，交押金
	此交易不给手续费
*/
type Tx_deposit_in struct {
	TxBase
}

/*
	构建hash值得到交易id
*/
func (this *Tx_deposit_in) BuildHash() {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return
	}
	delete(m, BlockTx_Gas)
	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
	//	this.Hash = utils.Hash_SHA3_256(bs)

	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_deposit_in)

	this.Hash = append(id, utils.Hash_SHA3_256(bs)...)
}

/*
	对整个交易签名
*/
func (this *Tx_deposit_in) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return nil, err
	}
	delete(m, BlockTx_Gas)
	bs, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return key.Sign(bs, pwd)
}

/*
	格式化成json字符串
*/
func (this *Tx_deposit_in) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	验证是否合法
*/
func (this *Tx_deposit_in) Check() bool {
	return true
}

/*
	验证是否合法
*/
func (this *Tx_deposit_in) GetWitness() *utils.Multihash {
	witness, err := keystore.ParseHashByPubkey(this.Vin[0].Puk)
	if err != nil {
		return nil
	}
	return witness
}

/*
	验证是否合法
*/
//func (this *Tx_deposit_in) GetHash() *[]byte {
//	bs, err := json.Marshal(this)
//	if err != nil {
//		return nil, err
//	}
//	return &bs, err
//}

/*
	见证人出块成功，退还押金
*/
type Tx_deposit_out struct {
	TxBase
	CreateTime int64 `json:"CreateTime"` //创建时间
}

/*
	构建hash值得到交易id
*/
func (this *Tx_deposit_out) BuildHash() {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return
	}

	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
	//	this.Hash = utils.Hash_SHA3_256(bs)

	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_deposit_out)
	this.Hash = append(id, utils.Hash_SHA3_256(bs)...)
}

/*
	对整个交易签名
*/
func (this *Tx_deposit_out) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return nil, err
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return key.Sign(bs, pwd)
}

/*
	格式化成json字符串
*/
func (this *Tx_deposit_out) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	验证是否合法
*/
func (this *Tx_deposit_out) Check() bool {
	return true
}

/*
	这个交易输出被使用之后，需要把UTXO输出标记下
*/
func (this *Tx_deposit_out) SetTxid(index uint64, txid *[]byte) error {
	this.Vout[index].Tx = *txid
	bs, err := this.Json()
	if err != nil {
		return err
	}
	err = db.Save(this.Hash, bs)
	if err != nil {
		return err
	}
	return nil
}

/*
	创建一个见证人押金交易
	@amount    uint64    押金额度
*/
func CreateTxDepositIn(key *keystore.Address, amount uint64) *Tx_deposit_in {
	if amount < config.Mining_deposit {
		fmt.Println("交押金数量最少", config.Mining_deposit)
		return nil
	}
	b := FindBalanceOne(key.Hash)
	if b == nil {
		fmt.Println("++++押金不够")
		return nil
	}
	//查找余额
	vins := make([]Vin, 0)
	total := uint64(0)
	b.Txs.Range(func(k, v interface{}) bool {
		item := v.(*TxItem)

		bs, err := db.Find(item.Txid)
		if err != nil {
			return false
		}
		txItr, err := ParseTxBase(bs)
		if err != nil {
			return false
		}

		var sign *[]byte
		for _, two := range *txItr.GetVout() {
			if bytes.Equal(two.Address, *key.Hash) {
				bs, err := two.CheckJson()
				if err != nil {
					return false
				}
				sign, err = key.Sign(*bs, "123456")
				if err != nil {
					return false
				}
				break
			}
		}
		if sign == nil {
			return false
		}

		vin := Vin{
			Txid: item.Txid,       //UTXO 前一个交易的id
			Vout: item.OutIndex,   //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
			Puk:  key.GetPubKey(), //公钥
			Sign: *sign,           //签名
		}
		vins = append(vins, vin)

		total = total + item.Value
		if total >= amount {
			return false
		}
		return true
	})

	//	for _, one := range b.Txs {

	//	}
	if total < amount {
		//押金不够
		fmt.Println("++++押金不够222")
		return nil
	}

	//构建交易输出
	vouts := make([]Vout, 0)
	vout := Vout{
		Value:   amount,    //输出金额 = 实际金额 * 100000000
		Address: *key.Hash, //钱包地址
	}
	vouts = append(vouts, vout)
	//检查押金是否刚刚好，多了的转账给自己
	if total > amount {
		vout := Vout{
			Value:   total - amount, //输出金额 = 实际金额 * 100000000
			Address: *key.Hash,      //钱包地址
		}
		vouts = append(vouts, vout)
	}

	bs, err := json.Marshal(vouts)
	if err != nil {
		fmt.Println("++++押金不够", err)
		return nil
	}

	sign, err := key.Sign(bs, "123456")
	if err != nil {
		fmt.Println("++++押金不够", err)
		return nil
	}
	for i, _ := range vins {
		vins[i].VoutSign = *sign
	}

	//
	base := TxBase{
		Type:      config.Wallet_tx_type_deposit_in, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total: 1,                                //输入交易数量
		Vin:       vins,                             //交易输入
		Vout:      vouts,                            //
	}
	txin := Tx_deposit_in{
		TxBase: base,
	}
	txin.BuildHash()
	return &txin
}
