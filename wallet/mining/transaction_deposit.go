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
	交押金，成为备用见证人
*/
type Tx_deposit_in struct {
	TxBase
}

/*
	构建hash值得到交易id
*/
func (this *Tx_deposit_in) BuildHash() {
	bs := this.Serialize()

	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_deposit_in)

	this.Hash = append(id, utils.Hash_SHA3_256(*bs)...)
}

/*
	对整个交易签名
*/
//func (this *Tx_deposit_in) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
//	bs := this.SignSerialize()
//	return key.Sign(*bs, pwd)
//}

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
//func (this *Tx_deposit_in) Check() bool {
//	return true
//}

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
	对本交易签名，用于支付
*/
//func (this *Tx_deposit_in) SignForPay(prk *ecdsa.PrivateKey) (*[]byte, error) {
//	bs := this.SignSerialize()
//	sign, err := utils.Sign(prk, *bs)
//	return sign, err

//}

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
}

/*
	构建hash值得到交易id
*/
func (this *Tx_deposit_out) BuildHash() {
	bs := this.Serialize()

	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_deposit_out)
	this.Hash = append(id, utils.Hash_SHA3_256(*bs)...)
}

/*
	对整个交易签名
*/
//func (this *Tx_deposit_out) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
//	bs := this.SignSerialize()
//	return key.Sign(*bs, pwd)
//}

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
//func (this *Tx_deposit_out) Check() bool {
//	return true
//}

/*
	对本交易签名，用于支付
*/
//func (this *Tx_deposit_out) SignForPay(prk *ecdsa.PrivateKey) (*[]byte, error) {
//	bs := this.SignSerialize()
//	sign, err := utils.Sign(prk, *bs)
//	return sign, err

//}

/*
	创建一个见证人押金交易
	@amount    uint64    押金额度
*/
func CreateTxDepositIn(key *keystore.Address, amount, gas uint64, pwd string) *Tx_deposit_in {
	if amount < config.Mining_deposit {
		fmt.Println("交押金数量最少", config.Mining_deposit)
		return nil
	}
	chain := forks.GetLongChain()
	b := chain.balance.FindBalanceOne(key.Hash)
	if b == nil {
		fmt.Println("++++押金不够")
		return nil
	}
	//获取解密后的私钥
	//	prk, err := key.GetPriKey(pwd)
	//	if err != nil {
	//		return nil
	//	}
	//查找余额
	vins := make([]Vin, 0)
	total := uint64(0)
	b.Txs.Range(func(k, v interface{}) bool {
		item := v.(*TxItem)

		//		bs, err := db.Find(item.Txid)
		//		if err != nil {
		//			return false
		//		}
		//		txItr, err := ParseTxBase(bs)
		//		if err != nil {
		//			return false
		//		}

		//		sign := txItr.GetSign(prk, item.OutIndex)
		//		if sign == nil {
		//			return false
		//		}

		vin := Vin{
			Txid: item.Txid,       //UTXO 前一个交易的id
			Vout: item.OutIndex,   //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
			Puk:  key.GetPubKey(), //公钥
			//			Sign: *sign,           //签名
		}
		vins = append(vins, vin)

		total = total + item.Value
		if total >= (amount + gas) {
			return false
		}
		return true
	})

	//	for _, one := range b.Txs {

	//	}
	if total < (amount + gas) {
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
			Value:   total - (amount + gas), //输出金额 = 实际金额 * 100000000
			Address: *key.Hash,              //钱包地址
		}
		vouts = append(vouts, vout)
	}

	//
	base := TxBase{
		Type:       config.Wallet_tx_type_deposit_in,  //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total:  uint64(len(vouts)),                //输入交易数量
		Vin:        vins,                              //交易输入
		Vout_total: uint64(len(vouts)),                //
		Vout:       vouts,                             //
		LockHeight: chain.GetLastBlock().Height + 100, //锁定高度
		//		CreateTime: time.Now().Unix(),                //创建时间
	}
	txin := Tx_deposit_in{
		TxBase: base,
	}
	//给输出签名，防篡改
	for i, one := range txin.Vin {
		for _, key := range keystore.GetAddr() {
			if bytes.Equal(key.GetPubKey(), one.Puk) {
				prk, err := key.GetPriKey(pwd)
				if err != nil {
					return nil
				}
				sign := txin.GetSign(prk, one.Txid, one.Vout, uint64(i))
				//				sign := txin.GetVoutsSign(prk, uint64(i))
				txin.Vin[i].Sign = *sign
			}
		}
	}
	txin.BuildHash()
	return &txin
}

/*
	创建一个退还押金交易
	额度超过了押金额度，那么会从自己账户余额转账到目标账户（因为考虑到押金太少还不够给手续费的情况）
	@addr      *utils.Multihash    退回到的目标账户地址
	@amount    uint64              押金额度
*/
func CreateTxDepositOut(addr string, amount, gas uint64, pwd string) *Tx_deposit_out {

	chain := forks.GetLongChain()

	item := chain.balance.GetDepositIn()
	if item == nil {
		fmt.Println("没有押金")
		return nil
	}

	vins := make([]Vin, 0)
	total := uint64(item.Value)
	//查看余额够不够
	if total < (amount + gas) {
		//余额不够给手续费，需要从其他账户余额作为输入给手续费
		keys := keystore.GetAddr()
		for _, one := range keys {
			bas, err := chain.balance.FindBalance(one.Hash)
			if err != nil {
				return nil
			}

			for _, two := range bas {
				two.Txs.Range(func(k, v interface{}) bool {
					item := v.(*TxItem)

					//					bs, err := db.Find(item.Txid)
					//					if err != nil {
					//						return false
					//					}
					//					txItr, err := ParseTxBase(bs)
					//					if err != nil {
					//						return false
					//					}
					//					prk, err := one.GetPriKey(pwd)
					//					if err != nil {
					//						return false
					//					}
					//					sign := txItr.GetSign(prk, item.OutIndex)
					//					if sign == nil {
					//						return false
					//					}

					vin := Vin{
						Txid: item.Txid,       //UTXO 前一个交易的id
						Vout: item.OutIndex,   //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
						Puk:  one.GetPubKey(), //公钥
						//						Sign: *sign,           //签名
					}
					vins = append(vins, vin)

					total = total + item.Value
					if total >= amount+gas {
						return false
					}

					return true
				})
				if total >= amount+gas {
					break
				}
			}
		}

	}

	if total < (amount + gas) {
		//押金不够
		return nil
	}

	//解析转账目标账户地址
	var dstAddr *utils.Multihash
	if addr == "" {
		//为空则转给自己
		dstAddr = keystore.GetAddr()[0].Hash
	} else {
		var err error
		*dstAddr, err = utils.FromB58String(addr)
		if err != nil {
			fmt.Println("解析地址失败")
			return nil
		}
	}

	bs, err := db.Find(item.Txid)
	if err != nil {
		return nil
	}
	txItr, err := ParseTxBase(bs)
	if err != nil {
		return nil
	}

	//下标为0的交易输出是见证人押金，大于0的输出是多余的钱退还。
	//地址字符串查私钥
	prvKey, err := keystore.GetPriKeyByAddress((*txItr.GetVout())[0].Address.B58String(), pwd)
	if err != nil {
		return nil
	}
	//给输出签名，用于下一个输入
	//	sign := txItr.GetSign(prvKey, item.OutIndex)

	//公钥格式化
	pub, err := utils.MarshalPubkey(&prvKey.PublicKey)
	if err != nil {
		return nil
	}
	vin := Vin{
		Txid: item.Txid,     //UTXO 前一个交易的id
		Vout: item.OutIndex, //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
		Puk:  pub,           //公钥
		//		Sign: *sign,         //签名
	}
	vins = append(vins, vin) // []Vin{vin} // append(vins, vin)

	//构建交易输出
	vouts := make([]Vout, 0)
	//下标为0的交易输出是见证人押金，大于0的输出是多余的钱退还。
	vout := Vout{
		Value:   amount,   //输出金额 = 实际金额 * 100000000
		Address: *dstAddr, //钱包地址
	}
	vouts = append(vouts, vout)

	//退还剩余的钱
	resultVout := Vout{
		Value:   total - amount - gas,
		Address: *keystore.GetAddr()[0].Hash,
	}
	vouts = append(vouts, resultVout)

	//
	base := TxBase{
		Type:       config.Wallet_tx_type_deposit_out, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total:  uint64(len(vins)),                 //输入交易数量
		Vin:        vins,                              //交易输入
		Vout_total: uint64(len(vouts)),                //
		Vout:       vouts,                             //
		LockHeight: chain.GetLastBlock().Height + 100, //锁定高度
		//		CreateTime: time.Now().Unix(),                 //创建时间
	}
	txin := Tx_deposit_out{
		TxBase: base,
	}
	//给输出签名，防篡改
	for i, one := range txin.Vin {
		for _, key := range keystore.GetAddr() {
			if bytes.Equal(key.GetPubKey(), one.Puk) {
				prk, err := key.GetPriKey(pwd)
				if err != nil {
					return nil
				}
				sign := txin.GetSign(prk, one.Txid, one.Vout, uint64(i))
				//				sign := txin.GetVoutsSign(prk, uint64(i))
				txin.Vin[i].Sign = *sign
			}
		}
	}
	txin.BuildHash()
	return &txin
}
