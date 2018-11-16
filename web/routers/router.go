package routers

import (
	//	"polarcloud/core/cache_store"

	"polarcloud/wallet/rpc"
	"polarcloud/web/controllers/store"
	"polarcloud/web/controllers/wallet"

	"github.com/astaxie/beego"
)

func Start() {

	// beego.Router("/", &controllers.MainController{})
	// beego.Router("/self/msg", &controllers.MsgController{}, "get:MsgPage")           //打开消息页面
	// beego.Router("/self/sendtextmsg", &controllers.MainController{}, "post:SendMeg") //给节点发送文本消息
	// beego.Router("/self/msg/getmsg", &controllers.MsgController{}, "post:GetMsg")    //轮询获取消息
	// beego.Router("/self/friend/add", &controllers.MsgController{}, "post:AddFriend") //添加一个好友

	// beego.Router("/self/test", &controllers.MainController{}, "get:Test") //
	// //	beego.Router("/self/applyname", &controllers.MainController{}, "post:ApplyName")         //申请一个域名
	// //	beego.Router("/self/sendmsgtoname", &controllers.MainController{}, "post:SendMegToName") //给一个域名发送消息

	// beego.Router("/self/bttest", &controllers.MainController{}, "post:BtTest") //

	// //	beego.Router("/*", &controllers.MainController{}, "*:Agent")          //
	// //	beego.Router("/:urls", &controllers.MainController{}, "get:AgentToo") //

	//
	//云存储部分
	beego.Router("/", &store.Index{}, "get:Index")                 //云存储首页
	beego.Router("/store/getlist", &store.Index{}, "get:GetList")  //获取文件列表
	beego.Router("/store/addfile", &store.Index{}, "post:AddFile") //添加一个文件
	beego.Router("/store/:hash", &store.Index{}, "get:GetFile")    //获取一个文件

}

func RegisterWallet() {
	beego.Router("/self/wallet", &wallet.Index{}, "get:Index") //钱包首页
}
func RegisterRpc() {
	beego.Router("/rpc", &rpc.Bind{}, "post:Index") //rpc调用
}

//func RegisterAnonymousNet() {
//	beego.Router("/*", &anonymousnet.MainController{}, "*:Agent")          //
//	beego.Router("/:urls", &anonymousnet.MainController{}, "get:AgentToo") //
//}
