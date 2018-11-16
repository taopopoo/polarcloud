package core

//import (
//	"encoding/hex"
//	"encoding/json"
//	"fmt"
//	"strings"
//	"time"

//	addrm "yunpan/core/addr_manager"
//	"yunpan/core/engine"
//	msgc "yunpan/core/message_center"
//	"yunpan/core/nodeStore"
//	"yunpan/core/utils"
//)

//保存一个键值对
//func SaveData(key, value string) {
//	clientConn, _ := engine.GetSession(nodeStore.SuperName)
//	data := []byte(key + "!" + value)
//	clientConn.Send(msgc.SaveKeyValueReqNum, &data, false)
//}

////给所有客户端发送消息
//func SendMsgForAll(message string) {
//	messageSend := msgc.Message{
//		//		ProtoId:    msgc.MSGID_Text,
//		CreateTime: time.Now().Unix(),
//		Content:    []byte(message),
//	}
//	for idOne, nodeOne := range nodeStore.GetAllNodes() {
//		if clientConn, ok := engine.GetSession(string(nodeOne.IdInfo.Build())); ok {
//			temp, _ := hex.DecodeString(idOne)
//			messageSend.ReceSuperId = temp
//			data, _ := json.Marshal(messageSend)
//			clientConn.Send(msgc.SendMessageNum, &data, false)
//		}
//	}
//}

////给某个人发送消息
//func SendMsgForOne_opt(target, message string) {
//	temp, _ := hex.DecodeString(target)
//	messageSend := &msgc.Message{
//		ReceSuperId: temp,
//		Sender:      nodeStore.NodeSelf.IdInfo.Id,
//		//		ProtoId:     msgc.MSGID_Text,
//		CreateTime: time.Now().Unix(),
//		Content:    []byte(message),
//		//		Accurate:   true,
//	}
//	msgc.SendMessage(messageSend)
//}

///*
//	发送消息给一个域名
//*/
//func SendMsgForDomain(tdomain, msg string) {

//}

///*
//	查看本地保存的所有节点id
//*/
//func See() {
//	allNodes := nodeStore.GetAllNodes()
//	for key, _ := range allNodes {
//		fmt.Println(key)
//	}
//}

/*
	查看本地保存的节点中，小于本节点id的所有节点
*/
//func SeeLeftNode() {
//	nodes := nodeStore.GetLeftNode(*nodeStore.Root.IdInfo.GetBigIntId(), nodeStore.MaxRecentCount)
//	for _, id := range nodes {
//		fmt.Println(id.IdInfo.GetId())
//	}
//}

/*
	查看本地保存的节点中，大于本节点id的所有节点
*/
//func SeeRightNode() {
//	nodes := nodeStore.GetRightNode(*nodeStore.Root.IdInfo.GetBigIntId(), nodeStore.MaxRecentCount)
//	for _, id := range nodes {
//		fmt.Println(id.IdInfo.GetId())
//	}
//}

///*
//	查看超级节点
//*/
//func SeeSuperNode() {
//	_, ok := engine.GetSession(nodeStore.SuperName)
//	fmt.Println(nodeStore.SuperName, "是否已连接：", ok)
//}

///*
//	添加一个超级节点ip地址
//	@addr   例如：121.45.6.157:8076
//*/
//func AddAddr(addr string) {
//	addrs := strings.Split(addr, ":")
//	if len(addrs) != 2 {
//		return
//	}
//	if !utils.IsOnlyIp(addrs[0]) {
//		return
//	}
//	if addrm.CheckOnline(addr) {
//		addrm.AddSuperPeerAddr(addr)
//	}
//}

/*
	注册一个域名帐号
	@name     姓名
	@email    邮箱
	@domain   网络唯一标识
*/
//func CreateAccount(name, email, domain string) {
//	//连接网络并得到一个idinfo
//	//	idInfo, err := GetId()
//	//	if err == nil {
//	//		nodeStore.Init_IdInfo = *idInfo
//	//	} else {
//	//		fmt.Println("从网络中获得idinfo失败")
//	//		return
//	//	}
//}
