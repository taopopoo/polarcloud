package mining

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"sync"
)

const (
	BlockTx_Gas       = "gas"
	BlockTx_Hash      = "hash"
	BlockTx_Vout      = "vout"
	BlockTx_Vout_Tx   = "tx"
	BlockTx_Blockhash = "blockhash"
)

type TxItr interface {
	Class() uint64                                                                  //交易类型
	BuildHash()                                                                     //构建交易hash
	GetHash() *[]byte                                                               //获得交易hash
	Check() bool                                                                    //检查交易是否合法
	Json() (*[]byte, error)                                                         //将交易格式化成json字符串
	Serialize() *[]byte                                                             //将需要签名的字段序列化
	Balance() *sync.Map                                                             //查询交易输出，统计输出地址余额key:utils.Multihash=收款地址;value:TxItem=地址余额;
	GetVin() *[]Vin                                                                 //
	GetVout() *[]Vout                                                               //
	GetGas() uint64                                                                 //
	SetTxid(bs *[]byte, index uint64, txid *[]byte) error                           //这个交易输出被使用之后，需要把UTXO输出标记下
	UnSetTxid(bs *[]byte, index uint64) error                                       //区块回滚，把之前标记为已经使用过的交易的标记去掉
	GetVoutSignSerialize(voutIndex uint64) *[]byte                                  //获取交易输出序列化
	GetSign(key *ecdsa.PrivateKey, txid []byte, voutIndex, vinIndex uint64) *[]byte //获取签名
	//	GetVoutsSign(key *ecdsa.PrivateKey, vinIndex uint64) *[]byte //对输出签名，防止输出被篡改
	SetBlockHash(bs []byte) //设置本交易所属的区块hash
	GetBlockHash() *[]byte  //
	//	SignForPay(prk *ecdsa.PrivateKey) (*[]byte, error)    //
}

/*
	交易
*/
type TxBase struct {
	Hash       []byte `json:"hash"`        //本交易hash
	Type       uint64 `json:"type"`        //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
	Vin_total  uint64 `json:"vin_total"`   //输入交易数量
	Vin        []Vin  `json:"vin"`         //交易输入
	Vout_total uint64 `json:"vout_total"`  //输出交易数量
	Vout       []Vout `json:"vout"`        //交易输出
	Gas        uint64 `json:"gas"`         //交易手续费，不参与区块hash，只用来保存
	LockHeight uint64 `json:"lock_height"` //本交易锁定在低于这个高度的块中，超过这个高度，块将不被打包到区块中。
	BlockHash  []byte `json:"blockhash"`   //本交易属于的区块hash
	//	CreateTime int64  `json:"lock_time"`  //创建时间
}

/*
	设置本交易所属的区块hash
*/
func (this *TxBase) SetBlockHash(bs []byte) {
	this.BlockHash = bs
}

/*
	设置本交易所属的区块hash
*/
func (this *TxBase) GetBlockHash() *[]byte {
	return &this.BlockHash
}

/*
	将需要hash的字段序列化
*/
func (this *TxBase) Serialize() *[]byte {
	buf := bytes.NewBuffer(nil)
	buf.Write(utils.Uint64ToBytes(this.Type))
	buf.Write(utils.Uint64ToBytes(this.Vin_total))
	if this.Vin != nil {
		for _, one := range this.Vin {
			bs := one.Serialize()
			buf.Write(*bs)
		}
	}
	buf.Write(utils.Uint64ToBytes(this.Vout_total))
	if this.Vout != nil {
		for _, one := range this.Vout {
			bs := one.Serialize()
			buf.Write(*bs)
		}
	}
	//	buf.Write(utils.Int64ToBytes(this.CreateTime))
	buf.Write(utils.Uint64ToBytes(this.LockHeight))
	bs := buf.Bytes()
	return &bs
}

func (this *TxBase) GetVin() *[]Vin {
	return &this.Vin
}
func (this *TxBase) GetVout() *[]Vout {
	return &this.Vout
}

func (this *TxBase) GetGas() uint64 {
	return this.Gas
}

func (this *TxBase) GetHash() *[]byte {
	return &this.Hash
}

func (this *TxBase) Class() uint64 {
	return this.Type
}

/*
	获取输出序列化
	[UTXO输入引用的块hash]+[UTXO输入引用的块交易hash]+[UTXO输入引用的输出index(uint64)]+
	[UTXO输入引用的输出序列化]
*/
func (this *TxBase) GetVoutSignSerialize(voutIndex uint64) *[]byte {
	if voutIndex > uint64(len(this.Vout)) {
		return nil
	}
	buf := bytes.NewBuffer(this.BlockHash)
	buf.Write(this.Hash)
	buf.Write(utils.Uint64ToBytes(voutIndex))
	vout := this.Vout[voutIndex]
	bs := vout.Serialize()
	buf.Write(*bs)
	*bs = buf.Bytes()
	return bs
}

/*
	获取本交易用作签名的序列化
	[上一个交易GetVoutSignSerialize()返回]+[本交易类型]+[本交易输入总数]+[本交易输入index]+
	[本交易输出总数]+[vouts序列化]+[锁定区块高度]
	@voutBs    *[]byte    上一个交易GetVoutSignSerialize()返回
*/
func (this *TxBase) GetSignSerialize(voutBs *[]byte, vinIndex uint64) *[]byte {
	if vinIndex > uint64(len(this.Vin)) {
		return nil
	}
	buf := bytes.NewBuffer(*voutBs)
	buf.Write(utils.Uint64ToBytes(this.Type))
	buf.Write(utils.Uint64ToBytes(this.Vin_total))
	buf.Write(utils.Uint64ToBytes(vinIndex))
	buf.Write(utils.Uint64ToBytes(this.Vout_total))
	bs := make([]byte, 0)
	for _, one := range this.Vout {
		bs = append(bs, *one.Serialize()...)
	}
	buf.Write(bs)
	buf.Write(utils.Uint64ToBytes(this.LockHeight))
	bs = buf.Bytes()
	return &bs
}

/*
	获取签名
*/
func (this *TxBase) GetSign(key *ecdsa.PrivateKey, txid []byte, voutIndex, vinIndex uint64) *[]byte {

	bs, err := db.Find(txid)
	if err != nil {
		return nil
	}
	txItr, err := ParseTxBase(bs)
	if err != nil {
		return nil
	}

	bs = txItr.GetVoutSignSerialize(voutIndex)
	//	fmt.Println("签名前1", len(*bs), *bs)
	bs = this.GetSignSerialize(bs, vinIndex)
	//	fmt.Println("签名前2", len(*bs), *bs)
	//	fmt.Println("签名前的字节", len(*bs), *bs)
	bs, err = utils.Sign(key, *bs)
	if err != nil {
		return nil
	}
	return bs
}

///*
//	对输出签名，防止输出被篡改
//	[vin index (uint64)]+[vin sign]+[vouts序列化]
//*/
//func (this *TxBase) GetVoutsSign(key *ecdsa.PrivateKey, vinIndex uint64) *[]byte {
//	if vinIndex > uint64(len(this.Vin)) {
//		return nil
//	}
//	buf := bytes.NewBuffer(utils.Uint64ToBytes(vinIndex))
//	vin := this.Vin[vinIndex]
//	buf.Write(vin.Sign)

//	for _, one := range this.Vout {
//		buf.Write(*one.SignSerialize())
//	}
//	buf.Write(utils.Uint64ToBytes(this.LockHeight))

//	bs, err := utils.Sign(key, buf.Bytes())
//	if err != nil {
//		return nil
//	}
//	return bs
//}

/*
	检查交易是否合法
*/
func (this *TxBase) Check() bool {
	//	fmt.Println("开始验证交易合法性")
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
	//	fmt.Println("这里的手续费是否正确", outTotal, inTotal, this.Gas)

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
		//		fmt.Println("签名前1", len(*bs), *bs)
		bs = this.GetSignSerialize(bs, uint64(i))
		//		fmt.Println("签名前2", len(*bs), *bs)
		ok, err := utils.Verify(one.Puk, *bs, one.Sign)
		if err != nil {
			fmt.Println("验证出错")
			return false
		}
		if !ok {
			fmt.Println("验证不通过")
			return false
		}
	}
	return true
}

func (this *TxBase) Balance() *sync.Map {
	result := new(sync.Map)
	for i, one := range this.Vout {
		if one.Tx != nil {
			continue
		}
		item := TxItem{
			Addr:     &one.Address,
			Value:    one.Value, //余额
			Txid:     this.Hash, //交易id
			OutIndex: uint64(i), //交易输出index，从0开始
		}
		result.Store(one.Address.B58String(), item)
	}
	return result
}

/*
	构建hash值得到交易id
*/
//func (this *TxBase) BuildMap() (map[string]interface{}, error) {
//	vs := make([]map[string]interface{}, 0)
//	for _, one := range this.Vout {
//		m, err := utils.ChangeMap(one)
//		if err != nil {
//			return nil, err
//		}
//		delete(m, BlockTx_Vout_Tx)
//		vs = append(vs, m)
//	}

//	m, err := utils.ChangeMap(this)
//	if err != nil {
//		return nil, err
//	}
//	delete(m, BlockTx_Gas)
//	delete(m, BlockTx_Hash)
//	delete(m, BlockTx_Blockhash)

//	m[BlockTx_Vout] = vs
//	return m, nil
//}

/*
	这个交易输出被使用之后，需要把UTXO输出标记下
	注意：本方法只会保存
*/
func (this *TxBase) SetTxid(bs *[]byte, index uint64, txid *[]byte) error {
	txMap := make(map[string]interface{})
	err := json.Unmarshal(*bs, &txMap)
	if err != nil {
		return err
	}
	v := txMap["vout"]
	if v == nil {
		return errors.New("解析失败")
	}
	vs := v.([]interface{})
	vouts := make([]Vout, 0)
	for _, one := range vs {
		voutBs, err := json.Marshal(one)
		if err != nil {
			return err
		}
		vout := new(Vout)
		err = json.Unmarshal(voutBs, vout)
		if err != nil {
			return err
		}
		vouts = append(vouts, *vout)
	}

	vouts[index].Tx = *txid
	txMap["vout"] = vouts

	txbs, err := json.Marshal(txMap)
	if err != nil {
		return err
	}
	err = db.Save(this.Hash, &txbs)
	return err
}

/*
	区块回滚，把之前标记为已经使用过的交易的标记去掉
*/
func (this *TxBase) UnSetTxid(bs *[]byte, index uint64) error {
	txMap := make(map[string]interface{})
	err := json.Unmarshal(*bs, &txMap)
	if err != nil {
		return err
	}
	v := txMap["vout"]
	if v == nil {
		return errors.New("解析失败")
	}
	vs := v.([]interface{})
	vouts := make([]Vout, 0)
	for _, one := range vs {
		voutBs, err := json.Marshal(one)
		if err != nil {
			return err
		}
		vout := new(Vout)
		err = json.Unmarshal(voutBs, vout)
		if err != nil {
			return err
		}
		vouts = append(vouts, *vout)
	}

	vouts[index].Tx = nil
	txMap["vout"] = vouts

	txbs, err := json.Marshal(txMap)
	if err != nil {
		return err
	}
	err = db.Save(this.Hash, &txbs)
	return err
}

/*
	格式化成json字符串
*/
func (this *TxBase) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	解析交易
*/
func ParseTxBase(bs *[]byte) (TxItr, error) {
	bh := new(TxBase)
	err := json.Unmarshal(*bs, bh)
	if err != nil {
		fmt.Println("1111 这里错误", err, string(*bs))
		return nil, err
	}

	var tx interface{}

	switch bh.Type {
	case config.Wallet_tx_type_mining: //挖矿所得
		tx = new(Tx_reward)
	case config.Wallet_tx_type_deposit_in: //投票参与挖矿输入，余额锁定
		tx = new(Tx_deposit_in)
	case config.Wallet_tx_type_deposit_out: //投票参与挖矿输出，余额解锁
		tx = new(Tx_deposit_out)
	case config.Wallet_tx_type_pay: //普通支付
		tx = new(Tx_Pay)
	case config.Wallet_tx_type_account: //申请账户
		tx = new(Tx_Pay)
	case config.Wallet_tx_type_vote_in: //
		tx = new(Tx_vote_in)
	case config.Wallet_tx_type_vote_out: //
		tx = new(Tx_vote_out)
	}
	err = json.Unmarshal(*bs, tx)
	if err != nil {
		fmt.Println("2222 这里错误", err)
		return nil, err
	}
	return tx.(TxItr), nil
}

/*
	挖矿所得收益输入
*/
//type Coinbase struct{
//	Coinbase :"",
//}

/*
	UTXO输入
*/
type Vin struct {
	Txid []byte `json:"txid"` //UTXO 前一个交易的id
	Vout uint64 `json:"vout"` //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（从零开始）
	Puk  []byte `json:"puk"`  //公钥
	Sign []byte `json:"sign"` //对上一个交易签名，是对整个交易签名（若只对输出签名，当地址和金额一样时，签名输出相同）。
	//	VoutSign []byte `json:"voutsign"` //对本交易的输出签名
}

/*
	将需要签名的字段序列化
*/
func (this *Vin) Serialize() *[]byte {
	buf := bytes.NewBuffer(nil)
	buf.Write(utils.Uint64ToBytes(this.Vout))
	buf.Write(this.Puk)
	buf.Write(this.Sign)
	//	buf.Write(this.VoutSign)
	bs := buf.Bytes()
	return &bs
}

/*
	格式化成json字符串
*/
func (this *Vin) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	UTXO输出
*/
type Vout struct {
	Value   uint64          `json:"value"`   //输出金额 = 实际金额 * 100000000
	Address utils.Multihash `json:"address"` //钱包地址
	Tx      []byte          `json:"tx"`      //本输出被使用后的交易id
}

/*
	将需要签名的字段序列化
*/
func (this *Vout) Serialize() *[]byte {
	buf := bytes.NewBuffer(nil)
	buf.Write(utils.Uint64ToBytes(this.Value))
	buf.Write(this.Address)
	bs := buf.Bytes()
	return &bs
}

/*
	格式化成json字符串
*/
func (this *Vout) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

/*
	格式化成json字符串
*/
//func (this *Vout) CheckJson() (*[]byte, error) {
//	m, err := utils.ChangeMap(this)
//	if err != nil {
//		return nil, err
//	}
//	delete(m, "tx")
//	bs, err := json.Marshal(m)
//	if err != nil {
//		return nil, err
//	}
//	return &bs, err
//}

/*
	全网广播交易
*/
func MulticastTx(bs *[]byte) {
	//		engine.NLog.Debug(engine.LOG_console, "是超级节点发起投票")
	//		log.Println("是超级节点发起投票")
	head := mc.NewMessageHead(nil, nil, false)
	body := mc.NewMessageBody(bs, "", nil, 0)
	message := mc.NewMessage(head, body)
	message.BuildHash()

	//广播给其他节点
	//		ids := nodeStore.GetIdsForFar(message.Content)
	for _, one := range nodeStore.GetAllNodes() {
		//		log.Println("发送给", one.B58String())
		if ss, ok := engine.GetSession(one.B58String()); ok {
			ss.Send(config.MSGID_multicast_transaction, head.JSON(), body.JSON(), false)
		} else {
			engine.NLog.Debug(engine.LOG_console, "发送消息失败")
		}
	}
}

/*
	查询数据库和解析交易
*/
func FindTxBase(txid []byte) (TxItr, error) {
	bs, err := db.Find(txid)
	if err != nil {
		return nil, err
	}
	return ParseTxBase(bs)
}
