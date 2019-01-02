/*
	数据库中key保存的格式
	1.保存第一个区块的hash:startblock
	2.保存区块头:[区块hash]
	3.交易历史纪录:1_[1/2]_[自己的地址]_[目标地址]
*/
package config

const (
	block_start_str = "startblock" //保存第一个区块hash
	History         = "1_"         //数据库中key以“1_”开头的数据都是交易历史数据
	//	BlockHeight     = "2_"         //数据库中key以“2_”开头的数据保存的是块高度和区块hash对应

//	In  = "1_"
//	Out = "2_"
)

var (
	Key_block_start = []byte(block_start_str) //保存第一个区块hash
)

///*
//	构建交易历史转入key
//*/
//func BuildHistoryInKey(self, tag string) []byte {
//	return []byte(History + In + self + "_" + tag)
//}

///*
//	构建交易历史转出key
//*/
//func BuildHistoryOutKey(self, tag string) []byte {
//	return []byte(History + Out + self + "_" + tag)
//}
