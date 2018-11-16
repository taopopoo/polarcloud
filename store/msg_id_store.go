package store

import (
	"fmt"
	"os"
	"path/filepath"
	gconfig "polarcloud/config"
	"polarcloud/core/engine"
	"polarcloud/core/utils"
)

const (
	MSGID_addFileShare           = 1000 //添加一个文件共享
	MSGID_addFileShare_recv      = 1001 //添加一个文件共享 返回
	MSGID_findFileinfo           = 1002 //网络中查找一个文件信息
	MSGID_findFileinfo_recv      = 1003 //网络中查找一个文件信息 返回
	MSGID_getFilesize            = 1004 //网络中查找一个文件信息
	MSGID_getFilesize_recv       = 1005 //网络中查找一个文件信息 返回
	MSGID_downloadFileChunk      = 1006 //网络中下载文件块
	MSGID_downloadFileChunk_recv = 1007 //网络中下载文件块 返回
	MSGID_getUploadinfo          = 1008 //返回节点上传地址信息
	MSGID_getUploadinfo_recv     = 1009 //返回节点上传地址信息 返回
	MSGID_syncFileInfo           = 1010 //返回同步文件信息到1/4节点
	MSGID_syncFileInfo_recv      = 1012 //返回同步文件信息到1/4节点
	MSGID_getfourNodeinfo        = 1013 //返回节点上传地址信息
	MSGID_getfourNodeinfo_recv   = 1014 //返回节点上传地址信息 返回
)

func RegsterStore() error {
	//删除本地所有文件分片及分片索引
	if gconfig.IsRemoveStore {
		err := os.RemoveAll(gconfig.Store_dir)
		if err != nil {
			return err
		}
	}

	//创建保存文件的文件夹
	utils.CheckCreateDir(gconfig.Store_dir)
	//创建保存文件索引的文件夹
	utils.CheckCreateDir(filepath.Join(gconfig.Store_fileinfo_self))
	//创建保存文件索引的文件夹
	utils.CheckCreateDir(gconfig.Store_fileinfo_local)
	//创建保存文件索引的文件夹
	utils.CheckCreateDir(gconfig.Store_fileinfo_net)
	//创建保存文件索引的文件夹
	utils.CheckCreateDir(gconfig.Store_fileinfo_cache)
	//创建临时文件夹
	utils.CheckCreateDir(gconfig.Store_temp)

	initTask()

	//加载自己共享的文件
	err := LoadFileInfoSelf()
	if err != nil {
		return err
	}
	//加载本地文件索引
	err = LoadFileInfoLocal()
	if err != nil {
		return err
	}
	//加载网络文件索引
	err = LoadFileInfoNet()
	if err != nil {
		return err
	}
	engine.RegisterMsg(MSGID_addFileShare, AddFileShare)
	engine.RegisterMsg(MSGID_addFileShare_recv, AddFileShare_recv)
	engine.RegisterMsg(MSGID_findFileinfo, FindFileinfoHandler)
	engine.RegisterMsg(MSGID_findFileinfo_recv, FindFileinfo_recv)
	engine.RegisterMsg(MSGID_getFilesize, FindFilesize)
	engine.RegisterMsg(MSGID_getFilesize_recv, FindFilesize_recv)
	engine.RegisterMsg(MSGID_downloadFileChunk, DownloadFilechunk)
	engine.RegisterMsg(MSGID_downloadFileChunk_recv, DownloadFilechunk_recv)
	engine.RegisterMsg(MSGID_getUploadinfo, Uploadinfo)
	engine.RegisterMsg(MSGID_getUploadinfo_recv, Uploadinfo_recv)
	engine.RegisterMsg(MSGID_syncFileInfo, syncFileInfo)
	engine.RegisterMsg(MSGID_syncFileInfo_recv, syncFileInfo_recv)
	engine.RegisterMsg(MSGID_getfourNodeinfo, GetfourNodeinfo)
	engine.RegisterMsg(MSGID_getfourNodeinfo_recv, GetfourNodeinfo_recv)
	return nil
}

/*
	判断一个路径的文件是否存在
*/
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func Mkdir(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	//	err := os.Mkdir(path, os.ModeDir)
	if err != nil {
		fmt.Println("创建文件夹失败", path, err)
		return err
	}
	return nil
}
