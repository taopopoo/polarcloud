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
	"time"
)

/*
	交押金，成为备用见证人
*/
type Tx_vote_in struct {
	TxBase
	Vote Tx_vote_info
}
type Tx_vote_info struct {
	Sign    [][]byte        `json:"sign"`    //每个输入的UTXO签名，对见证人地址+锁定块时间
	Address utils.Multihash `json:"address"` //见证人地址
}

/*
	构建hash值得到交易id
*/
func (this *Tx_vote_in) BuildHash() {
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
	binary.PutUvarint(id, config.Wallet_tx_type_vote_in)

	this.Hash = append(id, utils.Hash_SHA3_256(bs)...)
}

/*
	对整个交易签名
*/
func (this *Tx_vote_in) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
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
func (this *Tx_vote_in) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	验证是否合法
*/
func (this *Tx_vote_in) Check() bool {
	return true
}

/*
	验证是否合法
*/
func (this *Tx_vote_in) GetWitness() *utils.Multihash {
	witness, err := keystore.ParseHashByPubkey(this.Vin[0].Puk)
	if err != nil {
		return nil
	}
	return witness
}

/*
	创建一个见证人投票交易
	@amount    uint64    押金额度
*/
func CreateTxVoteIn(key *keystore.Address, amount uint64, witness *utils.Multihash) *Tx_vote_in {
	if amount < config.Mining_deposit {
		fmt.Println("投票交押金数量最少", config.Mining_deposit)
		return nil
	}
	chain := forks.GetLongChain()
	b := chain.balance.FindBalanceOne(key.Hash)
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
	//下标为0的交易输出是见证人押金，大于0的输出是多余的钱退还。
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

	crateTime := time.Now().Unix()

	voteInfo := Tx_vote_info{
		Sign:    make([][]byte, 0),
		Address: *witness,
	}
	for i, _ := range vins {
		vins[i].VoutSign = *sign

		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.BigEndian, crateTime)
		if err != nil {
			return nil
		}
		_, err = buf.Write(*witness)
		if err != nil {
			return nil
		}
		s, err := key.Sign(buf.Bytes(), "123456")
		if err != nil {
			//			fmt.Println("++++押金不够", err)
			return nil
		}
		voteInfo.Sign = append(voteInfo.Sign, *s)
	}

	//
	base := TxBase{
		Type:       config.Wallet_tx_type_vote_in, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total:  uint64(len(vins)),             //输入交易数量
		Vin:        vins,                          //交易输入
		Vout_total: uint64(len(vouts)),            //输出交易数量
		Vout:       vouts,                         //
		CreateTime: crateTime,                     //创建时间
	}
	txin := Tx_vote_in{
		TxBase: base,
		Vote:   voteInfo,
	}
	txin.BuildHash()
	return &txin
}

/*
	退还押金，赎回押金，见证人因此可能会降低排名
*/
type Tx_vote_out struct {
	TxBase
}

/*
	构建hash值得到交易id
*/
func (this *Tx_vote_out) BuildHash() {
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
	binary.PutUvarint(id, config.Wallet_tx_type_vote_out)

	this.Hash = append(id, utils.Hash_SHA3_256(bs)...)
}

/*
	对整个交易签名
*/
func (this *Tx_vote_out) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
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
func (this *Tx_vote_out) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	验证是否合法
*/
func (this *Tx_vote_out) Check() bool {
	return true
}

/*
	验证是否合法
*/
func (this *Tx_vote_out) GetWitness() *utils.Multihash {
	witness, err := keystore.ParseHashByPubkey(this.Vin[0].Puk)
	if err != nil {
		return nil
	}
	return witness
}

/*
	创建一个投票押金退还交易
	退还按交易为单位，交易的押金全退
*/
func CreateTxVoteOut(key *keystore.Address, amount uint64, witness *utils.Multihash) *Tx_vote_out {

	chain := forks.GetLongChain()
	b := chain.balance.GetVoteIn(witness)
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

	//构建交易输出
	vouts := make([]Vout, 0)
	//下标为0的交易输出是见证人押金，大于0的输出是多余的钱退还。
	vout := Vout{
		Value:   total,     //输出金额 = 实际金额 * 100000000
		Address: *key.Hash, //钱包地址
	}
	vouts = append(vouts, vout)

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

	crateTime := time.Now().Unix()

	for i, _ := range vins {
		vins[i].VoutSign = *sign
	}

	//
	base := TxBase{
		Type:       config.Wallet_tx_type_vote_out, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total:  uint64(len(vins)),              //输入交易数量
		Vin:        vins,                           //交易输入
		Vout_total: uint64(len(vouts)),             //输出交易数量
		Vout:       vouts,                          //
		CreateTime: crateTime,                      //创建时间
	}
	txout := Tx_vote_out{
		TxBase: base,
	}
	txout.BuildHash()
	return &txout
}
