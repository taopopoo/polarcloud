package mining

import (
	"errors"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
	"sync/atomic"
)

/*
	获取账户余额
*/
func GetBalance() uint64 {
	count := uint64(0)

	keys := keystore.GetAddr()
	addrs := make([]*utils.Multihash, 0)
	for _, one := range keys {
		addrs = append(addrs, one.Hash)
	}
	chain := forks.GetLongChain()
	if chain == nil {
		return 0
	}
	bs, err := chain.balance.FindBalance(addrs...)
	if err != nil {
		fmt.Println("查询收益错误", err)
		return count
	}
	for _, one := range bs {
		one.Txs.Range(func(k, v interface{}) bool {
			item := v.(*TxItem)
			count = count + item.Value
			return true
		})
	}
	return count
}

/*
	通过地址获取余额
*/
func GetBalanceForAddr(addr *utils.Multihash) uint64 {
	count := uint64(0)
	chain := forks.GetLongChain()
	bs, err := chain.balance.FindBalance(addr)
	if err != nil {
		fmt.Println("查询收益错误", err)
		return count
	}
	for _, one := range bs {
		one.Txs.Range(func(k, v interface{}) bool {
			item := v.(*TxItem)
			count = count + item.Value
			return true
		})
	}
	return count
}

/*
	获取区块是否同步完成
*/
func GetSyncFinish() bool {
	//判断是否同步完成
	if GetHighestBlock() <= 0 {
		//区块未同步完成，不能挖矿
		return false
	}
	if GetHighestBlock() > GetCurrentBlock() {
		//区块未同步完成，不能挖矿
		return false
	}
	return true
}

/*
	获取区块开始高度
*/
func GetStartingBlock() uint64 {
	return atomic.LoadUint64(&forks.StartingBlock)
}

/*
	获取网络节点广播的区块最高高度
*/
func GetHighestBlock() uint64 {
	return atomic.LoadUint64(&forks.HighestBlock)
}

/*
	获取已经同步到的区块高度
*/
func GetCurrentBlock() uint64 {
	return atomic.LoadUint64(&forks.CurrentBlock)
}

/*
	获取正在同步的区块高度
*/
func GetPulledStates() uint64 {
	return atomic.LoadUint64(&forks.PulledStates)
}

/*
	获取区块组高度
*/
func GetGroupHeight() uint64 {
	return forks.GetLongChain().GetLastBlock().Group.Height
}

/*
	给收款地址转账
*/
func SendToAddress(address *utils.Multihash, amount, gas uint64, pwd, comment string) (*Tx_Pay, error) {
	txpay, err := CreateTxPay(address, amount, gas, pwd, comment)
	if err != nil {
		fmt.Println("创建交易失败", err)
		return nil, err
	}
	txpay.BuildHash()
	bs, err := txpay.Json()
	if err != nil {
		//		fmt.Println("33333333333333 33333")
		return nil, err
	}
	//	fmt.Println("4444444444444444")
	MulticastTx(bs)

	txbase, err := ParseTxBase(bs)
	if err != nil {
		return nil, err
	}
	txbase.BuildHash()
	//	fmt.Println("66666666666666")
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return nil, errors.New("交易不合法，则不发送出去")
	}
	forks.GetLongChain().transactionManager.AddTx(txbase)

	//	unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
	return txpay, nil
}

/*
	从邻居节点查询起始区块hash
*/
func FindStartBlockForNeighbor() *[]byte {
	for _, key := range nodeStore.GetAllNodes() {
		mhead := mc.NewMessageHead(key, key, true)
		mbody := mc.NewMessageBody(nil, "", nil, 0)
		message := mc.NewMessage(mhead, mbody)
		message.BuildHash()

		session, ok := engine.GetSession(key.B58String())
		if ok {
			err := session.Send(config.MSGID_getBlockHead, mhead.JSON(), mbody.JSON(), false)
			if err != nil {
				continue
			}
			bs := mc.WaitRequest(mc.CLASS_getBlockHead, message.Body.Hash.B58String())
			//		fmt.Println("有消息返回了啊")
			if bs == nil {
				fmt.Println("22222 111 发送共享文件消息失败，可能超时")
				continue
			}
			return bs
		}
	}
	return nil
}

/*
	从邻居节点查询区块头和区块中的交易
*/
func FindBlockForNeighbor(bhash *[]byte) *BlockHeadVO {
	bhvo := new(BlockHeadVO)
	bs := getValueForNeighbor(bhash)
	if bs == nil {
		return nil
	}
	bh, err := ParseBlockHead(bs)
	if err != nil {
		return nil
	}
	bhvo.BH = bh
	bhvo.Txs = make([]TxItr, 0)
	for i, _ := range bh.Tx {
		txbs := getValueForNeighbor(&bh.Tx[i])
		//TODO 验证交易是否合法
		txItr, err := ParseTxBase(txbs)
		if err != nil {
			//TODO 这里一个节点错误，应该从另一个邻居节点拉取交易
			return nil
		}
		bhvo.Txs = append(bhvo.Txs, txItr)
	}
	return bhvo
}

/*
	查询邻居节点数据库，key：value查询
*/
func getValueForNeighbor(bhash *[]byte) *[]byte {
	var bs *[]byte
	for _, key := range nodeStore.GetAllNodes() {
		mhead := mc.NewMessageHead(key, key, true)
		mbody := mc.NewMessageBody(bhash, "", nil, 0)
		message := mc.NewMessage(mhead, mbody)
		message.BuildHash()

		session, ok := engine.GetSession(key.B58String())
		if ok {
			err := session.Send(config.MSGID_getTransaction, mhead.JSON(), mbody.JSON(), false)
			if err != nil {
				continue
			}
			bs = mc.WaitRequest(mc.CLASS_getTransaction, message.Body.Hash.B58String())
			if bs == nil {
				fmt.Println("22222 111 发送共享文件消息失败，可能超时")
				continue
			}
			break
		}
	}
	return bs
}

/*
	缴纳押金，成为备用见证人
*/
func DepositIn(amount, gas uint64, pwd string) error {
	//缴纳备用见证人押金交易
	err := forks.GetLongChain().balance.DepositIn(amount, gas, pwd)
	if err != nil {
		fmt.Println("缴纳押金失败", err)
	}
	fmt.Println("缴纳押金完成")
	return err
}

/*
	退还押金
	@addr    string    可选（默认退回到原地址）。押金赎回到的账户地址
	@amount  uint64    可选（默认退还全部押金）。押金金额
*/
func DepositOut(addr string, amount, gas uint64, pwd string) error {
	//缴纳备用见证人押金交易
	err := forks.GetLongChain().balance.DepositOut(addr, amount, gas, pwd)
	if err != nil {
		fmt.Println("退还押金失败", err)
	}
	fmt.Println("退还押金完成")
	return err
}

/*
	给见证人投票
*/
func VoteIn(witnessAddr *utils.Multihash, addr string, amount, gas uint64, pwd string) error {
	//缴纳备用见证人押金交易
	err := forks.GetLongChain().balance.VoteIn(witnessAddr, addr, amount, gas, pwd)
	if err != nil {
		fmt.Println("缴纳押金失败", err)
	}
	fmt.Println("缴纳押金完成")
	return err
}

/*
	退还见证人投票押金
*/
func VoteOut(witnessAddr *utils.Multihash, addr string, amount, gas uint64, pwd string) error {
	//缴纳备用见证人押金交易
	err := forks.GetLongChain().balance.VoteOut(witnessAddr, addr, amount, gas, pwd)
	if err != nil {
		fmt.Println("退还押金失败", err)
	}
	fmt.Println("退还押金完成")
	return err
}
