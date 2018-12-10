package mining

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
)

const (
	BlockHead_Hash              = "Hash"
	BlockHead_Height            = "Height"
	BlockHead_MerkleRoot        = "MerkleRoot"
	BlockHead_Previousblockhash = "Previousblockhash"
	BlockHead_Nextblockhash     = "Nextblockhash"
	BlockHead_Sign              = "sign"
	BlockHead_Tx                = "Tx"
	BlockHead_Time              = "Time"
)

//var (

//	//	headBlock     = new(sync.Map)              //保存对应区块高度的区块头hash。key:uint64=区块高度;value:*[]byte=区块高度对应区块头hash;
//	lastBlockHead *BlockHead                   //最高区块
//	preBlockHead  *BlockHead                   //最高区块的上一个区块
//	syncBlock     = make(chan *BlockHeadVO, 1) //连续导入区块
//)

///*
//	从数据库加载区块
//*/
//func LoadBlockInDB() {

//}

/*
	区块头
*/
type BlockHead struct {
	Hash              []byte          `json:"Hash"`              //区块头hash
	Height            uint64          `json:"Height"`            //区块高度(每秒产生一个块高度，uint64容量也足够使用上千亿年)
	GroupHeight       uint64          `json:"GroupHeight"`       //矿工组高度
	Previousblockhash []byte          `json:"Previousblockhash"` //上一个区块头hash
	Nextblockhash     [][]byte        `json:"Nextblockhash"`     //下一个区块头hash,可能有多个分叉，但是要保证排在第一的链是最长链
	NTx               uint64          `json:"NTx"`               //交易数量
	MerkleRoot        []byte          `json:"MerkleRoot"`        //交易默克尔树根hash
	Tx                [][]byte        `json:"Tx"`                //本区块包含的交易id
	Time              int64           `json:"Time"`              //unix时间戳
	Witness           utils.Multihash `json:"Witness"`           //此块见证人地址
	Nonce             uint64          `json:"nonce"`             //随机数，用以调整当前区块头hash
	Sign              []byte          `json:"sign"`              //见证人出块时，见证人对块签名，以证明本块是指定见证人出块。
}

/*
	构建默克尔树根
*/
func (this *BlockHead) BuildMerkleRoot() {
	this.MerkleRoot = BuildMerkleRoot(this.Tx)
}

/*
	检查区块头合法性
*/
func (this *BlockHead) Check() bool {
	old := hex.EncodeToString(this.Hash)
	//	fmt.Println("检查区块头前", old)
	this.BuildHash()
	//	fmt.Println("检查区块头后", hex.EncodeToString(this.Hash))
	if old == hex.EncodeToString(this.Hash) {
		return true
	}
	return false
}

/*
	寻找幸运数字
	@zoroes        uint64       难度，前导零数量
	@stopSignal    chan bool    停止信号 true=已经找到；false=未找到，被终止；
*/
func (this *BlockHead) FindNonce(zoroes uint64, stopSignal chan bool) chan bool {
	fmt.Println("开始工作，寻找幸运数字。请等待...")
	result := make(chan bool, 1)
	stop := false
	for !stop {
		this.Nonce++
		this.BuildHash()
		if utils.CheckNonce(this.Hash, zoroes) {
			result <- true
			return result
		}
		select {
		case <-stopSignal:
			stop = true
		default:
		}
	}
	result <- false
	return result
}

/*
	构建区块头hash
*/
func (this *BlockHead) BuildHash() {
	m, err := utils.ChangeMap(this)
	if err != nil {
		return
	}
	delete(m, BlockHead_Hash)
	delete(m, BlockHead_Nextblockhash)
	delete(m, BlockHead_Sign)
	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
	this.Hash = utils.Hash_SHA3_256(bs)
}

/*
	区块签名
*/
func (this *BlockHead) BuildSign(key *keystore.Address) {
	m, err := utils.ChangeMap(this)
	if err != nil {
		return
	}
	delete(m, BlockHead_Hash)
	delete(m, BlockHead_Nextblockhash)
	delete(m, BlockHead_Sign)
	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
	signBs, err := key.Sign(bs, "123456")
	if err != nil {
		return
	}
	this.Sign = *signBs
}

/*
	保存到本地磁盘
*/
func (this *BlockHead) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, nil
}

/*
	解析区块头
*/
func ParseBlockHead(bs *[]byte) (*BlockHead, error) {
	bh := new(BlockHead)
	err := json.Unmarshal(*bs, bh)
	if err != nil {
		return nil, err
	}
	return bh, nil
}

/*
	构建默克尔树根
*/
func BuildMerkleRoot(tx [][]byte) []byte {
	if len(tx) == 0 {
		return []byte{}
	}

	if len(tx) == 1 {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, 1)
		return utils.Hash_SHA3_256(append(b, append(tx[0], tx[0]...)...))
	}

	txbs := merkleroot(0, tx)
	return txbs[0]
}

/*
	计算默克尔树根
*/
func merkleroot(level uint64, tx [][]byte) [][]byte {
	//	fmt.Println("计算默克尔树", len(tx))
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, level)
	//	fmt.Println("计算默克尔树", b)
	if len(tx) == 1 {
		return [][]byte{append(b, tx[0]...)}
	}

	newtx := make([][]byte, 0)
	for i := 0; i < len(tx)/2; i++ {
		newtx = append(newtx, utils.Hash_SHA3_256(append(b, append(tx[i*2], tx[((i+1)*2)-1]...)...)))
	}
	if len(tx)%2 != 0 {
		newtx = append(newtx, utils.Hash_SHA3_256(append(b, append(tx[0], tx[len(tx)-1]...)...)))
	}
	return merkleroot(level+1, newtx)
}

/*
	对交易输出签名，防止输出被篡改
*/
func BuildSignForVouts(vouts []Vout) string {
	_, err := json.Marshal(vouts)
	if err != nil {
		return ""
	}
	//TODO 签名
	//	utils.Sign()
	return ""
}

type BlockHeadVO struct {
	BH  *BlockHead `json:"bh"`  //区块
	Txs []TxItr    `json:"txs"` //交易明细
	//	BM  *BackupMiners `json:"bm"`  //见证人投票结果
}

/*
	json格式化
*/
func (this *BlockHeadVO) Json() (*[]byte, error) {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}
	return &bs, nil
}

/*
	创建
*/
func CreateBlockHeadVO(bh *BlockHead, txs []TxItr) *BlockHeadVO {
	//	itrs := make([]interface{}, 0)
	//	for _, one := range txs {
	//		itrs = append(itrs, one)
	//	}
	bhvo := BlockHeadVO{
		BH:  bh,  //
		Txs: txs, //交易明细
		//		BM:  bm,  //见证人投票结果
	}
	return &bhvo
}

type BlockHeadVOParse struct {
	BH  *BlockHead    `json:"bh"`  //区块
	Txs []interface{} `json:"txs"` //交易明细
	BM  *BackupMiners `json:"bm"`  //见证人投票结果
}

/*
	解析区块头
*/
func ParseBlockHeadVO(bs *[]byte) (*BlockHeadVO, error) {
	bh := new(BlockHeadVOParse)
	err := json.Unmarshal(*bs, bh)
	if err != nil {
		return nil, err
	}

	txitrs := make([]TxItr, 0)
	for _, one := range bh.Txs {
		bs, err := json.Marshal(one)
		if err != nil {
			return nil, err
		}
		txitr, err := ParseTxBase(&bs)
		if err != nil {
			return nil, err
		}
		txitrs = append(txitrs, txitr)
	}
	bhvo := BlockHeadVO{
		BH:  bh.BH,  //区块
		Txs: txitrs, //交易明细
		//		BM:  bh.BM,  //见证人投票结果
	}
	return &bhvo, nil
}
