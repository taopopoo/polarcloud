package config

import (
	"path/filepath"
)

const (
	Wallet_path          = "wallet"        //钱包目录
	Wallet_path_prkName  = "ec_prk.pem"    //私钥文件名称
	Wallet_path_pukName  = "ec_puk.pem"    //公钥文件名称
	Wallet_seed          = "seed_key.json" //密钥种子文件名称
	Wallet_addr_puk_type = "EC PUBLIC KEY"

	Wallet_MDL_Total        = 10 * 10000 * 10000                                           //货币发行总量10亿
	Wallet_MDL_lock         = 5 * 10000 * 10000                                            //锁仓量
	Wallet_MDL_first_mining = 10                                                           //第一个旷工第一个块分配量
	Wallet_MDL_mining       = Wallet_MDL_Total - Wallet_MDL_lock - Wallet_MDL_first_mining //剩余
)

const (
	Wallet_tx_type_start       = 0   //
	Wallet_tx_type_mining      = 0   //挖矿所得
	Wallet_tx_type_deposit_in  = 1   //投票参与挖矿输入，余额锁定
	Wallet_tx_type_deposit_out = 2   //投票参与挖矿输出，余额解锁
	Wallet_tx_type_pay         = 3   //普通支付
	Wallet_tx_type_account     = 4   //申请账户
	Wallet_tx_type_end         = 100 //
)

///*
//	判断交易类型是否计入余额
//*/
//func CheckTxTypeForBalance(t uint64) bool {
//	if t == Wallet_tx_type_pay || t == Wallet_tx_type_deposit_out {
//		return true
//	}
//	return false
//}

const (
	Mining_block_time  = 10               //出块时间，单位：秒
	Mining_group_min   = 3                //挖矿组最少成员，少于最少成员不出块
	Mining_group_max   = 7                //挖矿组最多成员，最多只有这么多个成员构成一个组
	Mining_group_lock  = 10               //锁定链上10组矿工不参与竟票，除非无新人参与竟票
	Mining_deposit     = 1 * 1e8          //押金最少金额
	Mining_reward      = 1 * 1e8          //出块奖励
	Mining_gas         = 1                //交易矿工费
	Mining_difficulty  = 20               //出块难度，前导零个数
	Witness_backup_min = Mining_group_min //备用见证人数需要满足见证人组最少数量
	Witness_backup_max = 100              //备用见证人排名靠前的最多数量，之后的人依然是选举中的备用见证人。
)

const (
	DB_name = "data" //数据库目录名称
)

var (
	DB_path  = filepath.Join(Wallet_path, DB_name) //数据库目录路径
	Miner    = false                               //本节点是否是矿工
	InitNode = false                               //本节点是否是创世节点
)
