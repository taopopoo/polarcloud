package message_center

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	gconfig "yunpan/config"
	"yunpan/core/config"
	"yunpan/core/engine"
	"yunpan/core/nodeStore"
	"yunpan/core/utils"
)

var sendHash = new(sync.Map) //保存1分钟内的消息sendhash，用于判断重复消息
var sendhashTask = utils.NewTask(sendhashTaskFun)

func sendhashTaskFun(class, params string) {
	sendHash.Delete(params)
}

/*
	检查这个消息是否发送过
*/
func CheckHash(sendhash string) bool {
	_, ok := sendHash.Load(sendhash)
	if !ok {
		sendHash.Store(sendhash, nil)
		sendhashTask.Add(time.Now().Unix()+60, "", sendhash)
	}
	return !ok
}

var (
	task        = utils.NewTask(msgTimeOutProsess)
	msgHashLock = new(sync.RWMutex)
	msgHash     = make(map[string]int64)
)

/*
	添加一个消息超时
*/
func addMsgTimeOut(md5 string) {
	now := time.Now().Unix()
	msgHashLock.Lock()
	msgHash[md5] = now
	msgHashLock.Unlock()
	task.Add(now+60*10, config.TSK_msg_timeout_remove, md5)
}

/*
	检查一个消息是否超时或者非法
*/
func checkMsgTimeOut(md5 string) (ok bool) {
	msgHashLock.Lock()
	_, ok = msgHash[md5]
	if ok {
		delete(msgHash, md5)
	}
	msgHashLock.Unlock()
	return
}

type MessageHead struct {
	RecvId        *utils.Multihash `json:"r_id"`     //接收者id
	RecvSuperId   *utils.Multihash `json:"r_s_id"`   //接收者的超级节点id
	Sender        *utils.Multihash `json:"s_id"`     //发送者id
	SenderSuperId *utils.Multihash `json:"s_s_id"`   //发送者超级节点id
	Accurate      bool             `json:"accurate"` //是否准确发送给一个节点
}

func NewMessageHead(recvid, recvSuperid *utils.Multihash, accurate bool) *MessageHead {
	if nodeStore.NodeSelf.IsSuper {
		//		head := NewMessageHead(nil, nil, nil, nodeStore.NodeSelf.IdInfo.Id, false)
		return &MessageHead{
			RecvId:        recvid,                       //接收者id
			RecvSuperId:   recvSuperid,                  //接收者的超级节点id
			Sender:        nodeStore.NodeSelf.IdInfo.Id, //发送者id
			SenderSuperId: nodeStore.NodeSelf.IdInfo.Id, //发送者超级节点id
			Accurate:      accurate,                     //是否准确发送给一个节点
		}
	} else {
		return &MessageHead{
			RecvId:        recvid,                       //接收者id
			RecvSuperId:   recvSuperid,                  //接收者的超级节点id
			Sender:        nodeStore.NodeSelf.IdInfo.Id, //发送者id
			SenderSuperId: nodeStore.SuperPeerId,        //发送者超级节点id
			Accurate:      accurate,                     //是否准确发送给一个节点
		}
	}
}

/*
	检查参数是否合法
*/
func (this *MessageHead) Check() bool {
	if this.RecvId == nil {
		return false
	}
	if this.RecvSuperId == nil {
		return false
	}
	if this.Sender == nil {
		return false
	}
	if this.SenderSuperId == nil {
		return false
	}
	return true
}

func (this *MessageHead) JSON() *[]byte {
	//	this.BuildReplyHash()
	bs, _ := json.Marshal(this)
	return &bs
}

type MessageBody struct {
	CreateTime string           `json:"c_time"`  //消息创建时间unix
	ReplyTime  string           `json:"r_time"`  //消息回复时间unix
	Hash       *utils.Multihash `json:"hash"`    //消息的hash值
	ReplyHash  *utils.Multihash `json:"r_hash"`  //回复消息的hash
	SendRand   uint64           `json:"s_rand"`  //发送随机数
	RecvRand   uint64           `json:"r_rand"`  //接收随机数
	Content    *[]byte          `json:"content"` //发送的内容
}

func NewMessageBody(content *[]byte, creatTime string, hash *utils.Multihash, sendRand uint64) *MessageBody {
	return &MessageBody{
		CreateTime: creatTime,
		Hash:       hash,
		SendRand:   sendRand,
		Content:    content, //发送的内容
	}
}

func (this *MessageBody) JSON() *[]byte {
	//	this.BuildReplyHash()
	bs, _ := json.Marshal(this)
	return &bs
}

/*
	发送消息序列化对象
*/
type Message struct {
	msgid    uint64       //
	Head     *MessageHead `json:"head"` //
	Body     *MessageBody `json:"body"` //
	DataPlus *[]byte      `json:"dp"`   //body部分加密数据，消息路由时候不需要解密，临时保存
}

//type Message struct {
//	RecvId        *utils.Multihash `json:"recv_id"`         //接收者id
//	RecvSuperId   *utils.Multihash `json:"recv_super_id"`   //接收者的超级节点id
//	CreateTime    string           `json:"create_time"`     //消息创建时间unix
//	Sender        *utils.Multihash `json:"sender_id"`       //发送者id
//	SenderSuperId *utils.Multihash `json:"sender_super_id"` //发送者超级节点id
//	ReplyTime     string           `json:"reply_time"`      //消息回复时间unix
//	Hash          *utils.Multihash `json:"hash"`            //消息的hash值
//	ReplyHash     *utils.Multihash `json:"reply_hash"`      //回复消息的hash
//	Accurate      bool             `json:"accurate"`        //是否准确发送给一个节点
//	Content       []byte           `json:"content"`         //发送的内容
//	Rand          uint64           `json:"rand"`            //随机数
//}

func (this *Message) BuildHash() {
	this.Body.ReplyHash = nil
	this.Body.Hash = nil
	this.Body.SendRand = utils.GetAccNumber()
	this.Body.RecvRand = 0
	this.Body.CreateTime = utils.TimeFormatToNanosecond()
	bs, _ := json.Marshal(this)
	hash := sha1.New()
	hash.Write(bs)
	mhBs, _ := utils.Encode(hash.Sum(nil), gconfig.HashCode)
	mh := utils.Multihash(mhBs)
	this.Body.Hash = &mh
	//	this.Hash = hex.EncodeToString(hash.Sum(nil))
}
func (this *Message) BuildReplyHash(createtime string, sendhash *utils.Multihash, sendrand uint64) {
	this.Body.CreateTime = createtime
	this.Body.Hash = sendhash
	this.Body.SendRand = sendrand
	this.Body.ReplyHash = nil
	this.Body.RecvRand = utils.GetAccNumber()
	this.Body.ReplyTime = utils.TimeFormatToNanosecond()
	bs, _ := json.Marshal(this)
	hash := sha1.New()
	hash.Write(bs)
	mhBs, _ := utils.Encode(hash.Sum(nil), gconfig.HashCode)
	mh := utils.Multihash(mhBs)
	this.Body.ReplyHash = &mh
	//	this.ReplyHash = hex.EncodeToString(hash.Sum(nil))
}

var debuf_msgid uint64 = 0

//var debuf_msgid uint64 = 1000
//var debuf_msgid uint64 = MSGID_TextMsg
//var debuf_msgid uint64 = gconfig.MSGID_findSuperID

/*
	检查该消息是否是自己的
	不是自己的则自动转发出去
	@safe 安全协议使用
*/
func (this *Message) Send(msgId uint64) bool {
	//安全协议不需buildhash
	this.BuildHash()
	//TODO 这里对消息加密
	if nodeStore.NodeSelf.IsSuper {
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 111111111111")
		}
		//收消息人就是自己
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == this.Head.RecvId.B58String() {
			return false
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 333333333333333")
		}
		//查找代理节点
		if _, ok := nodeStore.GetProxyNode(this.Head.RecvId.B58String()); ok {
			//发送给代理节点
			if session, ok := engine.GetSession(this.Head.RecvId.B58String()); ok {
				if msgId == debuf_msgid {
					fmt.Println("-=-=- 4444444444444")
				}
				session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
			} else {
				fmt.Println("这个代理节点的链接断开了")
			}
			return true
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 5555555")
		}

		//		fmt.Println(string(*this.Head.JSON()))
		var targetId *utils.Multihash
		if this.Head.Accurate {
			targetId = nodeStore.FindNearInSuper(this.Head.RecvSuperId, nil, false)
		} else {
			targetId = nodeStore.FindNearInSuper(this.Head.RecvSuperId, nil, true)
		}
		//		fmt.Println("本节点的其他超级节点", msgId, nodeStore.GetAllNodes(), targetId.B58String())
		if targetId == nil {
			fmt.Println("没有可用的邻居节点")
			return true
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 666666666666")
		}
		//收消息人就是自己
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == targetId.B58String() {
			if msgId == debuf_msgid {
				fmt.Println("-=-=- 777777777777777")
			}
			return false
		}

		if msgId == debuf_msgid {
			fmt.Println("-=-=- 88888888888888")
		}

		//转发出去
		if session, ok := engine.GetSession(targetId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
			if msgId == debuf_msgid {
				fmt.Println("-=-=- 999999999999")
			}
		} else {
			fmt.Println("111 这个超级节点的链接断开了", msgId, targetId.B58String())
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 101010101010101010101010")
		}
		return true

		//		return IsSendToOtherSuperToo(this.Head, this.Body.JSON(), msgId, nil)
	} else {
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 22222222222")
		}
		if nodeStore.SuperPeerId == nil {
			fmt.Println("没有可用的超级节点")
			return true
		}
		if session, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
		} else {
			fmt.Println("超级节点的session未找到")
		}
		return true
	}
}

/*
	强制发送消息给邻居节点
	用于节点发现，实现网络自治
*/
func (this *Message) SendForce(msgId uint64) bool {
	this.BuildHash()
	//TODO 这里对消息加密

	fmt.Println("本节点是否是超级节点", nodeStore.NodeSelf.IsSuper)

	if nodeStore.NodeSelf.IsSuper {
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 111111111111")
		}
		//收消息人就是自己
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == this.Head.RecvId.B58String() {
			return false
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 333333333333333")
		}
		//查找代理节点
		if _, ok := nodeStore.GetProxyNode(this.Head.RecvId.B58String()); ok {
			//发送给代理节点
			if session, ok := engine.GetSession(this.Head.RecvId.B58String()); ok {
				if msgId == debuf_msgid {
					fmt.Println("-=-=- 4444444444444")
				}
				session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
			} else {
				fmt.Println("这个代理节点的链接断开了")
			}
			return true
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 5555555")
		}
		targetId := nodeStore.FindNearInSuper(this.Head.RecvSuperId, nil, false)
		if targetId == nil {
			fmt.Println("没有可用的邻居节点")
			return true
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 666666666666")
		}
		//收消息人就是自己
		if nodeStore.NodeSelf.IdInfo.Id.B58String() == targetId.B58String() {
			if msgId == debuf_msgid {
				fmt.Println("-=-=- 777777777777777")
			}
			return false
		}

		if msgId == debuf_msgid {
			fmt.Println("-=-=- 88888888888888")
		}

		//转发出去
		if session, ok := engine.GetSession(targetId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
			if msgId == debuf_msgid {
				fmt.Println("-=-=- 999999999999")
			}
		} else {
			fmt.Println("222 这个超级节点的链接断开了")
		}
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 101010101010101010101010")
		}
		return true

		//		return IsSendToOtherSuperToo(this.Head, this.Body.JSON(), msgId, nil)
	} else {
		if msgId == debuf_msgid {
			fmt.Println("-=-=- 22222222222")
		}
		if nodeStore.SuperPeerId == nil {
			fmt.Println("没有可用的超级节点")
			return true
		}
		if session, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
		} else {
			fmt.Println("超级节点的session未找到")
		}
		return true
	}
}

/*
	检查该消息是否是自己的
	不是自己的则自动转发出去
*/
func (this *Message) IsSendOther(form *utils.Multihash) bool {
	return IsSendToOtherSuperToo(this.Head, this.DataPlus, this.msgid, form)
}

/*
	解析内容
*/
func (this *Message) ParserContent() error {
	//TODO 解密内容

	this.Body = new(MessageBody)
	err := json.Unmarshal(*this.DataPlus, this.Body)
	if err != nil {
		return err
	}
	return nil
}

/*
	验证hash
*/
func (this *Message) CheckSendhash() bool {
	//TODO 验证sendhash是否正确
	//TODO 验证时间不能相差太远

	//验证sendhash是否已经接受过此消息
	return CheckHash(this.Body.Hash.B58String())
}

/*
	验证hash
*/
func (this *Message) CheckReplyhash() bool {
	//TODO 验证replyhash是否正确
	//TODO 验证时间不能相差太远

	//验证replyhash是否已经接受过此消息
	return CheckHash(this.Body.ReplyHash.B58String())
}

/*
	检查该消息是否是自己的
	不是自己的则自动转发出去
*/
func (this *Message) Reply(msgId uint64) bool {
	this.BuildReplyHash(this.Body.CreateTime, this.Body.Hash, this.Body.SendRand)
	//TODO 这里对消息加密

	if nodeStore.NodeSelf.IsSuper {
		return IsSendToOtherSuperToo(this.Head, this.Body.JSON(), msgId, nil)
	} else {
		if nodeStore.SuperPeerId == nil {
			fmt.Println("没有可用的超级节点")
			return true
		}
		if session, ok := engine.GetSession(nodeStore.SuperPeerId.B58String()); ok {
			session.Send(msgId, this.Head.JSON(), this.Body.JSON(), false)
		} else {
			fmt.Println("超级节点的session未找到")
		}
		return true
	}
}

func NewMessage(head *MessageHead, body *MessageBody) *Message {
	return &Message{
		Head: head,
		Body: body,
	}
}

func ParserMessage(data, dataplus *[]byte, msgId uint64) (*Message, error) {
	head := new(MessageHead)
	err := json.Unmarshal(*data, head)
	if err != nil {
		return nil, err
	}

	msg := Message{
		msgid:    msgId,
		Head:     head,
		DataPlus: dataplus,
	}
	return &msg, nil
}

/*
	得到一条消息的hash值
*/
//func GetHash(msg *Message) string {
//	hash := sha256.New()
//	hash.Write([]byte(msg.RecvId))
//	//	binary.Write(hash, binary.BigEndian, uint64(msg.ProtoId))
//	binary.Write(hash, binary.BigEndian, msg.CreateTime)
//	// hash.Write([]byte(int64(msg.ProtoId)))
//	// hash.Write([]byte(msg.CreateTime))
//	hash.Write([]byte(msg.Sender))
//	// hash.Write([]byte(msg.RecvTime))
//	binary.Write(hash, binary.BigEndian, msg.ReplyTime)
//	hash.Write(msg.Content)
//	hash.Write([]byte(msg.ReplyHash))
//	return hex.EncodeToString(hash.Sum(nil))
//}

/*
	消息超时删除md5
*/
func msgTimeOutProsess(class, params string) {
	switch class {
	case config.TSK_msg_timeout_remove: //删除超时的消息md5
		//		fmt.Println("开始删除临时域名", tempName)
		//		tempNameLock.Lock()
		//		delete(tempName, params)
		//		tempNameLock.Unlock()
		//		fmt.Println("删除了这个临时域名", params, tempName)
	default:
		//		//剩下是需要更新的域名
		//		flashName := FlashName{
		//			Name:  params,
		//			Class: class,
		//		}
		//		OutFlashName <- &flashName
	}

}
