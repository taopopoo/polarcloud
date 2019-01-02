package startblock

import (
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"time"

	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"polarcloud/wallet/mining"
)

/*
	构建创世块
	创世块生成两个组，一个组一个块
	第一个块给3个见证者账户分配初始额度，下一个组见证者投票结果
	第二个块
*/
func BuildFirstBlock() {
	fmt.Println("开始创建创世区块")

	witness := make([]*keystore.KeyStore, 0)
	seed1 := keystore.NewKeyStore("wallet", config.Wallet_seed)
	if n, _ := seed1.Load(); n <= 0 {
		seed1.NewLoad("wallet1_seed", "123456")
	}
	witness = append(witness, seed1)

	db.InitDB("wallet/data")

	//----------------生成第一个区块-----------------
	balanceTotal := uint64(config.Mining_reward)
	//构建交易
	txHashes := make([][]byte, 0)
	txs := make([]*mining.Tx_reward, 0)
	//创世块矿工奖励
	vouts := make([]mining.Vout, 0)
	vouts = append(vouts, mining.Vout{
		Value:   balanceTotal,             //输出金额 = 实际金额 * 100000000
		Address: *seed1.GetAddr()[0].Hash, //钱包地址
	})
	base := mining.TxBase{
		Type:       config.Wallet_tx_type_mining, //交易类型，默认0=挖矿所得，没有输入;1=普通转账到地址交易
		Vout_total: 1,                            //输出交易数量
		Vout:       vouts,                        //交易输出
		LockHeight: 1,                            //
		//		CreateTime: time.Now().Unix(),            //创建时间
	}
	reward := mining.Tx_reward{
		TxBase: base,
	}
	txs = append(txs, &reward)
	reward.BuildHash()
	txHashes = append(txHashes, reward.Hash)
	//区块头
	blockHead1 := mining.BlockHead{
		//				Hash              string   //区块头hash
		Height:      1, //区块高度(每秒产生一个块高度，也足够使用上千亿年)
		GroupHeight: 1, //
		//	MerkleRoot        string   //交易默克尔树根hash
		//	Previousblockhash string   //上一个区块头hash
		//	Nextblockhash     string   //下一个区块头hash
		NTx:     uint64(len(txHashes)), //交易数量
		Tx:      txHashes,              //本区块包含的交易id
		Time:    time.Now().Unix(),     //unix时间戳
		Witness: *witness[0].GetAddr()[0].Hash,
		//				BackupMiner: , //备用矿工选举结果hash
	}
	blockHead1.BuildMerkleRoot()

	blockHead1.FindNonce(20, make(chan bool, 1))
	//	db.Save(blockHead1.BackupMiner, backupMiner1.JSON())
	bhbs, _ := blockHead1.Json()
	db.Save(blockHead1.Hash, bhbs)
	fmt.Println("key", "blockHead", hex.EncodeToString(blockHead1.Hash))
	fmt.Println("value", "blockHead", string(*bhbs), "\n")

	db.Save(config.Key_block_start, &blockHead1.Hash)
	//保存到数据库
	for _, one := range txs {

		one.TxBase.BlockHash = blockHead1.Hash
		bs, err := one.Json()
		if err != nil {
			fmt.Println("2 json格式化错误", err)
			return
		}
		db.Save(*one.GetHash(), bs)

		fmt.Println("key", "tx", hex.EncodeToString(*one.GetHash()))
		fmt.Println("value", "tx", string(*bs))
	}
	//	db.SaveBlockHeight(blockHead1.Height, &blockHead1.Hash)
	fmt.Println("创建初始块完成")

}
