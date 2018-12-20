package mining

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"polarcloud/config"
	"polarcloud/wallet/db"
	"polarcloud/wallet/keystore"
	"sync"
)

//保存网络中的交易
//var unpackedTransactions = new(sync.Map) //未打包的交易,key:string=交易hahs id；value=&TxItr

type TransactionManager struct {
	witnessBackup *WitnessBackup //备用见证人
	depositin     *sync.Map      //见证人缴押金,key:string=见证人公钥；value:&TxItr=;见证人不能有重复，因此单独管理
	unpacked      *sync.Map      //未打包的交易,key:string=交易hahs id；value:&TxItr=;
}

/*
	添加一个未打包的交易
*/
func (this *TransactionManager) AddTx(txItr TxItr) bool {
	fmt.Println("添加一个交易", hex.EncodeToString(*txItr.GetHash()))
	//判断双花的交易
	for _, one := range *txItr.GetVin() {
		//先判断数据库
		txBs, err := db.Find(one.Txid)
		if err != nil {
			fmt.Println("111111111111111111", err)
			return false
		}
		txItr, err := ParseTxBase(txBs)
		if err != nil {
			fmt.Println("2222222222222222")
			return false
		}
		if (*txItr.GetVout())[one.Vout].Tx != nil {
			//该笔交易已经被使用
			fmt.Println("3333333333333333333")
			return false
		}

		//判断押金交易是否有双花的交易
		have := false
		this.depositin.Range(func(k, v interface{}) bool {
			txBase := v.(TxItr)
			for _, two := range *txBase.GetVin() {
				if bytes.Equal(one.Txid, two.Txid) && one.Vout == two.Vout {
					have = true
					return false
				}
			}
			return true
		})
		if have {
			fmt.Println("44444444444444")
			return false
		}
		//判断未打包的交易里是否有双花交易
		this.unpacked.Range(func(k, v interface{}) bool {
			txBase := v.(TxItr)
			for _, two := range *txBase.GetVin() {
				if bytes.Equal(one.Txid, two.Txid) && one.Vout == two.Vout {
					have = true
					return false
				}
			}
			return true
		})
		if have {
			fmt.Println("555555555555555555")
			return false
		}
	}

	//见证人押金不能多次提交
	if txItr.Class() == config.Wallet_tx_type_deposit_in {
		addr, err := keystore.ParseHashByPubkey((*txItr.GetVin())[0].Puk)
		if err != nil {
			fmt.Println("666666666666666666")
			return false
		}

		//判断是否已经交了押金
		if this.witnessBackup.haveWitness(addr) {
			fmt.Println("7777777777777777777777")
			return false
		}

		//判断是否有重复的未打包的押金
		pukStr := hex.EncodeToString((*txItr.GetVin())[0].Puk)
		_, ok := this.depositin.Load(pukStr)
		if ok {
			fmt.Println("888888888888888")
			return false
		}
		//		fmt.Println("--------1111添加一个押金交易", hex.EncodeToString(*txItr.GetHash()))
		this.depositin.Store(pukStr, txItr)
		return true
	}

	txItr.BuildHash()
	this.unpacked.Store(hex.EncodeToString(*txItr.GetHash()), txItr)
	//	fmt.Println("--------2222添加一个普通交易")
	return true
}

/*
	添加一个未打包的交易
*/
func (this *TransactionManager) DelTx(txs []TxItr) {
	for _, one := range txs {
		str := hex.EncodeToString(*one.GetHash())
		if one.Class() == config.Wallet_tx_type_deposit_in {
			pukStr := hex.EncodeToString((*one.GetVin())[0].Puk)
			this.depositin.Delete(pukStr)
		}
		this.unpacked.Delete(str)
	}
}

/*
	打包交易
*/
func (this *TransactionManager) Package() ([]TxItr, [][]byte) {
	tx := make([]TxItr, 0)
	txids := make([][]byte, 0)
	this.depositin.Range(func(k, v interface{}) bool {
		txItr := v.(TxItr)
		tx = append(tx, txItr)
		txids = append(txids, *txItr.GetHash())
		fmt.Println("===111打包押金交易", hex.EncodeToString(*txItr.GetHash()))
		return true
	})
	this.unpacked.Range(func(k, v interface{}) bool {
		txItr := v.(TxItr)
		tx = append(tx, txItr)
		txids = append(txids, *txItr.GetHash())
		fmt.Println("===222打包普通交易", hex.EncodeToString(*txItr.GetHash()))
		return true
	})
	return tx, txids
}

/*
	查询见证人是否缴纳押金
*/
func (this *TransactionManager) FindDeposit(puk string) bool {
	_, ok := this.depositin.Load(puk)
	return ok
}

/*
	添加一个未打包的交易
*/
//func (this *TransactionManager) DelTx(txItr TxItr) {}

func NewTransactionManager(wb *WitnessBackup) *TransactionManager {
	tm := TransactionManager{
		witnessBackup: wb,            //
		depositin:     new(sync.Map), //见证人缴押金,key:string=交易hahs id；value=&TxItr
		unpacked:      new(sync.Map), //未打包的交易,key:string=交易hahs id；value:&TxItr=;
	}
	return &tm
}
