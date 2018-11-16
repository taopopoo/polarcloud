package mining

import (
	//	"bytes"
	//	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"yunpan/config"
	"yunpan/core/utils"
	//	"yunpan/wallet/db"
	"yunpan/wallet/keystore"
)

var balanceM = new(BalanceManager)

//var balanceSyncHeight uint64 = 0 //余额同步到的区块高度
////这里只保存keystore中地址的余额
//var balance = new(sync.Map)                    //保存各个地址的余额，key:string=收款地址;value:*Balance=收益列表;
//var syncBlockHead = make(chan *BlockHeadVO, 1) //正在同步的余额，准备导入到余额中
////var newTxItem = make([]*TxItem, 0)

func init() {
	//	go runBalance()
	balanceM.syncBlockHead = make(chan *BlockHeadVO, 1)
	balanceM.balance = new(sync.Map)
	go balanceM.run()
}

/*
	地址余额管理器
*/
type BalanceManager struct {
	syncHeight    uint64            //已经同步到的区块高度
	syncBlockHead chan *BlockHeadVO //正在同步的余额，准备导入到余额中
	balance       *sync.Map         //保存各个地址的余额，key:string=收款地址;value:*Balance=收益列表;
}

/*
	保存一个地址的余额列表
	一个地址余额等于多个交易输出相加
*/
type Balance struct {
	Addr *utils.Multihash //
	Txs  *sync.Map         //key:string=交易id;value:*TxItem=交易详细
}

/*
	交易列表
*/
type TxItem struct {
	Addr     *utils.Multihash //收款地址
	Value    uint64            //余额
	Txid     []byte            //交易id
	OutIndex uint64            //交易输出index，从0开始
}

func FindBalanceOne(addr *utils.Multihash) *Balance {
	bas, err := FindBalance(addr)
	if err != nil {
		fmt.Println("这里错误111")
		return nil
	}
	if bas == nil || len(bas) < 1 {
		fmt.Println("这里错误222")
		return nil
	}
	return bas[0]
}

/*
	从最后一个块开始统计多个地址的余额
*/
func FindBalance(addrs ...*utils.Multihash) ([]*Balance, error) {
	bas := make([]*Balance, 0)
	for _, one := range addrs {
		v, ok := balanceM.balance.Load(one.B58String())
		if ok {
			b := v.(*Balance)
			bas = append(bas, b)
			continue
		}
		//		b, err := CountBalance(one)
		//		if err != nil {
		//			return nil, err
		//		}
		//		if b != nil && len(b) >= 1 {
		//			balance.Store(b[0].Addr.B58String(), b[0])
		//			bas = append(bas, b[0])
		//		}
	}
	return bas, nil
}

/*
	启动时统计本钱包多个地址的余额
*/
//func LoadBalance() error {

//	//	keys := keystore.GetAddr()
//	block := chain.GetFirstBlock()
//	for {
//		bh, err := block.Load()
//		if err != nil {
//			return err
//		}

//		bhvo := BlockHeadVO{
//			BH:  bh,               //区块
//			Txs: make([]TxItr, 0), //交易明细
//		}

//		for _, one := range bh.Tx {
//			//获取交易类型
//			class, err := binary.ReadUvarint(bytes.NewBuffer(one[:8]))
//			if err != nil {
//				return err
//			}
//			//			fmt.Println("交易类型为", class)
//			//排除掉不计入余额的交易类型
//			if class != config.Wallet_tx_type_mining &&
//				class != config.Wallet_tx_type_deposit_in &&
//				class != config.Wallet_tx_type_deposit_out &&
//				class != config.Wallet_tx_type_pay {
//				continue
//			}

//			bs, err := db.Find(one)
//			if err != nil {
//				return err
//			}
//			txItr, err := ParseTxBase(bs)
//			if err != nil {
//				return err
//			}
//			bhvo.Txs = append(bhvo.Txs, txItr)
//		}
//		syncBlockHead <- &bhvo
//		if block.NextBlock == nil {
//			break
//		}
//		block = block.NextBlock
//	}
//	return nil
//}

/*
	引入最新的块
	将交易计入余额
	使用过的UTXO余额删除
*/
func CountBalanceForBlock(bhvo *BlockHeadVO) {
	//	height := atomic.LoadUint64(&balanceSyncHeight)
	//	if height+1 == bhvo.BH.Height {
	//		syncBlockHead <- bhvo
	//	}

	//	balanceM.syncBlockHead <- bhvo
	balanceM.count(bhvo)

}

func (this *BalanceManager) run() {
	for bhvo := range this.syncBlockHead {
		this.count(bhvo)
	}
}

/*
	开始统计
*/
func (this *BalanceManager) count(bhvo *BlockHeadVO) {
	//		fmt.Println("开始解析余额 111111")
	atomic.StoreUint64(&this.syncHeight, bhvo.BH.Height)
	for _, txItr := range bhvo.Txs {
		//不需要计入余额的类型
		if txItr.Class() != config.Wallet_tx_type_mining &&
			txItr.Class() != config.Wallet_tx_type_deposit_in &&
			txItr.Class() != config.Wallet_tx_type_deposit_out &&
			txItr.Class() != config.Wallet_tx_type_pay {
			continue
		}
		txItr.BuildHash()
		//			fmt.Println("txid", hex.EncodeToString(*txItr.GetHash()))

		for _, vin := range *txItr.GetVin() {
			if txItr.Class() == config.Wallet_tx_type_deposit_out {
				continue
			}
			addr, err := keystore.BuildAddrByPubkey(vin.Puk)
			if err != nil {
				continue
			}
			//				fmt.Println("vin", vinIndex, hex.EncodeToString(vin.Txid))
			validate := keystore.ValidateByAddress(addr.B58String())
			if !validate.IsVerify || !validate.IsMine {
				continue
			}
			//				if txItr.Class() == config.Wallet_tx_type_deposit_in {
			//					fmt.Println("押金指向的交易id", hex.EncodeToString(vin.Txid))
			//				}

			v, ok := this.balance.Load(addr.B58String())
			var ba *Balance
			if ok {
				ba = v.(*Balance)
			} else {
				ba = new(Balance)
				ba.Txs = new(sync.Map)
			}
			//				fmt.Println("删除掉的交易余额", hex.EncodeToString(vin.Txid)+"_"+strconv.Itoa(int(vin.Vout)))
			ba.Txs.Delete(hex.EncodeToString(vin.Txid) + "_" + strconv.Itoa(int(vin.Vout)))
			this.balance.Store(addr.B58String(), ba)
		}
		for voutIndex, vout := range *txItr.GetVout() {
			//找出需要统计余额的地址
			validate := keystore.ValidateByAddress(vout.Address.B58String())
			if !validate.IsVerify || !validate.IsMine {
				continue
			}

			//				fmt.Println("vout", voutIndex, vout.Address.B58String(), vout.Value)

			v, ok := this.balance.Load(vout.Address.B58String())
			var ba *Balance
			if ok {
				ba = v.(*Balance)
			} else {
				ba = new(Balance)
				ba.Txs = new(sync.Map)
			}
			switch txItr.Class() {
			case config.Wallet_tx_type_mining:
			case config.Wallet_tx_type_deposit_in:
				if voutIndex == 0 {
					continue
				}
			case config.Wallet_tx_type_deposit_out:
			case config.Wallet_tx_type_pay:
			}
			txItem := TxItem{
				Addr:     &vout.Address,
				Value:    vout.Value,        //余额
				Txid:     *txItr.GetHash(),  //交易id
				OutIndex: uint64(voutIndex), //交易输出index，从0开始
			}
			//				fmt.Println("保存的交易余额", hex.EncodeToString(*txItr.GetHash())+"_"+strconv.Itoa(voutIndex))
			ba.Txs.Store(hex.EncodeToString(*txItr.GetHash())+"_"+strconv.Itoa(voutIndex), &txItem)
			this.balance.Store(vout.Address.B58String(), ba)
		}

	}

	total := uint64(0)
	key, _ := keystore.GetCoinbase()
	bas, _ := FindBalance(key.Hash)
	for _, one := range bas {
		one.Txs.Range(func(k, v interface{}) bool {
			ba := v.(*TxItem)
			//				fmt.Println("余额+", hex.EncodeToString(ba.Txid), ba.Value)
			total += ba.Value
			return true
		})
	}
	fmt.Println("引入新的交易后 余额", total, "高度", bhvo.BH.Height)

}
