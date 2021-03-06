/*
	竞选为矿工
*/
package mining

/*
	参加选举
	1.先广播自己的账户地址，为自己拉票。
	2.收到广播的节点，发送自己的选票，选票要签名，发送给指定矿工。(考虑一个时间段只能给一个人投票)
	3.矿工和预备矿工保存票数，最终记录到区块链上。
	注意：1.参与竞选的矿工足够多时，最近7组已经出块的矿工不能参与竞选。
	     2.维护一个恶意节点库，屏蔽恶意节点的竞选消息。
*/

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	//	"sync"
	"time"
	"polarcloud/config"
	"polarcloud/core/engine"
	mc "polarcloud/core/message_center"
	"polarcloud/core/nodeStore"
	"polarcloud/core/utils"
	"polarcloud/wallet/keystore"
)

////参与挖矿竟票计数器
////var voteNumber = new(sync.Map) //key:string=矿工账户地址；value:uint64=票数；

//var electionMap = new(sync.Map) //key:uint64=组高度；value=（key:string=矿工地址;value:map=map[id]票数;）

///*
//	定时删除过时的竟票计数器
//*/
//func init() {
//	groupHeightNew := uint64(0)
//	//TODO 定时删除过时的投票改为事件驱动删除方式
//	electionMap.Range(func(key, value interface{}) bool {
//		groupHeight := key.(uint64)
//		if groupHeightNew >= groupHeight {
//			electionMap.Delete(key)
//		}
//		return true
//	})
//}

///*
//	添加一个选举
//*/
////func AddElection(ele *Election) {
////	value, ok := electionMap.Load(ele.GroupHeight)
////	if ok {
////		v := value.(*sync.Map)
////		v.Store()
////	}

////	key := ele.Addr + "_" + strconv.Itoa(int(ele.Time))
////	if _, ok := electionMap.Load(key); ok {
////		return
////	}
////	electionMap.Store(key, new(sync.Map))
////}

///*
//	增加一个选票
//*/
//func AddBallotTicket(bt *BallotTicket) {
//	//TODO 异步问题
//	value, ok := electionMap.Load(bt.GroupHeight)
//	if ok {
//		v := value.(*sync.Map)
//		value, ok = v.Load(bt.Miner.B58String())
//		if ok {
//			addrs := value.(*sync.Map)
//			addrs.LoadOrStore(bt.Addr.B58String(), bt)
//		} else {
//			addrs := new(sync.Map)
//			addrs.Store(bt.Addr.B58String(), bt)
//			v.Store(bt.Miner.B58String(), addrs)
//		}
//	} else {
//		addrs := new(sync.Map)
//		addrs.Store(bt.Addr.B58String(), bt)
//		miners := new(sync.Map)
//		miners.Store(bt.Miner.B58String(), addrs)
//		electionMap.Store(bt.GroupHeight, miners)
//	}
//}

///*
//	查询矿工票数
//*/
//func FindTotal(gheight uint64) map[string]uint64 {
//	total := make(map[string]uint64)
//	value, ok := electionMap.Load(gheight)
//	if ok {
//		gheight := value.(*sync.Map)
//		gheight.Range(func(key, value interface{}) bool {
//			//			fmt.Println("++++++++height", key, value)
//			miner := key.(string)
//			count := uint64(0)
//			addrs := value.(*sync.Map)
//			addrs.Range(func(key, value interface{}) bool {
//				count = count + 1
//				return true
//			})
//			//			fmt.Println("=-=-=", miner, count)
//			total[miner] = count
//			return true
//		})
//	}
//	return total
//}

/*
	竞选，拉票消息
*/
type Election struct {
	GroupHeight uint64           `json:"groupheight"` //区块组高度
	Addr        *utils.Multihash `json:"addr"`        //矿工地址
	Time        int64            `json:"time"`        //竞选时间
}

func (this *Election) JSON() *[]byte {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil
	}
	return &bs
}

func NewElection(addr *utils.Multihash) *Election {
	return &Election{
		Addr: addr,              //矿工地址
		Time: time.Now().Unix(), //竞选时间
	}
}

func ParseElection(bs *[]byte) *Election {
	ec := new(Election)
	err := json.Unmarshal(*bs, ec)
	if err != nil {
		return nil
	}
	return ec
}

/*
	选票
*/
type BallotTicket struct {
	Addr    *utils.Multihash `json:"addr"`    //投票者地址
	Puk     []byte           `json:"puk"`     //投票者公钥
	Sign    []byte           `json:"sign"`    //签名
	Witness *utils.Multihash `json:"witness"` //见证人地址
	Deposit []byte           `json:"deposit"` //见证人押金交易id
	//	Time    int64            `json:"time"`    //时间
	//	GroupHeight uint64           `json:"groupheight"` //矿工组高度
}

func (this *BallotTicket) Json() *[]byte {
	bs, err := json.Marshal(this)
	if err != nil {
		return nil
	}
	return &bs
}

func ParseBallotTicket(bs *[]byte) *BallotTicket {
	ec := new(BallotTicket)
	err := json.Unmarshal(*bs, ec)
	if err != nil {
		return nil
	}
	return ec
}

/*
	创建一个投票并广播出去
	@height    uint64    要投票的备用见证人组高度
	@addr      *utils.Multihash    见证人地址
	@key       *keystore.Address   投票者的key
*/
func MulticastBallotTicket(deposit *[]byte, addr *utils.Multihash) {
	//	fmt.Println("=============给自己投票", addr.B58String(), hex.EncodeToString(*deposit))
	key, err := keystore.GetCoinbase()
	if err != nil {
		fmt.Println("111获取矿工地址失败", err)
		return
	}

	bt := BallotTicket{
		Addr:    key.Hash,          //投票者地址
		Puk:     key.GetPubKey(),   //投票者公钥
		Sign:    []byte("先不做签名验证"), //签名
		Witness: addr,              //见证人地址
		Deposit: *deposit,          //见证人押金交易id
	}

	AddBallotTicket(&bt)

	head := mc.NewMessageHead(nil, nil, false)
	body := mc.NewMessageBody(bt.Json(), "", nil, 0)
	message := mc.NewMessage(head, body)
	message.BuildHash()

	//继续广播给其他节点
	if nodeStore.NodeSelf.IsSuper {
		//广播给其他超级节点
		//		mh := utils.Multihash(*message.Body.Content)
		ids := nodeStore.GetIdsForFar(message.Head.SenderSuperId)
		for _, one := range ids {
			//			log.Println("发送给", one.B58String())
			if ss, ok := engine.GetSession(one.B58String()); ok {
				ss.Send(config.MSGID_multicast_vote_recv, head.JSON(), body.JSON(), false)
			}
		}

		//广播给代理对象
		pids := nodeStore.GetProxyAll()
		for _, one := range pids {
			if ss, ok := engine.GetSession(one); ok {
				//				ss.Send(MSGID_multicast_online_recv, &msg.Data, false)
				ss.Send(config.MSGID_multicast_vote_recv, head.JSON(), body.JSON(), false)
			}
		}

	}
}

/*
	增加一个选票
*/
func AddBallotTicket(bt *BallotTicket) {
	//	fmt.Println("增加选票到交易", hex.EncodeToString(bt.Deposit))
	//	bhBs, err := db.Find(bt.Deposit)
	//	if err != nil {
	//		//		fmt.Println("11111", err)
	//		return
	//	}
	//	txItr, err := ParseTxBase(bhBs)
	//	if err != nil {
	//		return
	//	}
	//	base := txItr.(*Tx_deposit_in)
	//	//	fmt.Println("选票区块id", hex.EncodeToString(base.BlockHash))
	//	bhBs, err = db.Find(base.BlockHash)
	//	if err != nil {
	//		return
	//	}

	//	fmt.Println("找到的块", string(*bhBs))
	//	bh, err := ParseBlockHead(bhBs)
	//	if err != nil {
	//		//		fmt.Println("22222", err)
	//		return
	//	}
	//	fmt.Println("应该添加投票到区块", bh.Height, hex.EncodeToString(bh.Hash))
	//	chain.PrintBlockList()

	witness := chain.witnessChain.GetBackupWitness()
	for {
		if witness == nil {
			break
		}
		if hex.EncodeToString(witness.DepositId) == hex.EncodeToString(bt.Deposit) {
			break
		}
		if witness.NextWitness == nil {
			//未找到
			return
		}
		witness = witness.NextWitness
	}
	if witness == nil {
		return
	}

	//	block := chain.GetLastBlock()
	//	for {
	//		if block.Height == bh.Height {
	//			break
	//		}
	//		if block.PreBlock == nil {
	//			//			fmt.Println("33333", block.Height, bh.Height)
	//			return
	//		}
	//		block = block.PreBlock
	//	}
	//	fmt.Println("添加投票到区块", block.Height)

	//	bs := bt.Json()
	//	id := utils.Hash_SHA3_256(*bs)

	if witness.ElectionMap == nil {
		witness.ElectionMap = new(sync.Map)
	}
	witness.ElectionMap.Store(bt.Addr.B58String(), bt)

	//	value, ok := block.ElectionMap.Load(hex.EncodeToString(bt.Deposit))
	//	//	fmt.Println("选票的交易id", ok, hex.EncodeToString(bt.Deposit))
	//	if ok {
	//		bts := value.(*sync.Map)
	//		bts.Store(hex.EncodeToString(id), bt)
	//		block.ElectionMap.Store(hex.EncodeToString(bt.Deposit), bts)

	//		total := 0
	//		block.ElectionMap.Range(func(k, v interface{}) bool { total++; return true })
	//		//		fmt.Println("----1111", total)
	//		return
	//	}
	//	bts := new(sync.Map)
	//	bts.Store(hex.EncodeToString(id), bt)
	//	block.ElectionMap.Store(hex.EncodeToString(bt.Deposit), bts)

	//	total := 0
	//	block.ElectionMap.Range(func(k, v interface{}) bool { total++; return true })
	//	fmt.Println("----2222", total)
}
