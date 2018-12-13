package store

const (
	Time_sharefile             = 120                   //共享文件添加索引间隔时间 单位：秒
	Time_shareUserOfflineClear = Time_sharefile*15 + 1 //共享的用户超过间隔时间（Time_sharefile）3倍后删除这个共享用户
	Time_loopClearUser         = 60 * 60 * 24          //定时清理文件索引，文件索引中超过60天没有用户共享的块删除掉

	Chunk_size   = 1024 * 1024 * 8  //1024 * 1024 * 8 //文件分块大小，默认8M
	UploadScheme = "http"           //文件上传协议
	UploadPath   = "/store/addfile" //文件HTTP上传地址
	UploadField  = "files[]"        //文件HTTP上传表单名
)
