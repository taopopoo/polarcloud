package core

//import (
//	"fmt"
//	"time"
//	gconfig "yunpan/config"
//	"yunpan/core/config"
//	"yunpan/core/engine"
//	msg "yunpan/core/message_center"
//	"yunpan/core/nodeStore"
//	_ "yunpan/core/persistence"
//)

//func InitNodeStore(node *nodeStore.Node) {
//	//	once.Do(run)
//	//	Manager = NewNodeManager(gconfig.NodeIDLevel, node)
//	nodeStore.InitNodeStore(node)

//	go read()
//	go getNearSuperIP()
//}

///*
//	处理查找节点的请求
//	本节点定期查询已知节点是否在线，更新节点信息
//*/
//func read() {
//	for {
//		nodeIdStr := <-nodeStore.Manager.OutFindNode

//		mhead := msg.NewMessageHead(nodeIdStr, nodeIdStr, false)
//		mbody := msg.NewMessageBody(nil, "", nil, 0)
//		message := msg.NewMessage(mhead, mbody)
//		//		message.SendForce(gconfig.MSGID_findSuperID)
//		if nodeStore.Manager.NodeSelf.IsSuper {
//			nearId := nodeStore.Manager.FindNearInSuper(nodeIdStr, nil, false)
//			if nearId == nil {
//				continue
//			}
//			session, ok := engine.GetSession(nearId.B58String())
//			if !ok {
//				//			fmt.Println("查找逻辑节点，这个session未找到")
//				continue
//			}
//			session.Send(gconfig.MSGID_checkNodeOnline, message.Head.JSON(), message.Body.JSON(), false)
//		} else {
//			if nodeStore.Manager.SuperPeerId == nil {
//				continue
//			}
//			session, ok := engine.GetSession(nodeStore.Manager.SuperPeerId.B58String())
//			if !ok {
//				fmt.Println("查找逻辑节点，这个session未找到", nodeStore.Manager.SuperPeerId.B58String())
//				continue
//			}
//			session.Send(gconfig.MSGID_checkNodeOnline, message.Head.JSON(), message.Body.JSON(), false)
//		}

//	}
//}

///*
//	定时获得相邻节点的超级节点ip地址
//*/
//func getNearSuperIP() {
//	for {
//		for _, key := range nodeStore.Manager.GetAllNodes() {
//			//直接查找最近的超级节点
//			mhead := msg.NewMessageHead(key, key, false)
//			mbody := msg.NewMessageBody(nil, "", nil, 0)
//			message := msg.NewMessage(mhead, mbody)
//			message.BuildHash()
//			session, ok := engine.GetSession(key.B58String())
//			if ok {
//				//				fmt.Println("给这个session发送消息成功", key.B58String())
//				session.Send(gconfig.MSGID_getNearSuperIP, mhead.JSON(), mbody.JSON(), false)
//				time.Sleep(time.Second * 1)
//			}
//		}
//		//		fmt.Println("完成一轮查找邻居节点地址")
//		time.Sleep(time.Second * config.Time_getNear_super_ip)
//	}
//}
