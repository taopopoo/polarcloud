package mining

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/wallet/db"
	//	"polarcloud/wallet/keystore"
)

func RegisteMSG() {
	engine.RegisterMsg(config.MSGID_multicast_vote_recv, MulticastVote_recv)          //接收投票旷工广播
	engine.RegisterMsg(config.MSGID_multicast_blockhead, MulticastBlockHead_recv)     //接收区块头广播
	engine.RegisterMsg(config.MSGID_heightBlock, FindHeightBlock)                     //查询邻居节点区块高度
	engine.RegisterMsg(config.MSGID_heightBlock_recv, FindHeightBlock_recv)           //查询邻居节点区块高度_返回
	engine.RegisterMsg(config.MSGID_getBlockHead, GetBlockHead)                       //查询起始区块头
	engine.RegisterMsg(config.MSGID_getBlockHead_recv, GetBlockHead_recv)             //查询起始区块头_返回
	engine.RegisterMsg(config.MSGID_getTransaction, GetTransaction)                   //查询交易
	engine.RegisterMsg(config.MSGID_getTransaction_recv, GetTransaction_recv)         //查询交易_返回
	engine.RegisterMsg(config.MSGID_multicast_transaction, MulticastTransaction_recv) //接收交易广播
}

/*
	接收备用见证人投票广播
*/
func MulticastVote_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收见证人投票广播")
	//	engine.NLog.Debug(engine.LOG_console, "接收投票旷工广播")
	//	log.Println("接收投票旷工广播", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}
	if !message.CheckSendhash() {
		return
	}

	//	fmt.Println("--接收见证人投票广播", hex.EncodeToString(bt.Deposit))

	//TODO 先验证选票是否合法

	//TODO 再判断是否要为他投票

	//	bt := ParseBallotTicket(message.Body.Content)
	//	AddBallotTicket(bt)

	//继续广播给其他节点
	if nodeStore.NodeSelf.IsSuper {
		//广播给其他超级节点
		//		mh := utils.Multihash(*message.Body.Content)
		ids := nodeStore.GetIdsForFar(message.Head.SenderSuperId)
		for _, one := range ids {
			//			log.Println("发送给", one.B58String())
			if ss, ok := engine.GetSession(one.B58String()); ok {
				ss.Send(msg.MsgID, &msg.Data, &msg.Dataplus, false)
			}
		}

		//广播给代理对象
		pids := nodeStore.GetProxyAll()
		for _, one := range pids {
			if ss, ok := engine.GetSession(one); ok {
				//				ss.Send(MSGID_multicast_online_recv, &msg.Data, false)
				ss.Send(msg.MsgID, &msg.Data, &msg.Dataplus, false)
			}
		}

	}

}

/*
	接收区块广播
	当矿工挖到一个新的区块后，会广播这个区块
*/
func MulticastBlockHead_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收区块头广播")
	//	engine.NLog.Debug(engine.LOG_console, "接收区块头广播")
	//	log.Println("接收区块头广播", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}
	if !message.CheckSendhash() {
		return
	}

	//	fmt.Println("接收区块头广播")
	//	fmt.Println("接收区块头广播", string(*message.Body.Content))

	bhVO, err := ParseBlockHeadVO(message.Body.Content)
	if err != nil {
		fmt.Println("解析区块广播错误", err)
		return
	}
	//	fmt.Println("接收区块广播", bhVO.BH.Height)
	go AddBlockHead(bhVO)
	//	go ImportBlock(bhVO)

	//继续广播给其他节点
	if nodeStore.NodeSelf.IsSuper {
		//广播给其他超级节点
		//		mh := utils.Multihash(*message.Body.Content)
		ids := nodeStore.GetIdsForFar(message.Head.SenderSuperId)
		for _, one := range ids {
			//			log.Println("发送给", one.B58String())
			if ss, ok := engine.GetSession(one.B58String()); ok {
				ss.Send(msg.MsgID, &msg.Data, &msg.Dataplus, false)
			}
		}

		//广播给代理对象
		pids := nodeStore.GetProxyAll()
		for _, one := range pids {
			if ss, ok := engine.GetSession(one); ok {
				//				ss.Send(MSGID_multicast_online_recv, &msg.Data, false)
				ss.Send(msg.MsgID, &msg.Data, &msg.Dataplus, false)
			}
		}
	}

}

/*
	接收邻居节点区块高度
*/
func FindHeightBlock(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收邻居节点区块高度")
	//	engine.NLog.Debug(engine.LOG_console, "接收邻居节点区块高度")
	//	log.Println("接收邻居节点区块高度", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	dataBuf := bytes.NewBuffer([]byte{})
	binary.Write(dataBuf, binary.LittleEndian, GetStartingBlock())
	binary.Write(dataBuf, binary.LittleEndian, GetCurrentBlock())
	bs := dataBuf.Bytes()

	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	mbody := mc.NewMessageBody(&bs, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(config.MSGID_heightBlock_recv)
}

/*
	接收邻居节点区块高度_返回
*/
func FindHeightBlock_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收邻居节点区块高度")
	//	engine.NLog.Debug(engine.LOG_console, "接收邻居节点区块高度")
	//	log.Println("接收邻居节点区块高度", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	mc.ResponseWait(mc.CLASS_findHeightBlock, message.Body.Hash.B58String(), message.Body.Content)
}

/*
	接收邻居节点起始区块头查询
*/
func GetBlockHead(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("++++接收邻居节点区块头查询")
	//	engine.NLog.Debug(engine.LOG_console, "接收邻居节点区块头查询")
	//	log.Println("接收邻居节点区块头查询", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	var bs *[]byte
	bhash, err := db.Find(config.Key_block_start)
	if err == nil {
		bs = bhash
	}

	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	mbody := mc.NewMessageBody(bs, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(config.MSGID_getBlockHead_recv)
}

/*
	接收邻居节点起始区块头查询_返回
*/
func GetBlockHead_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收邻居节点区块头查询_返回")
	//	engine.NLog.Debug(engine.LOG_console, "接收邻居节点区块头查询_返回")
	//	log.Println("接收邻居节点区块头查询_返回", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}
	//	fmt.Println("++++接收邻居节点区块头查询_返回", len(*message.Body.Content))

	mc.ResponseWait(mc.CLASS_getBlockHead, message.Body.Hash.B58String(), message.Body.Content)

}

/*
	接收查询交易
*/
func GetTransaction(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收查询交易")
	//	engine.NLog.Debug(engine.LOG_console, "接收查询交易")
	//	log.Println("接收查询交易", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	var bs *[]byte

	bs, err = db.Find(*message.Body.Content)
	if err != nil {

	}

	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	mbody := mc.NewMessageBody(bs, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(config.MSGID_getTransaction_recv)
}

/*
	接收查询交易_返回
*/
func GetTransaction_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收查询交易_返回")
	//	engine.NLog.Debug(engine.LOG_console, "接收查询交易_返回")
	//	log.Println("接收查询交易_返回", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	mc.ResponseWait(mc.CLASS_getTransaction, message.Body.Hash.B58String(), message.Body.Content)

}

/*
	接收交易广播
*/
func MulticastTransaction_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收区块头广播")
	//	engine.NLog.Debug(engine.LOG_console, "接收区块头广播")
	//	log.Println("接收区块头广播", msg.Session.GetName())

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}
	if !message.CheckSendhash() {
		return
	}

	//	fmt.Println("---------接收交易广播", len(*message.Body.Content))
	//	if len(*message.Body.Content) == 410 {
	//		fmt.Println(string(*message.Body.Content))
	//	}

	//自己处理
	txbase, err := ParseTxBase(message.Body.Content)
	if err != nil {
		fmt.Println("解析广播的交易错误", err)
		return
	}
	txbase.BuildHash()
	//验证交易
	if !txbase.Check() {
		//交易不合法，则不发送出去
		fmt.Println("交易不合法，则不发送出去")
		return
	}

	forks.GetLongChain().transactionManager.AddTx(txbase)

	//	unpackedTransactions.Store(hex.EncodeToString(*txbase.GetHash()), txbase)

	//继续广播给其他节点
	if nodeStore.NodeSelf.IsSuper {
		//广播给其他超级节点
		//		mh := utils.Multihash(*message.Body.Content)
		ids := nodeStore.GetIdsForFar(message.Head.SenderSuperId)
		for _, one := range ids {
			//			log.Println("发送给", one.B58String())
			if ss, ok := engine.GetSession(one.B58String()); ok {
				ss.Send(msg.MsgID, &msg.Data, &msg.Dataplus, false)
			}
		}

		//广播给代理对象
		pids := nodeStore.GetProxyAll()
		for _, one := range pids {
			if ss, ok := engine.GetSession(one); ok {
				//				ss.Send(MSGID_multicast_online_recv, &msg.Data, false)
				ss.Send(msg.MsgID, &msg.Data, &msg.Dataplus, false)
			}
		}
	}

}
