package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	gconfig "yunpan/config"
	"yunpan/core/nodeStore"
	"yunpan/core/utils"
)

func main() {
	logicId()
	logicId2()
	//	near()
	//	BuildLogicIds()
	BuildIds()

	distance()

	//	nodeStore.NodeSelf.IdInfo = nodeStore.NewIdInfo("", "", "tao")

	//	self, _ := hex.DecodeString("3adb430304874d1adb2a77d75206deb3559bb8924e1fc98fb908f60acb0ccc4e")
	//	nodeStore.NodeSelf.IdInfo.Id.Id = self

	//	bs, _ := hex.DecodeString("78bf1110d2fc89865cd374f3ebeb0c93f6b9bb447f68c15bef6187797a1e251a")
	//	nodeStore.AddNode(&nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: nodeStore.NewIdAddress(bs)}})

	//	bs, _ = hex.DecodeString("7a9d51f65cf06bdfe7220ca99f2955260f2b6a042721d0c8d3e240f0d311fddb")
	//	nodeStore.AddNode(&nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: nodeStore.NewIdAddress(bs)}})

	//	bs, _ = hex.DecodeString("819360e97f4e8b91f1674537f06b6f921eeb427d9a000ebb7b2f1a30ec7c36ab")
	//	nodeStore.AddNode(&nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: nodeStore.NewIdAddress(bs)}})

	//	bs, _ = hex.DecodeString("873fb52e92033f073bab1d6a9429db6c4cde96e527222e3680b3be43df73becf")
	//	nodeStore.NodeSelf = &nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: nodeStore.NewIdAddress(bs)}}

	//	bs, _ = hex.DecodeString("b2009f3713b6c18caef1e50b1e5acdb31e065628d09336ddd3615a7910f3f5ee")
	//	nodeStore.AddNode(&nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: nodeStore.NewIdAddress(bs)}})

	//	bs = nodeStore.FindNearInSuper(self, nil, false)
	//	fmt.Println(hex.EncodeToString(bs))

	//	//	bs, _ = hex.DecodeString("3adb430304874d1adb2a77d75206deb3559bb8924e1fc98fb908f60acb0ccc4e")
	//	//	ok, repl := nodeStore.CheckNeedNode(bs)
	//	//	fmt.Println("---------", ok, repl)
}

/*
	获得逻辑节点地址
*/
func GetLogicNetId(n int64) string {
	x := big.NewInt(n)
	idbs, _ := utils.Encode(x.Bytes(), utils.SHA1)
	idmh := utils.Multihash(idbs)
	fmt.Println(n, "节点为", idmh.B58String())
	nlb := nodeStore.NewLogicNumBuider(&idmh, 3)
	for _, one := range nlb.GetNodeNetworkNum() {
		fmt.Println("逻辑节点为", new(big.Int).SetBytes(one.Data()).Int64(), one.B58String())
	}
	return idmh.B58String()
}

/*
	计算每个节点的逻辑节点
*/
func logicId() {
	nums := []int64{1, 2, 3, 4, 5, 6, 7}
	ids := make([]string, 0)
	for _, one := range nums {
		ids = append(ids, GetLogicNetId(one))
	}

	fmt.Println("============================")

	ids = []string{
		"5dsEMMhVbww4hUXV6VzaeRfHKv1nhh",
		"5dqqnW3YTxTw9EESzT63qM63zf9BYj",
		"5dtPgE32MoURWep7QEViBkZh5iVLDZ",
		"5duDDfkY1tChLKGxbAtPdPysp9ghYn",
		"5dry99WLtouG46QKHkR4Y52eJYU1N8",
	}

	for i, one := range ids {
		fmt.Println(nums[i], "的逻辑节点id", "\n")

		//		self := nodeStore.NewIdInfo("", "", "tao")
		//		idMH, _ := utils.FromB58String(one)
		//		self.Id = &idMH

		//		node := new(nodeStore.Node)
		//		node.IdInfo = self
		//		node.IsSuper = true

		//		m := nodeStore.NewNodeManager(gconfig.NodeIDLevel, node)

		nodeStore.NodeSelf.IdInfo = nodeStore.NewIdInfo("", "", "tao")
		idMH, _ := utils.FromB58String(one)
		nodeStore.NodeSelf.IdInfo.Id = &idMH
		nodeStore.NodeSelf.IsSuper = true

		nodeStore.Nodes = new(sync.Map)

		for _, two := range ids {
			if two == one {
				continue
			}
			idMH, _ := utils.FromB58String(two)
			nodeStore.AddNode(&nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: &idMH}})
		}

		for _, one := range nodeStore.GetAllNodes() {
			//			fmt.Println(new(big.Int).SetBytes(one.Data()).Int64(), one.B58String())
			fmt.Println(new(big.Int).SetBytes(one.Data()).Int64(), one.B58String())
		}
		fmt.Println("-----------")

	}

}

/*
	计算每个节点的逻辑节点
*/
func logicId2() {
	fmt.Println("logicId2")
	ids := []string{
		"W1rC5R6EMPSeQL62qCpDqsD4ZkWCPfcStAP3L4JZKWZTeM",
		"W1jJoD6HMhphYMjRDKYhL58PZjVH4YfrhxvystnvaUWKwk",
		"W1gw1neJNKN2jduwNKZkoMXv29zoVaCrBjJUJXpe7tiKA5",
		"W1gX5eLiRXPbWjyUEDC34MxeugTQ9gX4Tsd1rrVDjqvipL",
		"W1aomarfZM1XBwH6g94Mgf3a5Sjt4goYVKib2WQBHFAjBP",
		//"W1eY8KrS9ucP4y8tKX1bTJ8QuLe9bSRcgsKawwGQ8LysFx",
	}

	for _, one := range ids {
		fmt.Println(one, "的逻辑节点id", "\n")

		//		self := nodeStore.NewIdInfo("", "", "tao")
		//		idMH, _ := utils.FromB58String(one)
		//		self.Id = &idMH

		//		node := new(nodeStore.Node)
		//		node.IdInfo = self
		//		node.IsSuper = true

		//		m := nodeStore.NewNodeManager(gconfig.NodeIDLevel, node)

		nodeStore.NodeSelf.IdInfo = nodeStore.NewIdInfo("", "", "tao")
		idMH, _ := utils.FromB58String(one)
		//		fmt.Println(len(idMH.Data()), idMH.Data())
		nodeStore.NodeSelf.IdInfo.Id = &idMH
		nodeStore.NodeSelf.IsSuper = true

		nodeStore.Nodes = new(sync.Map)

		for _, two := range ids {
			if two == one {
				continue
			}
			idMH, _ := utils.FromB58String(two)
			nodeStore.AddNode(&nodeStore.Node{IdInfo: nodeStore.IdInfo{Id: &idMH}})
		}

		for _, one := range nodeStore.GetAllNodes() {
			//			fmt.Println(new(big.Int).SetBytes(one.Data()).Int64(), one.B58String())
			fmt.Println(one.B58String())
		}
		fmt.Println("-----------")

	}

}

/*
	计算一个节点的最近节点
*/
func near() {
	nears := []string{
		"5dsEMMhVbww4hUXV6VzaeRfHKv1nhh",
		"5dqqnW3YTxTw9EESzT63qM63zf9BYj",
		"5dtPgE32MoURWep7QEViBkZh5iVLDZ",
		"5duDDfkY1tChLKGxbAtPdPysp9ghYn",
		"5dry99WLtouG46QKHkR4Y52eJYU1N8",
	}
	checkIndex := 4
	kl := nodeStore.NewKademlia()

	for i, one := range nears {
		if i == checkIndex {
			continue
		}
		idMH, _ := utils.FromB58String(one)
		kl.Add(new(big.Int).SetBytes(idMH.Data()))
	}
	idMH, _ := utils.FromB58String(nears[checkIndex])
	id := kl.Get(new(big.Int).SetBytes(idMH.Data()))
	for _, one := range id {
		idbs, _ := utils.Encode(one.Bytes(), utils.SHA1)
		idmh := utils.Multihash(idbs)
		fmt.Println(idmh.B58String())
	}
}

/*
	测试构建逻辑节点
*/
func BuildLogicIds() {

	id := "5dsEMMhVbww4hUXV6VzaeRfHKv1nhh"
	idMH, _ := utils.FromB58String(id)
	lnb := nodeStore.NewLogicNumBuider(&idMH, 256)
	lnb.GetNodeNetworkNum()

}

/*
	测试查询逻辑节点
*/
func BuildIds() {

	fmt.Println("---------- BuildIds ----------")

	ids := []string{
		"W1aLWC4unTJZhSFc4VNLFsazAJ1PyTocV7agmteQDL3J3N",
		"W1gfVGa52yUJ4Gws4TiA9YbwGP8qCGgaYeeT8APjSiNk6U",
		"W1j9RJ1xYHaoAuRk2HGBrVA82njoxFAoctYKQMH43k8hXu",
		"W1atFt7bJ5Ubk4MXuV5GfsEYE7srWXR51exDgUEJcVr5fZ",
		"W1n9XtbLAjRsh9sr2kbwfkfy3VGenyhazbHJwrEYsnDZ8M",
	}

	for n := 0; n < len(ids); n++ {
		fmt.Println("本节点为", ids[n])
		index := n

		idMH, _ := utils.FromB58String(ids[index])
		idsm := nodeStore.NewIds(&idMH, gconfig.NodeIDLevel)
		for i, one := range ids {
			if i == index {
				continue
			}

			idMH, _ := utils.FromB58String(one)
			idsm.AddId(&idMH)
			//		ok, remove := idsm.AddId(&idMH)
			//		if ok {
			//			fmt.Println(one, remove)
			//		}
		}

		is := idsm.GetIds()
		for _, one := range is {
			fmt.Println("--逻辑节点", one.B58String())
		}

	}

}

/*
	计算节点距离
*/
func distance() {

	fmt.Println("---------- distance ----------")

	ids := []string{
		"W1aLWC4unTJZhSFc4VNLFsazAJ1PyTocV7agmteQDL3J3N",
		"W1gfVGa52yUJ4Gws4TiA9YbwGP8qCGgaYeeT8APjSiNk6U",
		"W1j9RJ1xYHaoAuRk2HGBrVA82njoxFAoctYKQMH43k8hXu",
		"W1atFt7bJ5Ubk4MXuV5GfsEYE7srWXR51exDgUEJcVr5fZ",
		"W1n9XtbLAjRsh9sr2kbwfkfy3VGenyhazbHJwrEYsnDZ8M",
	}

	index := 4

	kl := nodeStore.NewKademlia()
	for i, one := range ids {
		if i == index {
			continue
		}

		idMH, _ := utils.FromB58String(one)
		kl.Add(new(big.Int).SetBytes(idMH.Data()))

	}

	idMH, _ := utils.FromB58String(ids[index])
	is := kl.Get(new(big.Int).SetBytes(idMH.Data()))
	src := new(big.Int).SetBytes(idMH.Data())

	//	is := idsm.GetIds()
	for _, one := range is {
		tag := new(big.Int).SetBytes(one.Bytes())
		juli := tag.Xor(tag, src)

		bs, err := utils.Encode(one.Bytes(), gconfig.HashCode)
		if err != nil {
			fmt.Println("编码失败")
			continue
		}
		mh := utils.Multihash(bs)

		fmt.Println("排序结果", mh.B58String(), "距离", hex.EncodeToString(juli.Bytes()))
	}

}
