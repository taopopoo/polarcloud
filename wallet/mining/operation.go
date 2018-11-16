package mining

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
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
	bs, err := FindBalance(addrs...)
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
	bs, err := FindBalance(addr)
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
	获取区块开始高度
*/
func GetStartingBlock() uint64 {
	return atomic.LoadUint64(&chain.StartingBlock)
}

/*
	获取所链接的节点的最高高度
*/
func GetHighestBlock() uint64 {
	return atomic.LoadUint64(&chain.HighestBlock)
}

/*
	获取已经同步到的区块高度
*/
func GetCurrentBlock() uint64 {
	return atomic.LoadUint64(&chain.CurrentBlock)
}

/*
	获取正在同步的区块高度
*/
func GetPulledStates() uint64 {
	return atomic.LoadUint64(&chain.PulledStates)
}

/*
	获取区块组高度
*/
func GetGroupHeight() uint64 {
	return chain.GetLastBlock().Group.Height
}

/*
	给收款地址转账
*/
func SendToAddress(address *utils.Multihash, amount uint64, comment string) (*Tx_Pay, error) {
	//	//构建utxo输出
	//	vouts := make([]Vout, 0)
	//	vout := Vout{
	//		Value:   amount,   //输出金额 = 实际金额 * 100000000
	//		Address: *address, //钱包地址
	//	}
	//	vouts = append(vouts, vout)

	//	key, err := keystore.GetCoinbase()
	//	if err != nil {
	//		return nil, err
	//	}
	txpay, err := CreateTxPay(address, amount, config.Mining_gas, comment)
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
	unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)
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
