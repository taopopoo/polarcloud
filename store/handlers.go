package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	gconfig "yunpan/config"
	"yunpan/core/engine"
	mc "yunpan/core/message_center"
	"yunpan/core/nodeStore"
	"yunpan/core/utils"
)

/*
	收到共享文件消息
*/
func AddFileShare(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("收到共享文件消息")

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
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

	fi, err := ParseFileinfo(*message.Body.Content)
	if err != nil {
		fmt.Println(err)
	}

	//	fmt.Println("本节点保存文件索引", string(fi.JSON()))

	//判断本地网络是否存在文件，若不存在则添加
	filocal := FindFileinfoToNet(fi.Hash.B58String())
	if filocal == nil {
		//添加文件
		err = AddFileinfoToNet(fi, true)
		if err != nil {
			fmt.Println(err)
		}
		//		fmt.Println("文件索引保存到本地")
	} else {
		//		fmt.Println("本地有文件索引")
		//文件中添加共享用户
		for _, v := range fi.FileChunk.GetAll() {
			one := v.(*FileChunk)
			filocal.AddShareUser(one.No, message.Head.Sender)
		}
		//		for _, one := range fi.FileChunk {
		//			filocal.AddShareUser(one.No, message.Head.Sender)
		//		}
	}
	//回复给发送者
	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	mbody := mc.NewMessageBody(nil, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(MSGID_addFileShare_recv)
	//自动下载文件到本地
	/*go func() {
		err = DownloadFileOpt(fi)
		if err != nil {
			fmt.Println("自动下载文件失败", err)
			return
		}
		AddFileinfoToLocal(fi, true)
		fmt.Println("自动下载文件成功")
	}()*/
	//	fmt.Println("返回消息自己处理", string(*message.Body.JSON()))
}

/*
	收到共享文件消息 返回
*/
func AddFileShare_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("收到共享文件消息 返回")

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
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
	//	fmt.Println("===", string(msg.Data), "\n", string(*message.Body.JSON()))

	//	message := new(mc.Message)
	//	err := json.Unmarshal(msg.Data, message)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	//	form, _ := hex.DecodeString(msg.Session.GetName())

	//	mh, _ := utils.FromB58String(msg.Session.GetName())

	//	if ok := mc.IsSendToOtherSuper(message, msg.MsgID, &mh); ok {
	//		fmt.Println("发给其他小伙伴了")
	//		return
	//	}
	//	fmt.Println("是本节点的")
	mc.ResponseWait(mc.CLASS_sharefile, message.Body.Hash.B58String(), &[]byte{})
}

/*
	收到查询文件信息消息
*/
func FindFileinfoHandler(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("收到查询文件信息消息")

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println("解析查询索引文件消息错误：", err)
		return
	}
	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}
	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		fmt.Println("解析消息内容错误", err)
		return
	}
	var hashid *utils.Multihash
	if message.Body.Content != nil {
		umul := utils.Multihash(*message.Body.Content)
		hashid = &umul
	} else {
		hashid = message.Head.RecvId
	}
	var bs []byte
	//fileinfo := FindFileinfoToNet(message.Head.RecvId.B58String())
	fileinfo := FindFileinfoToNet(hashid.B58String())
	if fileinfo != nil {
		bs = fileinfo.JSON()
		fmt.Println("查询到了文件", string(bs))
	} else {
		fmt.Println("没有找到文件索引")
	}
	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, false)
	mbody := mc.NewMessageBody(&bs, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(MSGID_findFileinfo_recv)

}

/*
	收到查询文件索引 返回
*/
func FindFileinfo_recv(c engine.Controller, msg engine.Packet) {
	fmt.Println("收到查询文件索引 返回")

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		return
	}
	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}
	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		//				fmt.Println("---2", err)
		//		fmt.Println(string(msg.Dataplus))
		//		engine.Log.Debug("%s", err.Error())
		//		engine.Log.Debug("%s", string(msg.Dataplus))
		//		engine.NLog.Error(engine.LOG_file, "%s", err.Error())
		//		engine.NLog.Error(engine.LOG_file, "%s", string(msg.Dataplus))
		return
	}

	//	message := new(mc.Message)
	//	err := json.Unmarshal(msg.Data, message)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	//	form, _ := hex.DecodeString(msg.Session.GetName())

	//	mh, _ := utils.FromB58String(msg.Session.GetName())

	//	if ok := mc.IsSendToOtherSuper(message, msg.MsgID, &mh); ok {
	//		fmt.Println("发给其他小伙伴了")
	//		return
	//	}

	mc.ResponseWait(mc.CLASS_findfileinfo, message.Body.Hash.B58String(), message.Body.Content)

}

/*
	收到查询文件长度
*/
func FindFilesize(c engine.Controller, msg engine.Packet) {

}

/*
	收到查询文件长度 返回
*/
func FindFilesize_recv(c engine.Controller, msg engine.Packet) {

}

type FileChunkVO struct {
	FileHash      *utils.Multihash //完整文件hash
	No            uint64           //文件块编号
	ChunkHash     *utils.Multihash //块 hash
	Index         uint64           //下载块起始位置
	Length        uint64           //下载块长度
	Content       []byte           //数据块内容
	ContentLength uint64           //数据块总大小
}

func (this *FileChunkVO) JSON() []byte {
	bs, _ := json.Marshal(this)
	return bs
}
func ParseFileChunkVO(bs []byte) *FileChunkVO {
	fcvo := new(FileChunkVO)
	if json.Unmarshal(bs, fcvo) != nil {
		return nil
	}
	return fcvo
}

/*
	收到下载文件块
*/
func DownloadFilechunk(c engine.Controller, msg engine.Packet) {
	//fmt.Println("收到下载文件块")
	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
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

	filechunk := ParseFileChunkVO(*message.Body.Content)

	var resultErrorMsgFun = func() {
		//给发送者返回错误消息
		mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
		mbody := mc.NewMessageBody(nil, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
		message = mc.NewMessage(mhead, mbody)
		if message.Reply(MSGID_downloadFileChunk_recv) {
			return
		}

	}
	bs, err := ioutil.ReadFile(filepath.Join(gconfig.Store_dir, filechunk.ChunkHash.B58String()))
	//start
	datalength := uint64(len(bs))
	if filechunk.Index > datalength {
		resultErrorMsgFun()
		return
	}
	var length uint64
	if filechunk.Length == uint64(0) || filechunk.Length > datalength {
		length = datalength
	} else {
		length = filechunk.Length
	}
	bs = bs[filechunk.Index:length]
	if err != nil {
		fmt.Println(err)
		resultErrorMsgFun()
		return
	}
	filechunk.Content = bs
	filechunk.ContentLength = datalength
	fmt.Println("**********收到块下载信息********")
	fmt.Println("块", filechunk.ChunkHash.B58String())
	fmt.Println("-------- 从这里下载的文件块 -------")
	fmt.Println(filechunk.Index, length)
	fmt.Println("发送给", message.Head.Sender.B58String())
	fmt.Println("预计发送大小", len(bs))
	fmt.Println("*********end***********")
	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	content := filechunk.JSON()
	mbody := mc.NewMessageBody(&content, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(MSGID_downloadFileChunk_recv)
}

/*
	收到下载文件块 返回
*/
func DownloadFilechunk_recv(c engine.Controller, msg engine.Packet) {
	//	fmt.Println("收到下载文件块 返回", string(msg.Data))
	fmt.Println("收到下载文件块 返回", len(msg.Dataplus))

	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println("error  1", err)
		return
	}
	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}
	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		//		fmt.Println("error  2", err)
		//		fmt.Println(string(msg.Dataplus) + "end")
		engine.NLog.Error(engine.LOG_file, "%s", err.Error())
		engine.NLog.Error(engine.LOG_file, "%s", string(msg.Dataplus))
		return
	}

	//	message := new(mc.Message)
	//	err := json.Unmarshal(msg.Data, message)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	//	fmt.Println("111查看是否被改变", string(*message.JSON()))
	//	//	form, _ := hex.DecodeString(msg.Session.GetName())

	//	form, _ := utils.FromB58String(msg.Session.GetName())

	//	if ok := mc.IsSendToOtherSuper(message, msg.MsgID, &form); ok {
	//		fmt.Println("发给其他小伙伴了")
	//		return
	//	}
	//	fmt.Println("222查看是否被改变", string(*message.JSON()))

	fmt.Println("返回的文件块内容大小", len(*message.Body.Content))

	mc.ResponseWait(mc.CLASS_downloadfile, message.Body.Hash.B58String(), message.Body.Content)

}

//上传地址信息
type UpInfo struct {
	Scheme string
	Ip     string
	Port   uint16
	Path   string
	Field  string
}

func (u *UpInfo) Json() []byte {
	res, err := json.Marshal(u)
	if err != nil {
		fmt.Println("upinfo marshal:", err)
		return nil
	}

	return res
}

//获取上传地址信息
func Uploadinfo(c engine.Controller, msg engine.Packet) {
	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
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
	upinfo := UpInfo{}
	upinfo.Scheme = UploadScheme
	upinfo.Ip = nodeStore.NodeSelf.Addr
	upinfo.Port = gconfig.WebPort
	upinfo.Path = UploadPath
	upinfo.Field = UploadField
	content := upinfo.Json()
	//回复给发送者
	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	mbody := mc.NewMessageBody(&content, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(MSGID_getUploadinfo_recv)
}

//获取上传地址信息 返回
func Uploadinfo_recv(c engine.Controller, msg engine.Packet) {
	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println("error  1", err)
		return
	}
	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}
	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		engine.NLog.Error(engine.LOG_file, "%s", err.Error())
		engine.NLog.Error(engine.LOG_file, "%s", string(msg.Dataplus))
		return
	}
	mc.ResponseWait(mc.CLASS_uploadinfo, message.Body.Hash.B58String(), message.Body.Content)
}

//根据文件hash获取1/4节点地址信息（app用）
func GetfourNodeinfo(c engine.Controller, msg engine.Packet) {
	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
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
	var idstr []string
	ids := getQuarterLogicIds(message.Head.RecvId)
	for _, v := range ids {
		idstr = append(idstr, v.B58String())
	}
	content, err := json.Marshal(idstr)
	if err != nil {
		fmt.Println(err)
		return
	}
	//回复给发送者
	mhead := mc.NewMessageHead(message.Head.Sender, message.Head.SenderSuperId, true)
	mbody := mc.NewMessageBody(&content, message.Body.CreateTime, message.Body.Hash, message.Body.SendRand)
	message = mc.NewMessage(mhead, mbody)
	message.Reply(MSGID_getfourNodeinfo_recv)
}

//根据文件hash获取1/4节点地址信息（app用）
func GetfourNodeinfo_recv(c engine.Controller, msg engine.Packet) {
	message, err := mc.ParserMessage(&msg.Data, &msg.Dataplus, msg.MsgID)
	if err != nil {
		fmt.Println("error  1", err)
		return
	}
	form, _ := utils.FromB58String(msg.Session.GetName())
	if message.IsSendOther(&form) {
		return
	}
	//发送给自己的，自己处理
	if err := message.ParserContent(); err != nil {
		engine.NLog.Error(engine.LOG_file, "%s", err.Error())
		engine.NLog.Error(engine.LOG_file, "%s", string(msg.Dataplus))
		return
	}
	mc.ResponseWait(mc.CLASS_getfourNodeinfo, message.Body.Hash.B58String(), message.Body.Content)
}
