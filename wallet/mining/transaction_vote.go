package mining

import (
	"bytes"
	"crypto/ecdsa"
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
type Tx_vote_in struct {
	TxBase
	Vote utils.Multihash `json:"vote"`
}

//type Tx_vote_info struct {
//	Sign    [][]byte        `json:"sign"`    //每个输入的UTXO签名，对见证人地址+锁定块时间
//	Address utils.Multihash `json:"address"` //见证人地址
//}

/*
	将需要签名的字段序列化
*/
//func (this *Tx_vote_info) SignSerialize() *[]byte {
//	buf := bytes.NewBuffer(nil)
//	if this.Sign != nil {
//		for _, one := range this.Sign {
//			buf.Write(one)
//		}
//	}
//	buf.Write(this.Address)
//	bs := buf.Bytes()
//	return &bs
//}

/*
	构建hash值得到交易id
*/
func (this *Tx_vote_in) BuildHash() {
	bs := this.Serialize()

	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_vote_in)

	this.Hash = append(id, utils.Hash_SHA3_256(*bs)...)
}

/*
	对整个交易签名
*/
//func (this *Tx_vote_in) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
//	bs := this.SignSerialize()
//	return key.Sign(*bs, pwd)
//}

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
	格式化成json字符串
*/
func (this *Tx_vote_in) Serialize() *[]byte {
	bs := this.TxBase.Serialize()
	buf := bytes.NewBuffer(*bs)
	//	voteinfo := this.Vote.SignSerialize()
	buf.Write(this.Vote)
	*bs = buf.Bytes()
	return bs
}

/*
	检查交易是否合法
*/
func (this *Tx_vote_in) Check() bool {
	fmt.Println("开始验证交易合法性")
	//检查输入输出是否对等，还有手续费
	inTotal := uint64(0)
	for _, one := range this.Vin {
		txbs, err := db.Find(one.Txid)
		if err != nil {
			return false
		}
		txItr, err := ParseTxBase(txbs)
		if err != nil {
			return false
		}
		vout := (*txItr.GetVout())[one.Vout]
		inTotal = inTotal + vout.Value
	}
	outTotal := uint64(0)
	for _, one := range this.Vout {
		outTotal = outTotal + one.Value
	}
	if outTotal > inTotal {
		return false
	}
	this.Gas = inTotal - outTotal
	fmt.Println("这里的手续费是否正确", outTotal, inTotal, this.Gas)

	//序列化输出
	voutsBs := make([]byte, 0)
	for _, one := range this.Vout {
		voutsBs = append(voutsBs, *one.Serialize()...)
	}

	//验证签名
	for i, one := range this.Vin {
		bs, err := db.Find(one.Txid)
		if err != nil {
			return false
		}
		txItr, err := ParseTxBase(bs)
		if err != nil {
			return false
		}
		//		puk := utils.DecodePubkey(one.Sign)

		//验证公钥是否和地址对应
		pukAddr, err := keystore.BuildAddrByPubkey(one.Puk)
		if err != nil {
			return false
		}
		if !bytes.Equal(*pukAddr, (*txItr.GetVout())[one.Vout].Address) {
			return false
		}
		//判断签名是否正确
		bs = txItr.GetVoutSignSerialize(one.Vout)
		bs = this.GetSignSerialize(bs, uint64(i))
		*bs = append(*bs, this.Vote...)

		//		buf := bytes.NewBuffer(*txItr.GetBlockHash())
		//		buf.Write(*txItr.GetHash())
		//		buf.Write(utils.Uint64ToBytes(one.Vout))
		//		vout := (*txItr.GetVout())[one.Vout]
		//		bs = vout.SignSerialize()
		//		buf.Write(*bs)
		//		fmt.Println("验证时的字节\n", len(buf.Bytes()), buf.Bytes())
		ok, err := utils.Verify(one.Puk, *bs, one.Sign)
		if err != nil {
			return false
		}
		if !ok {
			return false
		}

		//验证输出签名，防止输出被篡改
		//		buf = bytes.NewBuffer(utils.Uint64ToBytes(uint64(i)))
		//		buf.Write(one.Sign)
		//		buf.Write(voutsBs)
		//		buf.Write(utils.Uint64ToBytes(this.LockHeight))
		//		ok, err = utils.Verify(one.Puk, buf.Bytes(), one.VoutSign)
		//		if err != nil {
		//			return false
		//		}
		//		if !ok {
		//			return false
		//		}
	}
	return true
}

/*
	对本交易签名，用于支付
*/
//func (this *Tx_vote_in) SignForPay(prk *ecdsa.PrivateKey) (*[]byte, error) {
//	bs := this.SignSerialize()
//	sign, err := utils.Sign(prk, *bs)
//	return sign, err
//}

/*
	获取签名
*/
func (this *Tx_vote_in) GetSign(key *ecdsa.PrivateKey, txid []byte, voutIndex, vinIndex uint64) *[]byte {
	bs, err := db.Find(txid)
	if err != nil {
		return nil
	}
	txItr, err := ParseTxBase(bs)
	if err != nil {
		return nil
	}

	bs = txItr.GetVoutSignSerialize(voutIndex)
	bs = this.GetSignSerialize(bs, vinIndex)
	*bs = append(*bs, this.Vote...)

	bs, err = utils.Sign(key, *bs)
	if err != nil {
		return nil
	}
	return bs
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
func CreateTxVoteIn(witnessAddr *utils.Multihash, addr string, amount, gas uint64, pwd string) *Tx_vote_in {
	if amount < config.Mining_deposit {
		fmt.Println("投票交押金数量最少", config.Mining_deposit)
		return nil
	}

	chain := forks.GetLongChain()

	//查找余额
	vins := make([]Vin, 0)
	total := uint64(0)
	keys := keystore.GetAddr()
	for _, one := range keys {
		bas, err := chain.balance.FindBalance(one.Hash)
		if err != nil {
			return nil
		}

		for _, two := range bas {
			two.Txs.Range(func(k, v interface{}) bool {
				item := v.(*TxItem)

				//				bs, err := db.Find(item.Txid)
				//				if err != nil {
				//					return false
				//				}
				//				txItr, err := ParseTxBase(bs)
				//				if err != nil {
				//					return false
				//				}

				//				prk, err := one.GetPriKey(pwd)
				//				if err != nil {
				//					return false
				//				}
				//				sign := txItr.GetSign(prk, item.OutIndex)
				//				if sign == nil {
				//					return false
				//				}

				vin := Vin{
					Txid: item.Txid,       //UTXO 前一个交易的id
					Vout: item.OutIndex,   //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
					Puk:  one.GetPubKey(), //公钥
					//					Sign: *sign,           //签名
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
		//押金不够
		fmt.Println("++++余额不足222")
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

	//构建交易输出
	vouts := make([]Vout, 0)
	//下标为0的交易输出是见证人押金，大于0的输出是多余的钱退还。
	vout := Vout{
		Value:   amount,   //输出金额 = 实际金额 * 100000000
		Address: *dstAddr, //钱包地址
	}
	vouts = append(vouts, vout)
	//检查押金是否刚刚好，多了的转账给自己
	//TODO 将剩余款项转入新的地址，保证资金安全
	if total > amount+gas {
		vout := Vout{
			Value:   total - amount - gas,        //输出金额 = 实际金额 * 100000000
			Address: *keystore.GetAddr()[0].Hash, //钱包地址
		}
		vouts = append(vouts, vout)
	}

	//	crateTime := time.Now().Unix()

	//	voteInfo := Tx_vote_info{
	//		Sign:    make([][]byte, 0),
	//		Address: *witnessAddr,
	//	}

	//
	base := TxBase{
		Type:       config.Wallet_tx_type_vote_in,     //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total:  uint64(len(vins)),                 //输入交易数量
		Vin:        vins,                              //交易输入
		Vout_total: uint64(len(vouts)),                //输出交易数量
		Vout:       vouts,                             //
		LockHeight: chain.GetLastBlock().Height + 100, //锁定高度
		//		CreateTime: crateTime,                     //创建时间
	}
	txin := Tx_vote_in{
		TxBase: base,
		Vote:   *witnessAddr,
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

	//给见证人地址签名，防篡改
	//	for i, one := range txin.Vin {
	//		for _, key := range keystore.GetAddr() {
	//			if bytes.Equal(key.GetPubKey(), one.Puk) {
	//				buf := new(bytes.Buffer)
	//				buf.Write(txin.Vin[i].Sign)
	//				buf.Write(txin.Vin[i].VoutSign)
	//				buf.Write(*witnessAddr)
	//				s, err := key.Sign(buf.Bytes(), pwd)
	//				if err != nil {
	//					return nil
	//				}
	//				txin.Vote.Sign = append(txin.Vote.Sign, *s)
	//			}
	//		}
	//	}

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
	bs := this.Serialize()

	id := make([]byte, 8)
	binary.PutUvarint(id, config.Wallet_tx_type_vote_out)

	this.Hash = append(id, utils.Hash_SHA3_256(*bs)...)
}

/*
	对整个交易签名
*/
//func (this *Tx_vote_out) Sign(key *keystore.Address, pwd string) (*[]byte, error) {
//	bs := this.SignSerialize()
//	return key.Sign(*bs, pwd)
//}

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
//func (this *Tx_vote_out) Check() bool {
//	return true
//}

/*
	对本交易签名，用于支付
*/
//func (this *Tx_vote_out) SignForPay(prk *ecdsa.PrivateKey) (*[]byte, error) {
//	bs := this.SignSerialize()
//	sign, err := utils.Sign(prk, *bs)
//	return sign, err
//}

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
func CreateTxVoteOut(witness *utils.Multihash, addr string, amount, gas uint64, pwd string) *Tx_vote_out {
	fmt.Println("==============1")
	chain := forks.GetLongChain()
	b := chain.balance.GetVoteIn(witness)
	if b == nil {
		fmt.Println("++++押金不够")
		return nil
	}
	fmt.Println("==============2")
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
		vout := (*txItr.GetVout())[item.OutIndex]
		prk, err := keystore.GetPriKeyByAddress(vout.Address.B58String(), pwd)
		if err != nil {
			return false
		}
		//		sign := txItr.GetSign(prk, item.OutIndex)
		//		if sign == nil {
		//			return false
		//		}

		pukBs, err := utils.MarshalPubkey(&prk.PublicKey)
		if err != nil {
			return false
		}
		vin := Vin{
			Txid: item.Txid,     //UTXO 前一个交易的id
			Vout: item.OutIndex, //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（第一个为零）
			Puk:  pukBs,         //公钥
			//			Sign: *sign,         //签名
		}
		vins = append(vins, vin)

		total = total + item.Value
		if total >= amount+gas {
			return false
		}
		return true
	})
	fmt.Println("==============3")
	//资金不够
	if total < amount+gas {
		//余额不够给手续费，需要从其他账户余额作为输入给手续费
		for _, one := range keystore.GetAddr() {
			bas, err := chain.balance.FindBalance(one.Hash)
			if err != nil {
				fmt.Println("==============3.1")
				return nil
			}
			fmt.Println("==============3.2")
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

				fmt.Println("==============3.3")
				if total >= amount+gas {

					fmt.Println("==============3.4")
					break
				}
			}
		}

		fmt.Println("==============3.5")
		//		return nil
	}
	fmt.Println("==============4")
	//余额不够给手续费
	if total < (amount + gas) {
		fmt.Println("押金不够")
		//押金不够
		return nil
	}
	fmt.Println("==============5")

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
	fmt.Println("==============6")

	//构建交易输出
	vouts := make([]Vout, 0)
	//下标为0的交易输出是见证人押金，大于0的输出是多余的钱退还。
	vout := Vout{
		Value:   total - gas, //输出金额 = 实际金额 * 100000000
		Address: *dstAddr,    //钱包地址
	}
	vouts = append(vouts, vout)

	//	crateTime := time.Now().Unix()

	//
	base := TxBase{
		Type:       config.Wallet_tx_type_vote_out,    //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vin_total:  uint64(len(vins)),                 //输入交易数量
		Vin:        vins,                              //交易输入
		Vout_total: uint64(len(vouts)),                //输出交易数量
		Vout:       vouts,                             //
		LockHeight: chain.GetLastBlock().Height + 100, //锁定高度
		//		CreateTime: crateTime,                      //创建时间
	}
	txout := Tx_vote_out{
		TxBase: base,
	}
	fmt.Println("==============7")

	//给输出签名，防篡改
	for i, one := range txout.Vin {
		for _, key := range keystore.GetAddr() {
			if bytes.Equal(key.GetPubKey(), one.Puk) {
				prk, err := key.GetPriKey(pwd)
				if err != nil {
					fmt.Println("获取key错误")
					return nil
				}
				sign := txout.GetSign(prk, one.Txid, one.Vout, uint64(i))
				//				sign := txout.GetVoutsSign(prk, uint64(i))
				txout.Vin[i].Sign = *sign
			}
		}
	}
	fmt.Println("==============8")
	txout.BuildHash()
	return &txout
}
