package mining

import (
	"encoding/json"
	"fmt"
	"sync"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
)

const (
	BlockTx_Hash      = "hash"
	BlockTx_Gas       = "gas"
	BlockTx_Vout      = "vout"
	BlockTx_Vout_Tx   = "tx"
	BlockTx_Blockhash = "blockhash"
)

//保存网络中的交易
//var txs = new(sync.Map)                  //保存未加入区块的交易,key=交易hahs id；value=&TxItr
//var txWitness = new(sync.Map)            //见证人缴押金,key:string=交易hahs id；value=&TxItr
var unpackedTransactions = new(sync.Map) //未打包的交易,key:string=交易hahs id；value=&TxItr

type TxItr interface {
	Class() uint64                            //交易类型
	BuildHash()                               //构建交易hash
	GetHash() *[]byte                         //获得交易hash
	Check() bool                              //检查交易是否合法
	Json() (*[]byte, error)                   //将交易格式化成json字符串
	Balance() *sync.Map                       //查询交易输出，统计输出地址余额key:utils.Multihash=收款地址;value:TxItem=地址余额;
	GetVin() *[]Vin                           //
	GetVout() *[]Vout                         //
	SetTxid(index uint64, txid *[]byte) error //
}

/*
	交易
*/
type TxBase struct {
	Hash       []byte `json:"hash"`       //本交易hash
	Type       uint64 `json:"type"`       //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
	Vin_total  uint64 `json:"vin_total"`  //输入交易数量
	Vin        []Vin  `json:"vin"`        //交易输入
	Vout_total uint64 `json:"vout_total"` //输出交易数量
	Vout       []Vout `json:"vout"`       //交易输出
	Gas        uint64 `json:"gas"`        //交易手续费
	BlockHash  []byte `json:"blockhash"`  //自己被打包到的块hash
}

func (this *TxBase) GetVin() *[]Vin {
	return &this.Vin
}

func (this *TxBase) GetVout() *[]Vout {
	return &this.Vout
}

func (this *TxBase) GetHash() *[]byte {
	return &this.Hash
}

func (this *TxBase) Class() uint64 {
	return this.Type
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
func (this *TxBase) BuildMap() (map[string]interface{}, error) {
	vs := make([]map[string]interface{}, 0)
	for _, one := range this.Vout {
		m, err := utils.ChangeMap(one)
		if err != nil {
			return nil, err
		}
		delete(m, BlockTx_Vout_Tx)
		vs = append(vs, m)
	}

	m, err := utils.ChangeMap(this)
	if err != nil {
		return nil, err
	}
	delete(m, BlockTx_Hash)
	delete(m, BlockTx_Blockhash)

	m[BlockTx_Vout] = vs
	return m, nil
}

/*
	这个交易输出被使用之后，需要把UTXO输出标记下
*/
func (this *TxBase) SetTxid(index uint64, txid *[]byte) error {
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
		fmt.Println("1111 这里错误", err)
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
	Txid     []byte `json:"txid"`     //UTXO 前一个交易的id
	Vout     uint64 `json:"vout"`     //一个输出索引（vout），用于标识来自该交易的哪个UTXO被引用（从零开始）
	Puk      []byte `json:"puk"`      //公钥
	Sign     []byte `json:"sign"`     //对上一个交易签名，是对整个交易签名（若只对输出签名，当地址和金额一样时，签名输出相同）。
	VoutSign []byte `json:"voutsign"` //对本交易的输出签名
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
func (this *Vout) CheckJson() (*[]byte, error) {
	m, err := utils.ChangeMap(this)
	if err != nil {
		return nil, err
	}
	delete(m, "tx")
	bs, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

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
