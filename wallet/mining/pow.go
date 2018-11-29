package mining

import (
	"bytes"
	"fmt"
	"polarcloud/config"
	"sync"
)

var powBlockHeadLock = new(sync.RWMutex)
var powBlockHead *BlockHead //正在挖矿寻找幸运数字的块
var stopSignalChan = make(chan bool, 1)

/*
	其他节点已经出块，停止寻找幸运数字
*/
func stopFindNonce(bh *BlockHead) {
	//	fmt.Println("调用中断方法")
	powBlockHeadLock.Lock()
	if powBlockHead != nil &&
		bytes.Equal(powBlockHead.Previousblockhash, bh.Previousblockhash) {
		fmt.Println("其他矿工率先出块，中断挖矿，下次继续努力！")
		stopSignalChan <- true
	}
	powBlockHeadLock.Unlock()
	//	fmt.Println("调用中断方法完成")
}

/*
	开始寻找幸运数字
*/
func findNonce(bh *BlockHead) (ok bool) {
	powBlockHeadLock.Lock()
	powBlockHead = bh
	powBlockHeadLock.Unlock()
	ok = <-bh.FindNonce(config.Mining_difficulty, stopSignalChan)
	powBlockHeadLock.Lock()
	powBlockHead = nil
	powBlockHeadLock.Unlock()
	return
}
