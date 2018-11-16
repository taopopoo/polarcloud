package message_center

import (
	//	"fmt"
	"sync"
	"time"
)

const (
	MSG_WAIT_http_request = "MSG_WAIT_http_request" //
	CLASS_findfileinfo    = "CLASS_findfileinfo"    //查找文件信息
	CLASS_downloadfile    = "CLASS_downloadfile"    //下载文件块
	CLASS_sharefile       = "CLASS_sharefile"       //共享文件
	CLASS_syncfileinfo    = "CLASS_syncfileinfo"    //文件块分布式存储
	CLASS_getfourNodeinfo = "CLASS_getfourNodeinfo" //获取1/4节点信息(APP用)
	CLASS_safemsginfo     = "CLASS_safemsginfo"     //消息安全协议

	CLASS_uploadinfo    = "CLASS_uploadinfo"   //文件http上传信息
	CLASS_syncdata      = "CLASS_syncdata"     //分布式数据同步
	CLASS_raftvote      = "CLASS_raftvote"     //raft发起投票
	CLASS_raftvoteheart = "CLASS_raftvotehear" //raft发起心跳

	CLASS_findHeightBlock = "CLASS_findHeightBlock" //查询区块高度
	CLASS_getBlockHead    = "CLASS_getBlockHead"    //获取区块头
	CLASS_getTransaction  = "CLASS_getTransaction"  //获取交易

	waitRequstTime = 30 //超时时间设置为60秒
)

var (
	//	waitRequestLock = new(sync.RWMutex)
	waitRequest = new(sync.Map)

//	make(map[string]*HttpRequestWait)
)

type HttpRequestWait struct {
	//	lock   *sync.RWMutex
	//	tagMap map[string]chan *[]byte
	tagMap *sync.Map
}

/*
	等待请求返回
*/
func WaitRequest(class, tag string) *[]byte {
	//	defer fmt.Println("finish done")
	//	fmt.Println("waitrecv params", class, tag)
	//	waitRequestLock.Lock()
	rwItr, ok := waitRequest.Load(class) //[class]
	if !ok {
		//		fmt.Println("1111111111111111")
		c := make(chan *[]byte, 1)
		//		fmt.Println("打印地址 1 ", &c)
		hrw := HttpRequestWait{
			//			lock:   new(sync.RWMutex),
			tagMap: new(sync.Map), //make(map[string]chan *[]byte),
		}
		hrw.tagMap.Store(tag, c)       //[tag] = c
		waitRequest.Store(class, &hrw) // [class] = &hrw
		//		waitRequestLock.Unlock()
		//		fmt.Println("2222222222222")
		ticker := time.NewTicker(time.Second * waitRequstTime)
		//		fmt.Println("2222222222222 111111111")

		//		for key, value := range hrw.tagMap {
		//			fmt.Println("-------", key, value)
		//		}

		select {
		case <-ticker.C:
			//			fmt.Println("33333333333333")
			//			rw.lock.Lock()
			//			delete(rw.tagMap, tag)
			hrw.tagMap.Delete(tag)
			//			rw.lock.Unlock()
			return nil
		case bs := <-c:
			//			fmt.Println("4444444444444")
			ticker.Stop()
			//			fmt.Println("4444444444444 5555")
			return bs
		}

	}
	rw := rwItr.(*HttpRequestWait)
	//	waitRequestLock.Unlock()
	//	fmt.Println("555555555555555")
	//	rw.lock.Lock()
	cItr, ok := rw.tagMap.Load(tag) // [tag]

	//	for key, value := range rw.tagMap {
	//		fmt.Println("++++++", key, value)
	//	}

	if !ok {
		//		fmt.Println("66666666666666666")
		c := make(chan *[]byte, 1)
		rw.tagMap.Store(tag, c) // [tag] = c
		//		rw.lock.Unlock()

		ticker := time.NewTicker(time.Second * waitRequstTime)
		//		fmt.Println("66666666666666666 111111")
		select {
		case <-ticker.C:
			//			fmt.Println("777777777777777")
			//			rw.lock.Lock()
			//			delete(rw.tagMap, tag)
			rw.tagMap.Delete(tag)
			//			rw.lock.Unlock()
			return nil
		case bs := <-c:
			//			fmt.Println("888888888888")
			ticker.Stop()
			return bs
		}
	}
	c := cItr.(chan *[]byte)
	//	fmt.Println("555555555555555 1111111")
	//	rw.lock.Unlock()

	ticker := time.NewTicker(time.Second * waitRequstTime)
	//	fmt.Println("555555555555555 22222222222")
	select {
	case <-ticker.C:
		//		fmt.Println("999999999999999")
		//		rw.lock.Lock()
		//		delete(rw.tagMap, tag)
		rw.tagMap.Delete(tag)
		//		rw.lock.Unlock()
		return nil
	case bs := <-c:
		//		fmt.Println("10101010101010101010101010101010")
		ticker.Stop()
		return bs
	}
}

/*
	返回等待
*/
func ResponseWait(class, tag string, bs *[]byte) {
	//	fmt.Println("waitrev recv", class, tag)
	//	fmt.Println("recv 111111111111111111")
	//	waitRequestLock.RLock()
	rwItr, ok := waitRequest.Load(class) // [class]
	if !ok {
		//		fmt.Println("recv 22222222222222")
		//		waitRequestLock.RUnlock()
		return
	}
	rw := rwItr.(*HttpRequestWait)
	//	waitRequestLock.RUnlock()

	//	rw.lock.Lock()
	//	fmt.Println("recv 333333333333")
	cItr, ok := rw.tagMap.Load(tag) // [tag]

	//	for key, value := range rw.tagMap {
	//		fmt.Println("recv", key, value)
	//	}

	if !ok {
		//		rw.lock.Unlock()
		//		fmt.Println("recv 444444444444444")
		return
	}
	//	rw.lock.Unlock()
	c := cItr.(chan *[]byte)

	select {
	case c <- bs:
		//		fmt.Println("recv 555555555555555")
		return
	default:
		//		fmt.Println("放入失败")
	}
}
