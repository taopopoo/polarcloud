package mining

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"time"
)

/*
	转账交易
*/
type Tx_Pay struct {
	TxBase
}

/*
	构建hash值得到交易id
*/
func (this *Tx_Pay) BuildHash() {
	m, err := this.TxBase.BuildMap()
	if err != nil {
		return
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_pay)
	this.Hash = append(id, utils.Hash_SHA3_256(bs)...)
}

/*
	检查交易合法性
	1.本交易输入对应UTXO输出
	2.输入额度=输出额度+gas额度
*/
func (this *Tx_Pay) Check() bool {
	return true
}

///*
//	一笔交易
//*/
//type Tx struct {
//	TxBase
//	Vin_total  uint64 `json:"vin_total"`  //输入交易数量
//	Vin        []Vin  `json:"vin"`        //交易输入
//	Vout_total uint64 `json:"vout_total"` //输出交易数量
//	Vout       []Vout `json:"vout"`       //交易输出
//}

///*
//	保存到本地磁盘
//*/
//func (this *Tx) Json() (*[]byte, error) {
//	bs, err := json.Marshal(this)
//	if err != nil {
//		return nil, err
//	}
//	//	err = utils.SaveFile(name, &bs)
//	return &bs, err
//}

/*
	创建一个转款交易
*/
func CreateTxPay(address *utils.Multihash, amount, gas uint64, comment string) (*Tx_Pay, error) {
	chain := forks.GetLongChain()

	//优先从非矿工账户扣款
	coinBaseKey, err := keystore.GetCoinbase()
	if err != nil {
		return nil, err
	}
	//查找余额
	vins := make([]Vin, 0)
	total := uint64(0)
	keys := keystore.GetAddr()
	for _, one := range keys {
		if one.Hash.B58String() == coinBaseKey.Hash.B58String() {
			continue
		}
		bas, err := chain.balance.FindBalance(one.Hash)
		if err != nil {
			return nil, err
		}

		for _, two := range bas {
			two.Txs.Range(func(k, v interface{}) bool {
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
				for _, item := range *txItr.GetVout() {
					if item.Address.B58String() == one.Hash.B58String() {
						bs, err := item.CheckJson()
						if err != nil {
							return false
						}
						sign, err = one.Sign(*bs, "123456")
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
					Puk:  one.GetPubKey(), //公钥
					Sign: *sign,           //签名
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
	//若资金还不够，则花费矿工账户的资金
	if total < amount+gas {
		bas, err := chain.balance.FindBalance(coinBaseKey.Hash)
		if err != nil {
			return nil, err
		}
		for _, two := range bas {

			two.Txs.Range(func(k, v interface{}) bool {
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
				for _, item := range *txItr.GetVout() {
					if item.Address.B58String() == coinBaseKey.Hash.B58String() {
						bs, err := item.CheckJson()
						if err != nil {
							return false
						}
						sign, err = coinBaseKey.Sign(*bs, "123456")
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
					Txid: item.Txid,               //UTXO 前一个交易的id
					Vout: item.OutIndex,           //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
					Puk:  coinBaseKey.GetPubKey(), //公钥
					Sign: *sign,                   //签名
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

	if total < amount+gas {
		//资金不够
		return nil, errors.New("余额不足")
	}

	//构建交易输出
	vouts := make([]Vout, 0)
	vout := Vout{
		Value:   amount,   //输出金额 = 实际金额 * 100000000
		Address: *address, //钱包地址
	}
	vouts = append(vouts, vout)
	//检查押金是否刚刚好，多了的转账给自己
	//TODO 将剩余款项转入新的地址，保证资金安全
	if total > amount+gas {
		vout := Vout{
			Value:   total - amount - gas, //输出金额 = 实际金额 * 100000000
			Address: *coinBaseKey.Hash,    //钱包地址
		}
		vouts = append(vouts, vout)
	}

	bs, err := json.Marshal(vouts)
	if err != nil {
		return nil, err
	}

	for i, one := range vins {
		for _, key := range keys {
			if hex.EncodeToString(key.GetPubKey()) == hex.EncodeToString(one.Puk) {
				sign, err := key.Sign(bs, "123456")
				if err != nil {
					return nil, err
				}
				vins[i].VoutSign = *sign
				break
			}
		}
	}

	//没有输出
	base := TxBase{
		Type:       config.Wallet_tx_type_pay, //交易类型
		Vin_total:  uint64(len(vins)),         //输入交易数量
		Vin:        vins,                      //交易输入
		Vout_total: uint64(len(vouts)),        //输出交易数量
		Vout:       vouts,                     //交易输出
		Gas:        gas,                       //交易手续费
		CreateTime: time.Now().Unix(),         //创建时间
		//	BlockHash  []byte `json:"blockhash"`  //自己被打包到的块hash
	}
	pay := Tx_Pay{
		TxBase: base,
	}
	pay.BuildHash()
	return &pay, nil
}
