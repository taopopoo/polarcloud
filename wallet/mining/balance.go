package mining

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/utils"
	"polarcloud/wallet/db"
	"strconv"
	"sync"
	"sync/atomic"

	"polarcloud/wallet/keystore"
)

/*
	地址余额管理器
*/
type BalanceManager struct {
	syncHeight    uint64              //已经同步到的区块高度
	syncBlockHead chan *BlockHeadVO   //正在同步的余额，准备导入到余额中
	balance       *sync.Map           //保存各个地址的余额，key:string=收款地址;value:*Balance=收益列表;
	depositin     *TxItem             //保存成为见证人押金交易
	votein        *sync.Map           //保存本节点投票的押金额度，key:string=见证人地址;value:*Balance=押金列表;
	witnessBackup *WitnessBackup      //
	txManager     *TransactionManager //
}

func NewBalanceManager(wb *WitnessBackup, tm *TransactionManager) *BalanceManager {
	bm := &BalanceManager{
		syncBlockHead: make(chan *BlockHeadVO, 1), //正在同步的余额，准备导入到余额中
		balance:       new(sync.Map),              //保存各个地址的余额，key:string=收款地址;value:*Balance=收益列表;
		witnessBackup: wb,                         //
		txManager:     tm,                         //
		votein:        new(sync.Map),              //
	}
	go bm.run()
	return bm
}

/*
	保存一个地址的余额列表
	一个地址余额等于多个交易输出相加
*/
type Balance struct {
	Addr *utils.Multihash //
	Txs  *sync.Map        //key:string=交易id;value:*TxItem=交易详细
}

/*
	交易列表
*/
type TxItem struct {
	Addr     *utils.Multihash //收款地址
	Value    uint64           //余额
	Txid     []byte           //交易id
	OutIndex uint64           //交易输出index，从0开始
}

/*
	获取一个地址的余额列表
*/
func (this *BalanceManager) FindBalanceOne(addr *utils.Multihash) *Balance {
	bas, err := this.FindBalance(addr)
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
	获取一个地址的押金列表
*/
func (this *BalanceManager) GetDepositIn() *TxItem {
	return this.depositin
}

/*
	获取一个地址的押金列表
*/
func (this *BalanceManager) GetVoteIn(witnessAddr *utils.Multihash) *Balance {
	v, ok := this.votein.Load(witnessAddr.B58String())
	if !ok {
		return nil
	}
	b := v.(*Balance)
	return b
}

/*
	从最后一个块开始统计多个地址的余额
*/
func (this *BalanceManager) FindBalance(addrs ...*utils.Multihash) ([]*Balance, error) {
	bas := make([]*Balance, 0)
	for _, one := range addrs {
		v, ok := this.balance.Load(one.B58String())
		if ok {
			b := v.(*Balance)
			bas = append(bas, b)
			continue
		}
	}
	return bas, nil
}

/*
	引入最新的块
	将交易计入余额
	使用过的UTXO余额删除
*/
func (this *BalanceManager) CountBalanceForBlock(bhvo *BlockHeadVO) {
	this.countBalance(bhvo)
}

func (this *BalanceManager) run() {
	for bhvo := range this.syncBlockHead {
		this.countBalance(bhvo)
	}
}

/*
	开始统计余额
*/
func (this *BalanceManager) countBalance(bhvo *BlockHeadVO) {
	//		fmt.Println("开始解析余额 111111")
	atomic.StoreUint64(&this.syncHeight, bhvo.BH.Height)
	for _, txItr := range bhvo.Txs {
		//不需要计入余额的类型
		if txItr.Class() != config.Wallet_tx_type_mining &&
			txItr.Class() != config.Wallet_tx_type_deposit_in &&
			txItr.Class() != config.Wallet_tx_type_deposit_out &&
			txItr.Class() != config.Wallet_tx_type_pay &&
			txItr.Class() != config.Wallet_tx_type_vote_in &&
			txItr.Class() != config.Wallet_tx_type_vote_out {
			continue
		}
		txItr.BuildHash()

		//将之前的UTXO标记为已经使用，余额中减去。
		for _, vin := range *txItr.GetVin() {
			addr, err := keystore.BuildAddrByPubkey(vin.Puk)
			if err != nil {
				continue
			}
			//验证地址
			validate := keystore.ValidateByAddress(addr.B58String())
			if !validate.IsVerify || !validate.IsMine {
				continue
			}

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

			switch txItr.Class() {
			case config.Wallet_tx_type_mining:
			case config.Wallet_tx_type_deposit_in:
			case config.Wallet_tx_type_deposit_out:

				if this.depositin != nil {
					if bytes.Equal(*addr, *this.depositin.Addr) {
						this.depositin = nil
					}
				}
			case config.Wallet_tx_type_pay:
			case config.Wallet_tx_type_vote_in:
			case config.Wallet_tx_type_vote_out:

				bs, err := db.Find(vin.Txid)
				if err != nil {
					//TODO 不能找到上一个交易，程序出错退出
					continue
				}
				voteinTxItr, err := ParseTxBase(bs)
				if err != nil {
					//TODO 不能解析上一个交易，程序出错退出
					continue
				}
				votein := voteinTxItr.(*Tx_vote_in)
				b, ok := this.votein.Load(votein.Vote.Address.B58String())
				if ok {
					ba := b.(*Balance)
					ba.Txs.Delete(hex.EncodeToString(*voteinTxItr.GetHash()))
					this.votein.Store(votein.Vote.Address.B58String(), ba)
				}

			}
		}
		//生成新的UTXO收益，保存到列表中
		for voutIndex, vout := range *txItr.GetVout() {
			//找出需要统计余额的地址
			validate := keystore.ValidateByAddress(vout.Address.B58String())
			if !validate.IsVerify || !validate.IsMine {
				continue
			}

			txItem := TxItem{
				Addr:     &vout.Address,
				Value:    vout.Value,        //余额
				Txid:     *txItr.GetHash(),  //交易id
				OutIndex: uint64(voutIndex), //交易输出index，从0开始
			}

			switch txItr.Class() {
			case config.Wallet_tx_type_mining:
			case config.Wallet_tx_type_deposit_in:
				if voutIndex == 0 {
					this.depositin = &txItem
					continue
				}
			case config.Wallet_tx_type_deposit_out:
			case config.Wallet_tx_type_pay:
			case config.Wallet_tx_type_vote_in:
				if voutIndex == 0 {
					voteIn := txItr.(*Tx_vote_in)
					witnessAddr := voteIn.Vote.Address.B58String()
					v, ok := this.votein.Load(witnessAddr)
					var ba *Balance
					if ok {
						ba = v.(*Balance)
					} else {
						ba = new(Balance)
						ba.Txs = new(sync.Map)
					}
					ba.Txs.Store(hex.EncodeToString(*txItr.GetHash())+"_"+strconv.Itoa(voutIndex), &txItem)
					this.votein.Store(witnessAddr, ba)
					continue
				}
			case config.Wallet_tx_type_vote_out:
			}

			v, ok := this.balance.Load(vout.Address.B58String())
			var ba *Balance
			if ok {
				ba = v.(*Balance)
			} else {
				ba = new(Balance)
				ba.Txs = new(sync.Map)
			}

			//				fmt.Println("保存的交易余额", hex.EncodeToString(*txItr.GetHash())+"_"+strconv.Itoa(voutIndex))
			ba.Txs.Store(hex.EncodeToString(*txItr.GetHash())+"_"+strconv.Itoa(voutIndex), &txItem)
			this.balance.Store(vout.Address.B58String(), ba)
		}

	}

	//TODO 纯粹的统计，发布版本去掉
	total := uint64(0)
	key, _ := keystore.GetCoinbase()
	bas, _ := this.FindBalance(key.Hash)
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

/*
	缴纳押金，并广播
*/
func (this *BalanceManager) DepositIn(amount uint64) error {
	key, err := keystore.GetCoinbase()
	if err != nil {
		return err
	}

	//不能重复提交押金
	if this.depositin != nil {
		return errors.New("不能重复缴纳押金")
	}
	if this.txManager.FindDeposit(hex.EncodeToString(key.GetPubKey())) {
		return errors.New("不能重复缴纳押金")
	}

	//	if this.witnessBackup.haveWitness(key.Hash) {
	//	}

	deposiIn := CreateTxDepositIn(key, amount)
	if deposiIn == nil {
		//		fmt.Println("33333333333333 22222")
		return errors.New("交押金失败")
	}
	deposiIn.BuildHash()
	bs, err := deposiIn.Json()
	if err != nil {
		//		fmt.Println("33333333333333 33333")
		return err
	}
	//	fmt.Println("4444444444444444")
	MulticastTx(bs)
	//	fmt.Println("5555555555555555")
	txbase, err := ParseTxBase(bs)
	if err != nil {
		return err
	}
	txbase.BuildHash()
	//	fmt.Println("66666666666666")
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return errors.New("交易不合法，则不发送出去")
	}
	ok := this.txManager.AddTx(txbase)
	fmt.Println("添加押金是否成功", ok)
	//		unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
	//	fmt.Println("7777777777777777")
	return nil
}

/*
	退还押金，并广播
*/
func (this *BalanceManager) DepositOut() error {
	key, err := keystore.GetCoinbase()
	if err != nil {
		return err
	}
	if this.depositin == nil {
		return errors.New("自己没有交押金")
	}

	deposiOut := CreateTxDepositOut(key)
	if deposiOut == nil {
		//		fmt.Println("33333333333333 22222")
		return errors.New("交押金失败")
	}
	deposiOut.BuildHash()
	bs, err := deposiOut.Json()
	if err != nil {
		//		fmt.Println("33333333333333 33333")
		return err
	}
	//	fmt.Println("4444444444444444")
	MulticastTx(bs)
	//	fmt.Println("5555555555555555")
	txbase, err := ParseTxBase(bs)
	if err != nil {
		return err
	}
	txbase.BuildHash()
	//	fmt.Println("66666666666666")
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return errors.New("交易不合法，则不发送出去")
	}
	this.txManager.AddTx(txbase)
	//		unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
	//	fmt.Println("7777777777777777")
	return nil
}

/*
	投票押金，并广播
*/
func (this *BalanceManager) VoteIn(witnessAddr *utils.Multihash, amount uint64) error {
	key, err := keystore.GetCoinbase()
	if err != nil {
		return err
	}

	voetIn := CreateTxVoteIn(key, amount, witnessAddr)
	if voetIn == nil {
		//		fmt.Println("33333333333333 22222")
		return errors.New("交押金失败")
	}
	voetIn.BuildHash()
	bs, err := voetIn.Json()
	if err != nil {
		//		fmt.Println("33333333333333 33333")
		return err
	}
	//	fmt.Println("4444444444444444")
	MulticastTx(bs)
	//	fmt.Println("5555555555555555")
	txbase, err := ParseTxBase(bs)
	if err != nil {
		return err
	}
	txbase.BuildHash()
	//	fmt.Println("66666666666666")
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return errors.New("交易不合法，则不发送出去")
	}
	ok := this.txManager.AddTx(txbase)
	fmt.Println("添加投票押金是否成功", ok)
	//		unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
	//	fmt.Println("7777777777777777")
	return nil
}

/*
	退还投票押金，并广播
*/
func (this *BalanceManager) VoteOut(witnessAddr *utils.Multihash, amount uint64) error {
	key, err := keystore.GetCoinbase()
	if err != nil {
		return err
	}

	balance := this.GetVoteIn(witnessAddr)
	if balance == nil {
		return errors.New("没有对这个见证人投票")
	}

	deposiOut := CreateTxVoteOut(key, amount, witnessAddr)
	if deposiOut == nil {
		//		fmt.Println("33333333333333 22222")
		return errors.New("交押金失败")
	}
	deposiOut.BuildHash()
	bs, err := deposiOut.Json()
	if err != nil {
		//		fmt.Println("33333333333333 33333")
		return err
	}
	//	fmt.Println("4444444444444444")
	MulticastTx(bs)
	//	fmt.Println("5555555555555555")
	txbase, err := ParseTxBase(bs)
	if err != nil {
		return err
	}
	txbase.BuildHash()
	//	fmt.Println("66666666666666")
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return errors.New("交易不合法，则不发送出去")
	}
	this.txManager.AddTx(txbase)
	//		unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
	//	fmt.Println("7777777777777777")
	return nil
}
