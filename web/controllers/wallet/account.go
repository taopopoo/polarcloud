package wallet

import (
	//	"polarcloud/config"
	"polarcloud/wallet"

	"github.com/astaxie/beego"
)

type Account struct {
	beego.Controller
}

func (this *Account) GetInfo() {
	//	names, _ := store.GetFileinfoToSelfAll()

	//	fmt.Println("网络文件个数为", len(names))
	//	this.Data["Names"] = names

	this.Data["CheckKey"] = wallet.CheckKey()

	this.TplName = "wallet/index.tpl"
}
