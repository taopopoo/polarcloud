package store

import (
	"errors"
	"fmt"
	"sync"
	//	gconfig "polarcloud/config"
	"path/filepath"
	gconfig "polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	//	"github.com/mr-tron/base58/base58"
)

/*
	共享本节点的所有文件块索引
	把文件索引上传到网络中去，并且增加本节点共享
*/
func UpNetFileinfo(fi *FileInfo) error {
	//	fi.FileChunk.Range(func(i int, v interface{}) bool {
	//		one := v.(*FileChunk)
	//		one.AddShareUser(one.No, nodeStore.NodeSelf.IdInfo.Id)
	//		one.AddUpdateUser()
	//		return true
	//	})
	//	fmt.Println("222222222222222222")

	for _, v := range fi.FileChunk.GetAll() {
		one := v.(*FileChunk)
		fi.AddShareUser(one.No, nodeStore.NodeSelf.IdInfo.Id)
	}
	recvId := fi.Hash
	mhead := mc.NewMessageHead(recvId, recvId, false)
	content := fi.JSON()
	mbody := mc.NewMessageBody(&content, "", nil, 0)
	message := mc.NewMessage(mhead, mbody)
	if message.Send(MSGID_addFileShare) {
		//		fmt.Println("发给其他小伙伴了----")
		bs := mc.WaitRequest(mc.CLASS_sharefile, message.Body.Hash.B58String())
		//		fmt.Println("有消息返回了啊")
		if bs == nil {
			fmt.Println("发送共享文件消息失败，可能超时")
			return errors.New("发送共享文件消息失败，可能超时")
		}
		//		fmt.Println("添加文件共享成功")
		return nil
	}
	//这个文件索引归自己管理
	AddFileinfoToNet(fi, true)
	//同步文件到最近的邻居节点
	//go SyncFiletoNearId(fi)
	//	fmt.Println("===== 9999999999999")
	return nil
}

/*
	网络中查找一个文件信息
*/
func FindFileinfoOpt(hash string) (fi *FileInfo, err error) {
	err = ErrNotFindCode
	mh, errs := utils.FromB58String(hash)
	if errs != nil {
		err = errs
		return
	}
	ids := getQuarterLogicIds(&mh)
	for _, id := range ids {
		content := []byte(mh)
		mhead := mc.NewMessageHead(id, id, false)
		mbody := mc.NewMessageBody(&content, "", nil, 0)
		message := mc.NewMessage(mhead, mbody)
		if message.Send(MSGID_findFileinfo) {
			//		fmt.Println("开始等待查找返回")
			bs := mc.WaitRequest(mc.CLASS_findfileinfo, message.Body.Hash.B58String())
			//		fmt.Println("等待查找已经返回", string(*bs))
			if bs != nil {
				fi, err = ParseFileinfo(*bs)
				return
			}

		}
	}
	//	fmt.Println("这里直接就返回错误了")
	return

}

/*
	网络中下载一个文件到本地
*/
func DownloadFileOpt(fileinfo *FileInfo) error {
	//如果缓存文件已存在，则直接返回
	ok, errs := utils.PathExists(filepath.Join(gconfig.Store_temp, fileinfo.Name))
	if errs != nil {
		return errs
	}
	if ok {
		return nil
	}
	file := NewFile(fileinfo)

	group := new(sync.WaitGroup)

	//分块下载
	for _, v := range fileinfo.FileChunk.GetAll() {
		one := v.(*FileChunk)
		group.Add(1)
		go func(one *FileChunk) {
			if err := DownloadFilechunkToLocal(fileinfo, one.No); err == nil {
				//				fmt.Println("下载文件成功了")
				//文件块下载成功
				fc := NewFileChunk(one.No, one.Hash)
				file.AddFileChunk(fc)

			} else {
				engine.Log.Warn("下载文件块错误 %s", err.Error())
			}
			group.Done()
		}(one)
	}
	group.Wait()
	if !file.Check() {
		//		fmt.Println("文件分片下载失败")
		return errors.New("文件分片下载失败")
	}
	err := file.Assemble()
	if err != nil {
		fmt.Println("组装文件失败", err)
		return err
	}
	return nil
}

//同步文件索引到邻居节点
func SyncFiletoNearId(fi *FileInfo) error {
	recvId := nodeStore.FindNearInSuper(nodeStore.NodeSelf.IdInfo.Id, nil, false)
	if recvId == nil {
		return errors.New("没有附近节点")
	}
	//加入自己为共享用户
	for _, v := range fi.FileChunk.GetAll() {
		one := v.(*FileChunk)
		fi.AddShareUser(one.No, recvId)
	}
	mhead := mc.NewMessageHead(recvId, recvId, false)
	content := fi.JSON()
	mbody := mc.NewMessageBody(&content, "", nil, 0)
	message := mc.NewMessage(mhead, mbody)
	if message.Send(MSGID_addFileShare) {
		bs := mc.WaitRequest(mc.CLASS_sharefile, message.Body.Hash.B58String())
		if bs == nil {
			fmt.Println("发送共享文件消息失败，可能超时")
			return errors.New("发送共享文件消息失败，可能超时")
		}
		return nil
	}
	return nil
}
