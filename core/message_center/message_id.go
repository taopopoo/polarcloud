package message_center

import (
	"encoding/json"
	"fmt"
	"time"
	gconfig "yunpan/config"
	//	"yunpan/core/addr_manager"
	"yunpan/core/config"
	"yunpan/core/engine"
	"yunpan/core/nodeStore"
	"yunpan/core/persistence"
	"yunpan/core/utils"
)

const (
//	FindNodeNum    = iota + 101 //查找结点服务id
//	SendMessageNum              //发送消息服务id

//	SaveKeyValueReqNum
//	SaveKeyValueRspNum
)

var MsgChannl = make(chan *MessageVO, 100)

type MessageVO struct {
	Name    string           //消息记录name
	Id      *utils.Multihash //发送消息者id
	Index   int64            //unix时间排序
	Time    string           //接收时间
	Content string           //消息内容
}

func init() {
	//TODO 正式发布将这个模拟函数去掉
	//模拟每10秒钟收到一个消息
	//	go func() {

	//		for {
	//			time.Sleep(time.Second * 10)
	//			now := time.Now()
	//			msgOne := MessageVO{
	//				Name:    "haha",
	//				Id:      "123456789",
	//				Index:   now.Unix(),
	//				Time:    utils.FormatTimeToSecond(now),
	//				Content: "nihaoa",
	//			}
	//			MsgChannl <- &msgOne
	//		}
	//	}()
}

func RegisterCoreMsg() {
	engine.RegisterMsg(gconfig.MSGID_checkNodeOnline, findSuperID)                //检查节点是否在线
	engine.RegisterMsg(gconfig.MSGID_checkNodeOnline_recv, findSuperID_recv)      //检查节点是否在线_返回
	engine.RegisterMsg(gconfig.MSGID_getNearSuperIP, GetNearSuperAddr)            //从邻居节点得到自己的逻辑节点
	engine.RegisterMsg(gconfig.MSGID_getNearSuperIP_recv, GetNearSuperAddr_recv)  //从邻居节点得到自己的逻辑节点_返回
	engine.RegisterMsg(gconfig.MSGID_multicast_online_recv, MulticastOnline_recv) //接收节点上线广播
	engine.RegisterMsg(gconfig.MSGID_ask_close_conn_recv, AskCloseConn_recv)      //询问关闭连接
	engine.RegisterMsg(gconfig.MSGID_TextMsg, TextMsg)                            //接收文本消息

}

/*
	广播节点上线
*/
func MulticastOnline() {
	//间隔一分钟广播一次，广播5次
	for i := 0; i < 5; i++ {
		if nodeStore.NodeSelf.IsSuper {
			//			fmt.Println("开始广播")
			head := NewMessageHead(nil, nil, false)
			content := []byte(*nodeStore.NodeSelf.IdInfo.Id)
			body := NewMessageBody(&content, "", nil, 0)
			message := NewMessage(head, body)
			message.BuildHash()

			//			message := &Message{
			//				//			ReceSuperId:   ,                             //接收者的超级节点id
			//				CreateTime:    utils.TimeFormatToNanosecond(), //消息创建时间unix
			//				SenderSuperId: nodeStore.NodeSelf.IdInfo.Id,   //发送者超级节点id
			//				//				ReplyHash:     "",                                    //回复消息的hash
			//				Accurate: false,                         //是否准确发送给一个节点
			//				Content:  *nodeStore.NodeSelf.IdInfo.Id, //发送的内容
			//			}
			//			data := message.JSON()
			//广播给其他节点
			//		ids := nodeStore.GetIdsForFar(message.Content)
			for _, one := range nodeStore.GetAllNodes() {
				//				fmt.Println()
				if ss, ok := engine.GetSession(one.B58String()); ok {
					ss.Send(gconfig.MSGID_multicast_online_recv, head.JSON(), body.JSON(), false)
				}
			}
		} else {
			//非超级节点不需要广播
			break
		}
		time.Sleep(time.Second * config.Time_Multicast_online)
	}

}

/*
	接收上线的广播
*/
func MulticastOnline_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("接收到有节点上线的广播")

	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
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

	//	message := new(Message)
	//	err := json.Unmarshal(msg.Data, message)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}

	if nodeStore.NodeSelf.IsSuper {
		mh := utils.Multihash(*message.Body.Content)
		//继续广播给其他节点
		ids := nodeStore.GetIdsForFar(&mh)
		for _, one := range ids {
			if ss, ok := engine.GetSession(one.B58String()); ok {
				//				ss.Send(MSGID_multicast_online_recv, &msg.Data, false)
				ss.Send(gconfig.MSGID_multicast_online_recv, &msg.Data, &msg.Dataplus, false)
			}
		}
		//广播给代理对象
		pids := nodeStore.GetProxyAll()
		for _, one := range pids {
			if ss, ok := engine.GetSession(one); ok {
				//				ss.Send(MSGID_multicast_online_recv, &msg.Data, false)
				ss.Send(gconfig.MSGID_multicast_online_recv, &msg.Data, &msg.Dataplus, false)
			}
		}

	}

	//自己处理新上线的节点
	//	fmt.Println("有节点上线", hex.EncodeToString(message.Content))
	//检查是否是自己的逻辑节点
	newId := utils.Multihash(*message.Body.Content)
	ok := nodeStore.CheckNeedNode(&newId)
	if ok {
		//		var head *MessageHead
		mhead := NewMessageHead(&newId, &newId, false)
		mbody := NewMessageBody(nil, "", nil, 0)
		message = NewMessage(mhead, mbody)

		nearId := nodeStore.FindNearInSuper(&newId, nil, false)
		if nearId == nil {
			return
		}
		session, ok := engine.GetSession(nearId.B58String())
		if !ok {
			return
		}
		session.Send(gconfig.MSGID_checkNodeOnline, message.Head.JSON(), nil, false)

	}
}

/*
	查询一个id最近的超级节点id
*/
func findSuperID(c engine.Controller, msg engine.Packet) {
	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}

	//自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	//	id := nodeStore.FindNearInSuper(message.Head.RecvId, message.Head.Sender, true)

	//给发送者回复
	//	node := nodeStore.FindNode(id)
	//	if node == nil {
	//		//也有可能查找的是自己
	//		if id.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
	//			node = nodeStore.NodeSelf
	//		}
	//	}
	data, _ := json.Marshal(nodeStore.NodeSelf)
	head := NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	body := NewMessageBody(&data, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = NewMessage(head, body)
	message.Reply(gconfig.MSGID_checkNodeOnline_recv)

	//	message = &Message{
	//		RecvId:        message.Sender,                 //接收者id
	//		RecvSuperId:   message.SenderSuperId,          //接收者的超级节点id
	//		CreateTime:    message.CreateTime,             //消息创建时间unix
	//		Sender:        nodeStore.NodeSelf.IdInfo.Id,   //发送者id
	//		SenderSuperId: nodeStore.NodeSelf.IdInfo.Id,   //发送者超级节点id
	//		ReplyTime:     utils.TimeFormatToNanosecond(), //消息回复时间unix
	//		Hash:          message.Hash,                   //消息的hash值
	//		Accurate:      true,                           //是否准确发送给一个节点
	//		Content:       data,                           //发送的内容
	//		Rand:          utils.GetAccNumber(),
	//	}
	//	message.BuildReplyHash()

	//	fmt.Println("发送给回复 id", hex.EncodeToString(message.ReceSuperId), "hash", message.Hash)
	//	IsSendToOtherSuper(message, MSGID_findSuperID_recv, nil)

	//---------------------------------------------------------

	//	//	fmt.Println("接收到查询最近超级节点请求")

	//	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}

	//	//自己处理
	//	if err := message.ParserContent(); err != nil {
	//		fmt.Println(err)
	//		return
	//	}

	//	if message.Head.Sender.B58String() == "5dtPgE32MoURWep7QEViBkZh5iVLDZ" {
	//		fmt.Println("要查询的节点id", message.Head.Sender.B58String())
	//	}

	//	//	ids := make([]*utils.Multihash, 0)
	//	nodes := make([]nodeStore.Node, 0)
	//	ns := nodeStore.GetAllNodes()
	//	idsm := nodeStore.NewIds(message.Head.Sender, nodeStore.NodeIdLevel)
	//	for _, one := range ns {

	//		if message.Head.Sender.B58String() == one.B58String() {
	//			continue
	//		}
	//		if message.Head.Sender.B58String() == "5dtPgE32MoURWep7QEViBkZh5iVLDZ" {
	//			fmt.Println("添加了一个地址", one.B58String())
	//		}
	//		//		ok, remove := idsm.AddId(one)
	//		idsm.AddId(one)
	//	}

	//	ids := idsm.GetIds()
	//	for _, one := range ids {
	//		if message.Head.Sender.B58String() == "5dtPgE32MoURWep7QEViBkZh5iVLDZ" {
	//			fmt.Println("查询到的节点", one.B58String())
	//		}
	//		node := nodeStore.FindNode(one)
	//		if node != nil {
	//			nodes = append(nodes, *node)
	//		} else {
	//			fmt.Println("这个节点为空")
	//		}
	//	}
	//	data, _ := json.Marshal(nodes)

	//	//	id := nodeStore.FindNearInSuper(message.Head.RecvId, message.Head.Sender, true)

	//	//	//给发送者回复
	//	//	node := nodeStore.FindNode(id)
	//	//	if node == nil {
	//	//		//也有可能查找的是自己
	//	//		if id.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
	//	//			node = nodeStore.NodeSelf
	//	//		}
	//	//	}
	//	//	data, _ := json.Marshal(node)

	//	head := NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	//	body := NewMessageBody(&data, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	//	message = NewMessage(head, body)
	//	message.Reply(gconfig.MSGID_findSuperID_recv)

}

func findSuperID_recv(c engine.Controller, msg engine.Packet) {
	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}

	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}

	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	//	nodes := make([]nodeStore.Node, 0)

	newNode := new(nodeStore.Node)
	if err = json.Unmarshal(*message.Body.Content, &newNode); err != nil {
		//		fmt.Println("解析失败", err)
		return
	}

	node := nodeStore.FindNode(newNode.IdInfo.Id)
	node.FlashOnlineTime()

	//	//	fmt.Println("查找结果", newNode.IdInfo.Id.B58String())

	//	for _, newNode := range nodes {
	//		//		if newNode.IdInfo.Id.B58String() == "5duDDfkY1tChLKGxbAtPdPysp9ghYn" || newNode.IdInfo.Id.B58String() == "5dqqnW3YTxTw9EESzT63qM63zf9BYj" {
	//		//		}
	//		if message.Head.Sender.B58String() == "5dsEMMhVbww4hUXV6VzaeRfHKv1nhh" {
	//			fmt.Println("查找结果", newNode.IdInfo.Id.B58String())
	//		}
	//		//检查是否需要这个逻辑节点
	//		ok := nodeStore.CheckNeedNode(newNode.IdInfo.Id)
	//		if !ok {
	//			//		fmt.Println("不需要这个逻辑节点")
	//			return
	//		}
	//		//	fmt.Println("需要这个节点")
	//		//检查是否有这个连接
	//		_, ok = engine.GetSession(newNode.IdInfo.Id.B58String())
	//		if !ok {
	//			//		fmt.Println("没有这个连接", message.Hash)
	//			_, err := engine.AddClientConn(newNode.Addr, uint32(newNode.TcpPort), false)
	//			if err != nil {
	//				//			fmt.Println("连接失败", err)
	//				return
	//			}
	//		} else {
	//			//		fmt.Println("有连接", message.Hash)
	//		}
	//		nodeStore.AddNode(&newNode)

	//		//非超级节点判断超级节点是否改变
	//		if !nodeStore.NodeSelf.IsSuper {
	//			nearId := nodeStore.FindNearInSuper(nodeStore.NodeSelf.IdInfo.Id, nil, false)
	//			//		nearIdStr := hex.EncodeToString(nearId)
	//			fmt.Println("判断是否需要替换超级节点", nearId.B58String(), nodeStore.SuperPeerId.B58String())
	//			if nearId.B58String() == nodeStore.SuperPeerId.B58String() {
	//				return
	//			}
	//			nodeStore.SuperPeerId = nearId
	//			//		nodeStore.SuperPeerIdStr = hex.EncodeToString(nearId)
	//			fmt.Println("超级节点换为:", nodeStore.SuperPeerId.B58String())
	//		}
	//	}

}

/*
	获取相邻节点的超级节点地址
*/
func GetNearSuperAddr(c engine.Controller, msg engine.Packet) {

	//	fmt.Println("接收到查询最近超级节点请求")
	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		return
	}

	//自己处理
	if err := message.ParserContent(); err != nil {
		return
	}

	//	if message.Head.Sender.B58String() == "5dtPgE32MoURWep7QEViBkZh5iVLDZ" {
	//		fmt.Println("要查询的节点id", message.Head.Sender.B58String())
	//	}

	//	ids := make([]*utils.Multihash, 0)
	nodes := make([]nodeStore.Node, 0)
	ns := nodeStore.GetAllNodes()
	idsm := nodeStore.NewIds(message.Head.Sender, nodeStore.NodeIdLevel)
	for _, one := range ns {

		if message.Head.Sender.B58String() == one.B58String() {
			continue
		}
		//		if message.Head.Sender.B58String() == "5dtPgE32MoURWep7QEViBkZh5iVLDZ" {
		//			fmt.Println("添加了一个地址", one.B58String())
		//		}
		//		ok, remove := idsm.AddId(one)
		idsm.AddId(one)
	}

	ids := idsm.GetIds()
	for _, one := range ids {
		//		if message.Head.Sender.B58String() == "5dtPgE32MoURWep7QEViBkZh5iVLDZ" {
		//			fmt.Println("查询到的节点", one.B58String())
		//		}
		node := nodeStore.FindNode(one)
		if node != nil {
			nodes = append(nodes, *node)
		} else {
			fmt.Println("这个节点为空")
		}
	}
	data, _ := json.Marshal(nodes)

	head := NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	body := NewMessageBody(&data, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	newmessage := NewMessage(head, body)
	newmessage.BuildReplyHash(message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	//	message.Reply(gconfig.MSGID_getNearSuperIP_recv)
	msg.Session.Send(gconfig.MSGID_getNearSuperIP_recv, head.JSON(), body.JSON(), false)

}

/*
	获取相邻节点的超级节点地址返回
*/
func GetNearSuperAddr_recv(c engine.Controller, msg engine.Packet) {
	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	//	fmt.Println("获取相邻节点的超级节点地址返回")

	nodes := make([]nodeStore.Node, 0)

	//	newNode := new(nodeStore.Node)
	if err = json.Unmarshal(*message.Body.Content, &nodes); err != nil {
		//		fmt.Println("解析失败", err)
		return
	}

	//	fmt.Println("查找结果", newNode.IdInfo.Id.B58String())

	for _, newNode := range nodes {
		//		if newNode.IdInfo.Id.B58String() == "5duDDfkY1tChLKGxbAtPdPysp9ghYn" || newNode.IdInfo.Id.B58String() == "5dqqnW3YTxTw9EESzT63qM63zf9BYj" {
		//		}
		//		if message.Head.Sender.B58String() == "5dsEMMhVbww4hUXV6VzaeRfHKv1nhh" {
		//			fmt.Println("查找结果", newNode.IdInfo.Id.B58String())
		//		}
		//		fmt.Println("查找结果", newNode.IdInfo.Id.B58String(), newNode.Addr, newNode.TcpPort)
		//检查是否需要这个逻辑节点
		ok := nodeStore.CheckNeedNode(newNode.IdInfo.Id)
		if !ok {
			//		fmt.Println("不需要这个逻辑节点")
			return
		}
		//	fmt.Println("需要这个节点")
		//检查是否有这个连接
		_, ok = engine.GetSession(newNode.IdInfo.Id.B58String())
		if !ok {
			//		fmt.Println("没有这个连接", message.Hash)
			_, err := engine.AddClientConn(newNode.Addr, uint32(newNode.TcpPort), false)
			if err != nil {
				//			fmt.Println("连接失败", err)
				return
			}
		} else {
			//		fmt.Println("有连接", message.Hash)
		}
		nodeStore.AddNode(&newNode)

		//非超级节点判断超级节点是否改变
		if !nodeStore.NodeSelf.IsSuper {
			nearId := nodeStore.FindNearInSuper(nodeStore.NodeSelf.IdInfo.Id, nil, false)
			//		nearIdStr := hex.EncodeToString(nearId)
			fmt.Println("判断是否需要替换超级节点", nearId.B58String(), nodeStore.SuperPeerId.B58String())
			if nearId.B58String() == nodeStore.SuperPeerId.B58String() {
				return
			}
			nodeStore.SuperPeerId = nearId
			//		nodeStore.SuperPeerIdStr = hex.EncodeToString(nearId)
			fmt.Println("超级节点换为:", nodeStore.SuperPeerId.B58String())
		}
	}

}

/*
	接收发送的文本消息
*/
func TextMsg(c engine.Controller, msg engine.Packet) {
	//	engine.Log.Debug("收到文本消息")

	message, err := ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println(err)
		return
	}
	//	form, _ := hex.DecodeString(msg.Session.GetName())

	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		fmt.Println("发送给其他小伙伴了")
		return
	}
	if err = message.ParserContent(); err != nil {
		fmt.Println(err)
		return
	}

	//	message := new(Message)
	//	err := json.Unmarshal(msg.Data, message)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	form, _ := hex.DecodeString(msg.Session.GetName())
	//	if ok := IsSendToOtherSuper(message, msg.MsgID, nil); ok {
	//		return
	//	}
	//	fmt.Println("这个文本消息是自己的")

	content := string(*message.Body.Content)

	//发送给自己的，自己处理
	fmt.Println(content)

	//	sendId := hex.EncodeToString(message.Sender)
	//	recvId := hex.EncodeToString(message.RecvId)
	sendId := message.Head.Sender.B58String()
	recvId := message.Head.RecvId.B58String()

	now := time.Now()

	msgVO := MessageVO{
		Id:      message.Head.Sender,
		Index:   now.Unix(),
		Time:    utils.FormatTimeToSecond(now),
		Content: content,
	}
	if persistence.Friends_findIdExist(sendId) {
		persistence.Friends_addMsgNum(sendId)
	} else {
		err = persistence.Friends_add(sendId)
		if err != nil {
			fmt.Println("添加用户失败", err)
		}

	}

	select {
	case MsgChannl <- &msgVO:
	default:
	}

	err = persistence.SaveMsgLog(sendId, recvId, content)
	if err != nil {
		fmt.Println("保存日志失败", err)
	}
}

/*
	询问关闭这个链接
*/
func AskCloseConn(name string) {
	if session, ok := engine.GetSession(name); ok {
		session.Send(gconfig.MSGID_ask_close_conn_recv, nil, nil, false)
	}
}

/*
	询问关闭这个链接
	当双方都没有这个链接的引用时，就关闭这个链接
*/
func AskCloseConn_recv(c engine.Controller, msg engine.Packet) {
	//	name := msg.Session.GetName()
	//	id, err := hex.DecodeString(name)
	//	if err != nil {
	//		fmt.Println("这个session name解析错误")
	//		return
	//	}
	mh, err := utils.FromB58String(msg.Session.GetName())
	if err != nil {
		fmt.Println("这个session name解析错误")
		return
	}
	node := nodeStore.FindNode(&mh)
	if node == nil {
		//自己也没有这个连接的引用，则关闭这个链接
		msg.Session.Close()
		fmt.Println("关闭了这个没用链接", msg.Session.GetName())
	}
}

/*
	检查该消息是否是自己的
	不是自己的则自动转发出去
*/
//func IsSendToOtherSuper(messageRecv *Message, msgId uint64, form *utils.Multihash) bool {
//	//	fmt.Println(hex.EncodeToString(messageRecv.ReceSuperId))

//	if !messageRecv.Check() {
//		fmt.Println("不能为空", messageRecv)
//		return true
//	}

//	//	recvSuperId := hex.EncodeToString(messageRecv.RecvSuperId)
//	//	recvId := hex.EncodeToString(messageRecv.RecvId)
//	recvSuperId := messageRecv.RecvSuperId
//	recvId := messageRecv.RecvId

//	//	//收消息人就是自己
//	//	if nodeStore.NodeSelf.IdInfo.Id.GetIdStr() == recvId {
//	//		return false
//	//	}

//	//	//自己不是超级节点
//	//	if !nodeStore.NodeSelf.IsSuper {
//	//		//用代理方式发送出去
//	//		if session, ok := engine.GetSession(nodeStore.SuperPeerId.GetIdStr()); ok {
//	//			session.Send(msgId, messageRecv.JSON(), false)
//	//		}
//	//		if msgId == debuf_msgid {
//	//			fmt.Println("发送给超级节点")
//	//		}
//	//		return true
//	//	}

//	//	//接收者超级节点id是自己，接收者不是自己，是自己的代理节点
//	//	if nodeStore.NodeSelf.IdInfo.Id.GetIdStr() == recvSuperId {
//	//		//查找自己的代理节点
//	//		if msgId == debuf_msgid {
//	//			fmt.Println("查找的是自己的代理节点")
//	//		}
//	//		//查找代理节点
//	//		if _, ok := nodeStore.GetProxyNode(recvId); ok {
//	//			//发送给代理节点
//	//			if session, ok := engine.GetSession(recvId); ok {
//	//				if msgId == debuf_msgid {
//	//					fmt.Println("发送出去了111")
//	//				}
//	//				session.Send(msgId, messageRecv.JSON(), false)
//	//			} else {
//	//				//这个链接断开了
//	//				if msgId == debuf_msgid {
//	//					fmt.Println("这个链接断开了")
//	//				}
//	//			}
//	//		} else {
//	//			//该节点不在线了
//	//			if msgId == debuf_msgid {
//	//				fmt.Println("111111111", recvId, recvSuperId)
//	//			}
//	//			//TODO 节点不在线，可能是切换到其他超级节点了，应该回复给消息发送者一个不在线的消息
//	//		}
//	//		return true
//	//	}

//	//	//消息是发送给逻辑节点，不是准确发送给一个节点，离逻辑节点最近的人接收并处理消息
//	//	targetId := nodeStore.FindNearInSuper(messageRecv.RecvSuperId, form, true)
//	//	if !messageRecv.Accurate && hex.EncodeToString(targetId) == nodeStore.NodeSelf.IdInfo.Id.GetIdStr() {
//	//		return false
//	//	}

//	//	//消息转发给其他节点
//	//	//	targetId = nodeStore.FindNearInSuper(messageRecv.RecvSuperId, form, false)
//	//	if session, ok := engine.GetSession(hex.EncodeToString(targetId)); ok {
//	//		session.Send(msgId, messageRecv.JSON(), false)
//	//	} else {
//	//		if msgId == debuf_msgid {
//	//			fmt.Println("1111这个session不存在", hex.EncodeToString(targetId), hex.EncodeToString(messageRecv.RecvSuperId))
//	//		}
//	//	}
//	//	return true

//	//------------------

//	if !nodeStore.NodeSelf.IsSuper {
//		if nodeStore.NodeSelf.IdInfo.Id.B58String() == recvId.B58String() {
//			return false
//		} else {
//			if messageRecv.Accurate {
//				//发错节点了
//				fmt.Println("发错节点了", nodeStore.NodeSelf.IdInfo.Id.B58String(), recvSuperId.B58String(), recvId.B58String())
//				return true
//			} else {
//				if session, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
//					session.Send(msgId, messageRecv.JSON(), false)
//				}
//				if msgId == debuf_msgid {
//					fmt.Println("发送给超级节点")
//				}
//				return true
//			}
//		}
//	}
//	if recvId.B58String() == recvSuperId.B58String() {
//		if recvId.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
//			return false
//		} else {
//			if msgId == debuf_msgid {
//				fmt.Println("----1111111")
//			}
//			targetId := nodeStore.FindNearInSuper(messageRecv.RecvSuperId, form, true)
//			if msgId == debuf_msgid {
//				fmt.Println("----222222222")
//			}
//			if targetId.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
//				//查找代理节点
//				_, ok := nodeStore.GetProxyNode(recvId.B58String())
//				if msgId == debuf_msgid {
//					fmt.Println("----333333333")
//				}
//				if ok {
//					//发送给代理节点
//					if session, ok := engine.GetSession(recvId.B58String()); ok {
//						if msgId == debuf_msgid {
//							fmt.Println("发送出去了111")
//						}
//						session.Send(msgId, messageRecv.JSON(), false)
//					} else {
//						//这个链接断开了
//						if msgId == debuf_msgid {
//							fmt.Println("这个链接断开了")
//						}
//					}
//				} else {
//					if !messageRecv.Accurate {
//						return false
//					}

//					if msgId == debuf_msgid {
//						fmt.Println("该节点不在线")
//					}
//					//该节点不在线了
//					if msgId == debuf_msgid {
//						fmt.Println("111111111", recvId, recvSuperId)
//					}
//				}
//				return true
//			}

//			session, ok := engine.GetSession(targetId.B58String())
//			if ok {
//				if msgId == debuf_msgid {
//					fmt.Println("发送出去了222")
//				}
//				session.Send(msgId, messageRecv.JSON(), false)
//			} else {
//				if msgId == debuf_msgid {
//					fmt.Println("这个链接断开了222")
//				}
//				if msgId == debuf_msgid {
//					fmt.Println("-=-=-=-= 这个session已经断开")
//				}
//			}
//		}
//		if msgId == debuf_msgid {
//			fmt.Println("4444444444", recvId, recvSuperId)
//		}
//		return true

//	} else {

//		if nodeStore.NodeSelf.IdInfo.Id.B58String() == recvSuperId.B58String() {
//			if recvId == nil {
//				return false
//			} else {
//				if msgId == debuf_msgid {
//					fmt.Println("----444444444")
//				}
//				_, ok := nodeStore.GetProxyNode(recvId.B58String())
//				if msgId == debuf_msgid {
//					fmt.Println("----555555555")
//				}
//				if ok {
//					if session, ok := engine.GetSession(recvId.B58String()); ok {
//						if msgId == debuf_msgid {
//							fmt.Println("发送出去了")
//						}
//						session.Send(msgId, messageRecv.JSON(), false)
//					} else {
//						if msgId == debuf_msgid {
//							fmt.Println("这个session不存在")
//						}
//					}
//				}
//				//代理节点转移或下线，忽略这个消息
//				if msgId == debuf_msgid {
//					fmt.Println("22222222")
//				}
//				return true
//			}
//		}
//		if msgId == debuf_msgid {
//			fmt.Println("----6666666666")
//		}
//		targetId := nodeStore.FindNearInSuper(messageRecv.RecvSuperId, form, true)
//		if msgId == debuf_msgid {
//			fmt.Println("----777777777777")
//		}
//		// hex.EncodeToString(targetId) == nodeStore.NodeSelf.IdInfo.Id.GetIdStr()
//		if targetId.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
//			if messageRecv.Accurate {
//				//该节点不在线
//				fmt.Println("该节点不在线，这个包会被丢弃", msgId, targetId.B58String(),
//					messageRecv.RecvSuperId.B58String(), string(*messageRecv.JSON()))
//				if msgId == debuf_msgid {
//					fmt.Println("33333333")
//				}
//				return true
//			} else {
//				return false
//			}
//		}

//		session, ok := engine.GetSession(targetId.B58String())
//		if ok {
//			session.Send(msgId, messageRecv.JSON(), false)
//		}
//		if msgId == debuf_msgid {
//			fmt.Println("5555555555", recvId, recvSuperId)
//		}
//		return true
//	}

//}

/*
	检查该消息是否是自己的
	不是自己的则自动转发出去
*/
func IsSendToOtherSuperToo(messageHead *MessageHead, dataplus *[]byte, msgId uint64, form *utils.Multihash) bool {
	//	fmt.Println(hex.EncodeToString(messageRecv.ReceSuperId))

	//	if !messageRecv.Check() {
	//		fmt.Println("不能为空", messageRecv)
	//		return true
	//	}

	//	recvSuperId := hex.EncodeToString(messageRecv.RecvSuperId)
	//	recvId := hex.EncodeToString(messageRecv.RecvId)
	recvSuperId := messageHead.RecvSuperId
	recvId := messageHead.RecvId

	//	//收消息人就是自己
	//	if nodeStore.NodeSelf.IdInfo.Id.GetIdStr() == recvId {
	//		return false
	//	}

	//	//自己不是超级节点
	//	if !nodeStore.NodeSelf.IsSuper {
	//		//用代理方式发送出去
	//		if session, ok := engine.GetSession(nodeStore.SuperPeerId.GetIdStr()); ok {
	//			session.Send(msgId, messageRecv.JSON(), false)
	//		}
	//		if msgId == debuf_msgid {
	//			fmt.Println("发送给超级节点")
	//		}
	//		return true
	//	}

	//	//接收者超级节点id是自己，接收者不是自己，是自己的代理节点
	//	if nodeStore.NodeSelf.IdInfo.Id.GetIdStr() == recvSuperId {
	//		//查找自己的代理节点
	//		if msgId == debuf_msgid {
	//			fmt.Println("查找的是自己的代理节点")
	//		}
	//		//查找代理节点
	//		if _, ok := nodeStore.GetProxyNode(recvId); ok {
	//			//发送给代理节点
	//			if session, ok := engine.GetSession(recvId); ok {
	//				if msgId == debuf_msgid {
	//					fmt.Println("发送出去了111")
	//				}
	//				session.Send(msgId, messageRecv.JSON(), false)
	//			} else {
	//				//这个链接断开了
	//				if msgId == debuf_msgid {
	//					fmt.Println("这个链接断开了")
	//				}
	//			}
	//		} else {
	//			//该节点不在线了
	//			if msgId == debuf_msgid {
	//				fmt.Println("111111111", recvId, recvSuperId)
	//			}
	//			//TODO 节点不在线，可能是切换到其他超级节点了，应该回复给消息发送者一个不在线的消息
	//		}
	//		return true
	//	}

	//	//消息是发送给逻辑节点，不是准确发送给一个节点，离逻辑节点最近的人接收并处理消息
	//	targetId := nodeStore.FindNearInSuper(messageRecv.RecvSuperId, form, true)
	//	if !messageRecv.Accurate && hex.EncodeToString(targetId) == nodeStore.NodeSelf.IdInfo.Id.GetIdStr() {
	//		return false
	//	}

	//	//消息转发给其他节点
	//	//	targetId = nodeStore.FindNearInSuper(messageRecv.RecvSuperId, form, false)
	//	if session, ok := engine.GetSession(hex.EncodeToString(targetId)); ok {
	//		session.Send(msgId, messageRecv.JSON(), false)
	//	} else {
	//		if msgId == debuf_msgid {
	//			fmt.Println("1111这个session不存在", hex.EncodeToString(targetId), hex.EncodeToString(messageRecv.RecvSuperId))
	//		}
	//	}
	//	return true

	//------------------

	if !nodeStore.NodeSelf.IsSuper {
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == recvId.B58String() {
			return false
		} else {
			if messageHead.Accurate {
				//发错节点了
				fmt.Println("发错节点了", nodeStore.NodeSelf.IdInfo.Id.B58String(), recvSuperId.B58String(), recvId.B58String())
				return true
			} else {
				if session, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
					session.Send(msgId, messageHead.JSON(), dataplus, false)
				}
				if msgId == debuf_msgid {
					fmt.Println("发送给超级节点")
				}
				return true
			}
		}
	}
	if recvId.B58String() == recvSuperId.B58String() {
		if recvId.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
			return false
		} else {
			if msgId == debuf_msgid {
				fmt.Println("----1111111")
			}
			targetId := nodeStore.FindNearInSuper(messageHead.RecvSuperId, form, true)
			if msgId == debuf_msgid {
				fmt.Println("----222222222", targetId.B58String(), nodeStore.NodeSelf.IdInfo.Id.B58String(), form)
			}
			if targetId.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
				//查找代理节点
				_, ok := nodeStore.GetProxyNode(recvId.B58String())
				if msgId == debuf_msgid {
					fmt.Println("----333333333")
				}
				if ok {
					//发送给代理节点
					if session, ok := engine.GetSession(recvId.B58String()); ok {
						if msgId == debuf_msgid {
							fmt.Println("发送出去了111")
						}
						session.Send(msgId, messageHead.JSON(), dataplus, false)
					} else {
						//这个链接断开了
						if msgId == debuf_msgid {
							fmt.Println("这个链接断开了")
						}
					}
				} else {
					if !messageHead.Accurate {
						return false
					}

					if msgId == debuf_msgid {
						fmt.Println("该节点不在线")
					}
					//该节点不在线了
					if msgId == debuf_msgid {
						fmt.Println("111111111", recvId, recvSuperId)
					}
				}
				return true
			}

			session, ok := engine.GetSession(targetId.B58String())
			if ok {
				if msgId == debuf_msgid {
					fmt.Println("发送出去了222")
				}
				session.Send(msgId, messageHead.JSON(), dataplus, false)
			} else {
				if msgId == debuf_msgid {
					fmt.Println("这个链接断开了222")
				}
				if msgId == debuf_msgid {
					fmt.Println("-=-=-=-= 这个session已经断开")
				}
			}
		}
		if msgId == debuf_msgid {
			fmt.Println("4444444444", recvId, recvSuperId)
		}
		return true

	} else {

		if nodeStore.NodeSelf.IdInfo.Id.B58String() == recvSuperId.B58String() {
			if recvId == nil {
				return false
			} else {
				if msgId == debuf_msgid {
					fmt.Println("----444444444")
				}
				_, ok := nodeStore.GetProxyNode(recvId.B58String())
				if msgId == debuf_msgid {
					fmt.Println("----555555555")
				}
				if ok {
					if session, ok := engine.GetSession(recvId.B58String()); ok {
						if msgId == debuf_msgid {
							fmt.Println("发送出去了")
						}
						session.Send(msgId, messageHead.JSON(), dataplus, false)
					} else {
						if msgId == debuf_msgid {
							fmt.Println("这个session不存在")
						}
					}
				}
				//代理节点转移或下线，忽略这个消息
				if msgId == debuf_msgid {
					fmt.Println("22222222")
				}
				return true
			}
		}
		if msgId == debuf_msgid {
			fmt.Println("----6666666666")
		}
		targetId := nodeStore.FindNearInSuper(messageHead.RecvSuperId, form, true)
		if msgId == debuf_msgid {
			fmt.Println("----777777777777")
		}
		// hex.EncodeToString(targetId) == nodeStore.NodeSelf.IdInfo.Id.GetIdStr()
		if targetId.B58String() == nodeStore.NodeSelf.IdInfo.Id.B58String() {
			if messageHead.Accurate {
				//该节点不在线
				fmt.Println("该节点不在线，这个包会被丢弃", msgId, targetId.B58String(),
					messageHead.RecvSuperId.B58String(), string(*messageHead.JSON()))
				if msgId == debuf_msgid {
					fmt.Println("33333333")
				}
				return true
			} else {
				return false
			}
		}

		session, ok := engine.GetSession(targetId.B58String())
		if ok {
			session.Send(msgId, messageHead.JSON(), dataplus, false)
		}
		if msgId == debuf_msgid {
			fmt.Println("5555555555", recvId, recvSuperId)
		}
		return true
	}

}

/*
	广播给其他人
*/
//func MulticastOther() {

//}
