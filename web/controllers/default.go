package controllers

import (
	"fmt"
	gconfig "yunpan/config"
	"yunpan/core/cache_store"
	"yunpan/core/message_center"
	"yunpan/core/nodeStore"
	"yunpan/core/utils"
	"yunpan/wallet/mining"

	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

func (this *MainController) Get() {

	this.TplName = "index.tpl"
}

/*
	测试
*/
func (this *MainController) Test() {

	this.Data["Ip"] = nodeStore.NodeSelf.Addr

	this.Data["RootExist"] = cache_store.Root.Exist

	this.Data["IsSuper"] = nodeStore.NodeSelf.IsSuper
	this.Data["SuperId"] = nodeStore.SuperPeerId.B58String()

	//	fmt.Println("首页")
	this.Data["ID"] = nodeStore.NodeSelf.IdInfo.Id.B58String()

	ids := nodeStore.GetAllNodes()
	idsStr := make([]string, 0)
	for _, one := range ids {
		idsStr = append(idsStr, one.B58String())
	}
	this.Data["ids"] = idsStr

	names := cache_store.Debug_GetAllName()
	this.Data["names"] = names

	this.TplName = "test.tpl"
}

/*
	发送消息
*/
func (this *MainController) SendMeg() {
	id := this.GetString("id")
	mh, err := utils.FromB58String(id)
	if err != nil {
		fmt.Println(err)
		return
	}

	//	recvId, _ := hex.DecodeString(id)
	recvId := utils.Multihash(mh)
	content := []byte(this.GetString("content"))
	//	fmt.Println("1111111111111")
	//	var message msg.Message

	mhead := message_center.NewMessageHead(&recvId, &recvId, true)
	mbody := message_center.NewMessageBody(&content, "", nil, 0)
	message := message_center.NewMessage(mhead, mbody)
	message.Send(gconfig.MSGID_TextMsg)

	//	if nodeStore.NodeSelf.IsSuper {
	//		//		fmt.Println("888888888888888")
	//		message := message_center.Message{
	//			RecvId:        &recvId,                        //
	//			RecvSuperId:   &recvId,                        //接收者的超级节点id
	//			CreateTime:    utils.TimeFormatToNanosecond(), //消息创建时间unix
	//			SenderSuperId: nodeStore.NodeSelf.IdInfo.Id,   //发送者超级节点id
	//			Sender:        nodeStore.NodeSelf.IdInfo.Id,
	//			Accurate:      true,
	//			Content:       []byte(content),
	//		}
	//		//		fmt.Println("999999999999999")
	//		message.BuildHash()
	//		//		fmt.Println("1010101010101")
	//		//		nearId := nodeStore.FindNearInSuper(recvId, []byte{}, false)
	//		message_center.IsSendToOtherSuper(&message, message_center.MSGID_TextMsg, nil)
	//		//		fmt.Println("11 11 11 11 11 11")
	//	} else {
	//		//		fmt.Println("2222222222")
	//		//用代理方式
	//		message := message_center.Message{
	//			RecvId:        &recvId,                        //
	//			RecvSuperId:   &recvId,                        //接收者的超级节点id
	//			CreateTime:    utils.TimeFormatToNanosecond(), //消息创建时间unix
	//			SenderSuperId: nodeStore.SuperPeerId,          //发送者超级节点id
	//			Sender:        nodeStore.NodeSelf.IdInfo.Id,   //发送者节点id
	//			Accurate:      true,
	//			Content:       []byte(content),
	//		}
	//		//		fmt.Println("333333333333")
	//		message.BuildHash()
	//		//		fmt.Println("4444444444444")
	//		//		nearId := nodeStore.FindNearInSuper(recvId, []byte{}, false)
	//		//		message_center.IsSendToOtherSuper(&message, message_center.MSGID_TextMsg, []byte{})
	//		if sess, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
	//			//			fmt.Println("55555555555")
	//			sess.Send(message_center.MSGID_TextMsg, message.JSON(), false)
	//		}
	//		//		fmt.Println("6666666666666666")
	//	}

	out := make(map[string]interface{})
	out["Code"] = 0
	this.Data["json"] = out
	this.ServeJSON(true)
	//	fmt.Println("777777777777")
}

/*
	代理
*/
func (this *MainController) AgentToo() {
	fmt.Println("没有命中")
}

/*
	测试按钮
*/
func (this *MainController) BtTest() {
	fmt.Println("测试按钮")
	mining.Seekvote()
	out := make(map[string]interface{})
	out["Code"] = 0
	this.Data["json"] = out
	this.ServeJSON(true)
}
