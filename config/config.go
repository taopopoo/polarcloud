package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"yunpan/core/config"
	"yunpan/core/engine"
	"yunpan/core/utils"
)

const (
	Path_configDir     = "conf"            //配置文件存放目录
	Path_config        = "config.json"     //配置文件名称
	Core_addr_prk      = "addr_ec_prk.pem" //地址私钥文件名称
	Core_addr_puk      = "addr_ec_puk.pem" //地址公钥文件名称
	Core_addr_prk_type = "EC PRIVATE KEY"  //地址私钥文件抬头
	Core_addr_puk_type = "EC PUBLIC KEY"   //地址公钥文件抬头
)

const (
	Name_prk = "name_ec_prk.pem" //地址私钥文件名称
	Name_puk = "name_ec_puk.pem" //地址公钥文件名称
)

const (
	Store_path_dir            = "store" //本地共享文件存储目录名称
	Store_path_fileinfo_self  = "self"  //自己上传的文件索引存储目录名称
	Store_path_fileinfo_local = "local" //本地下载过的文件索引存储目录名称
	Store_path_fileinfo_net   = "net"   //网络需要保存的文件索引存储目录名称
	Store_path_fileinfo_cache = "cache" //缓存中保存的文件索引存储目录名称
	Store_path_temp           = "temp"  //临时文件夹，本地上传存放目录，存放未切片的完整文件
	IsRemoveStore             = false   //启动时删除本地所有文件分片及分片索引
	//	IsCreateId                = true    //启动时是否要创建新的id

	HashCode    = utils.SHA3_256 //
	NodeIDLevel = 256            //节点id比特位数
)

var (
	WebAddr                = "" //
	WebPort         uint16 = 0  //本地监听端口
	Web_path_static        = "" //网页静态文件路径
	Web_path_views         = "" //网页模板文件路径
)

var (
	Store_dir            string = filepath.Join(Store_path_dir)                            //本地共享文件存储目录路径
	Store_fileinfo_self  string = filepath.Join(Store_path_dir, Store_path_fileinfo_self)  //自己上传的文件索引存储目录路径
	Store_fileinfo_local string = filepath.Join(Store_path_dir, Store_path_fileinfo_local) //本地下载过的文件索引存储目录路径
	Store_fileinfo_net   string = filepath.Join(Store_path_dir, Store_path_fileinfo_net)   //网络需要保存的文件索引存储目录路径
	Store_fileinfo_cache string = filepath.Join(Store_path_dir, Store_path_fileinfo_cache) //缓存中保存的文件索引存储目录路径
	Store_temp           string = filepath.Join(Store_path_dir, Store_path_temp)           //临时文件夹，本地上传存放目录，存放未切片的完整文件
)

func init() {

	ok, err := utils.PathExists(filepath.Join(Path_configDir, Path_config))
	if err != nil {
		panic("检查配置文件错误：" + err.Error())
		return
	}
	if !ok {
		cfi := new(Config)
		cfi.Port = 9981
		bs, _ := json.Marshal(cfi)

		f, err := os.OpenFile(filepath.Join(Path_configDir, Path_config), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic("创建配置文件错误：" + err.Error())
			return
		}
		_, err = f.Write(bs)
		if err != nil {
			panic("写入配置文件错误：" + err.Error())
			return
		}
		f.Close()
	}

	bs, err := ioutil.ReadFile(filepath.Join(Path_configDir, Path_config))
	if err != nil {
		panic("读取配置文件错误：" + err.Error())
		return
	}
	cfi := new(Config)
	err = json.Unmarshal(bs, cfi)
	if err != nil {
		panic("解析配置文件错误：" + err.Error())
		return
	}
	config.Init_LocalPort = cfi.Port
	config.Init_GatewayPort = cfi.Port
	Web_path_static = cfi.WebStatic
	Web_path_views = cfi.WebViews
	engine.Netid = cfi.Netid
	WebAddr = cfi.WebAddr
	WebPort = cfi.WebPort
	Miner = cfi.Miner
}

type Config struct {
	Netid       uint32 `json:"netid"`     //
	IP          string `json:"ip"`        //ip地址
	Port        uint16 `json:"port"`      //监听端口
	WebAddr     string `json:"WebAddr"`   //
	WebPort     uint16 `json:"WebPort"`   //
	WebStatic   string `json:"WebStatic"` //
	WebViews    string `json:"WebViews"`  //
	RpcServer   bool
	RpcUser     string
	RpcPassword string
	RpcPort     int
	RpcAllowip  string
	Miner       bool `json:"miner"` //本节点是否是矿工
}
