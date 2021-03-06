package store

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"polarcloud/core/utils"
)

type FileInfo struct {
	Hash       *utils.Multihash //文件hash
	Name       string           //真实文件名称
	Size       uint64           //文件总大小
	Time       int64            //文件上传时间
	FileChunk  *utils.SyncList  //文件块以及块共享者名单 value:*FileChunk
	ChunkCount uint64           //文件块总数
	//	lock       *sync.RWMutex    //读写锁
}

type FileInfoTempVO struct {
	Hash       *utils.Multihash   //文件md5
	Name       string             //真实文件名称
	Size       uint64             //文件总大小
	Time       int64              //文件上传时间
	FileChunk  []*FileChunkTempVO //文件块以及块共享者名单
	ChunkCount uint64             //文件块总数
}
type FileChunkTempVO struct {
	No    uint64             //文件块编号，从0开始递增
	Hash  *utils.Multihash   //块hash值
	Users []*utils.Multihash //共享的用户列表 key:string,value:*ShareUser
}

func (this *FileInfo) JSON() []byte {
	fitvo := FileInfoTempVO{
		Hash:       this.Hash,                   //文件md5
		Name:       this.Name,                   //真实文件名称
		Size:       this.Size,                   //文件大小
		Time:       this.Time,                   //文件上传时间
		FileChunk:  make([]*FileChunkTempVO, 0), //文件块以及块共享者名单
		ChunkCount: this.ChunkCount,             //文件块总数
	}
	for _, v := range this.FileChunk.GetAll() {
		one := v.(*FileChunk)
		fctVO := FileChunkTempVO{
			No:    one.No,                      //文件块编号，从0开始递增
			Hash:  one.Hash,                    //块hash值
			Users: make([]*utils.Multihash, 0), //共享的用户列表 key:string,value:*ShareUser
		}
		one.Users.Range(func(key interface{}, valueItr interface{}) bool {
			value := valueItr.(*ShareUser)
			fctVO.Users = append(fctVO.Users, value.Name)
			return true
		})
		fitvo.FileChunk = append(fitvo.FileChunk, &fctVO)
	}

	//	for _, one := range this.FileChunk {
	//		fctVO := FileChunkTempVO{
	//			No:    one.No,                      //文件块编号，从0开始递增
	//			Hash:  one.Hash,                    //块hash值
	//			Users: make([]*utils.Multihash, 0), //共享的用户列表 key:string,value:*ShareUser
	//		}
	//		one.Users.Range(func(key interface{}, valueItr interface{}) bool {
	//			value := valueItr.(*ShareUser)
	//			fctVO.Users = append(fctVO.Users, value.Name)
	//			return true
	//		})
	//		fitvo.FileChunk = append(fitvo.FileChunk, &fctVO)
	//	}

	bs, err := json.Marshal(fitvo)
	if err != nil {
		return nil
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(bs, &m)
	if err != nil {
		return nil
	}
	return bs
}

/*
	添加一个共享用户
	@return  bool  是否添加成功，已经存在也是添加成功
*/
func (this *FileInfo) AddShareUser(no uint64, user *utils.Multihash) bool {
	for i, v := range this.FileChunk.GetAll() {
		one := v.(*FileChunk)
		if one.No == no {
			itr := this.FileChunk.Get(i)
			two := itr.(*FileChunk)
			two.AddUpdateUser(user)
			return true
		}
	}
	return false

	//	done := false
	//	this.lock.RLock()
	//	//TODO 查找效率有待提高
	//	for i, one := range this.FileChunk {
	//		if one.No == no {
	//			this.FileChunk[i].AddUpdateUser(user)
	//			done = true
	//			break
	//		}
	//	}
	//	this.lock.RUnlock()
	//	return done
}

/*
	添加一个文件块
*/
func (this *FileInfo) AddChunk(chunk *FileChunk) {

	if chunk.No >= this.ChunkCount { //块编号从0开始
		return
	}

	have := false
	for _, v := range this.FileChunk.GetAll() {
		one := v.(*FileChunk)
		if one.No == chunk.No {
			have = true
			break
		}
	}
	if !have {
		this.FileChunk.Add(chunk)
	}

	//	this.lock.Lock()
	//	have := false
	//	//检查块编号是否存在
	//	for _, one := range this.FileChunk {
	//		if one.No == chunk.No {
	//			have = true
	//			break
	//		}
	//	}
	//	if !have {
	//		this.FileChunk = append(this.FileChunk, chunk)
	//	}
	//	this.lock.Unlock()
}

/*
	查找本地是否有文件块
*/
func (this *FileInfo) Have(hash string) (have bool) {

	for _, v := range this.FileChunk.GetAll() {
		one := v.(*FileChunk)
		if one.Hash.B58String() == hash {
			have = true
			break
		}
	}
	return

	//	this.lock.RLock()
	//	for _, one := range this.FileChunk {
	//		if one.Hash.B58String() == hash {
	//			have = true
	//			break
	//		}
	//	}
	//	have = false
	//	this.lock.RUnlock()
	//	return
}

/*
	查找本地是否有文件块
*/
func (this *FileInfo) FindChunk(hash string) (have *FileChunk) {
	for _, v := range this.FileChunk.GetAll() {
		one := v.(*FileChunk)
		if one.Hash.B58String() == hash {
			have = one
			break
		}
	}
	return

	//	this.lock.RLock()
	//	for _, one := range this.FileChunk {
	//		if one.Hash.B58String() == hash {
	//			have = one
	//			break
	//		}
	//	}
	//	this.lock.RUnlock()
	//	return
}

func ParseFileinfo(bs []byte) (*FileInfo, error) {
	fitVO := new(FileInfoTempVO)
	err := json.Unmarshal(bs, fitVO)
	if err != nil {
		return nil, err
	}

	fi := new(FileInfo)
	fi.FileChunk = utils.NewSyncList()
	fi.ChunkCount = fitVO.ChunkCount
	fi.Hash = fitVO.Hash
	fi.Name = fitVO.Name
	fi.Size = fitVO.Size
	fi.Time = fitVO.Time
	for _, one := range fitVO.FileChunk {
		fc := new(FileChunk)
		fc.Hash = one.Hash
		fc.No = one.No
		fc.Users = new(sync.Map)
		for _, name := range one.Users {
			fc.Users.Store(name.B58String(), NewShareUser(name))
		}
		fi.FileChunk.Add(fc)
		//		 = append(fi.FileChunk, fc)
	}
	return fi, nil
}

/*
	创建一个文件信息
*/
func NewFileInfo(hash *utils.Multihash, filename string, chunkCount uint64) *FileInfo {
	return &FileInfo{
		Hash:       hash,
		Name:       filename,
		FileChunk:  utils.NewSyncList(),
		ChunkCount: chunkCount,
	}
}

type FileChunk struct {
	No    uint64           //文件块编号，从0开始递增
	Hash  *utils.Multihash //块hash值
	Users *sync.Map        //共享的用户列表 key:string,value:*ShareUser
}

/*
	添加用户，用户已经存在则更新
*/
func (this *FileChunk) AddUpdateUser(name *utils.Multihash) {
	value, ok := this.Users.Load(name.B58String())
	if ok {
		u := value.(*ShareUser)
		atomic.StoreInt64(&u.UpdateTime, time.Now().Unix())
		return
	}
	u := NewShareUser(name)
	this.Users.Store(name.B58String(), u)
}

/*
	获取10分钟内在线的用户
*/
func (this *FileChunk) GetUserOnline() []*ShareUser {
	us := make([]*ShareUser, 0)
	this.Users.Range(func(key interface{}, valueItr interface{}) bool {
		value := valueItr.(*ShareUser)
		if !value.CheckOvertime(Time_sharefile * 2) {
			us = append(us, value)
		}
		return true
	})
	return us
}

/*
	获取所有用户
*/
func (this *FileChunk) GetUserAll() []*ShareUser {
	us := make([]*ShareUser, 0)
	this.Users.Range(func(key interface{}, valueItr interface{}) bool {
		value := valueItr.(*ShareUser)
		us = append(us, value)
		return true
	})
	return us
}

//随机获取一个共享用户
func (this *FileChunk) RandUser() *utils.Multihash {
	us := this.GetUserOnline()
	if len(us) <= 0 {
		us = this.GetUserAll()
	}
	names := make([]*utils.Multihash, 0)
	for _, one := range us {
		names = append(names, one.Name)
	}
	if len(names) <= 0 {
		return nil
	}
	rand.Seed(int64(time.Now().Nanosecond()))
	r := rand.Intn(len(names))
	return names[r]
}

/*
	清理60天都不在线的用户
*/
func (this *FileChunk) Clear() {
	us := this.GetUserAll()
	for _, one := range us {
		if one.CheckOvertime(Time_shareUserOfflineClear) {
			fmt.Println("清理掉用户", one.Name.B58String(), time.Now().Unix()-one.UpdateTime)
			this.Users.Delete(one.Name.B58String())
		}
	}
}

func NewFileChunk(no uint64, hash *utils.Multihash) *FileChunk {
	return &FileChunk{
		No:    no,
		Hash:  hash,
		Users: new(sync.Map),
	}
}

/*
	文件块共享用户
*/
type ShareUser struct {
	Name       *utils.Multihash //用户名称
	UpdateTime int64            //最后在线时间，一个用户3个月不在线，则从块中删除
}

/*
	检查是否超时
*/
func (this *ShareUser) CheckOvertime(t int64) bool {
	if this.UpdateTime+t <= time.Now().Unix() {
		return true
	}
	return false
}

func NewShareUser(name *utils.Multihash) *ShareUser {
	return &ShareUser{
		Name:       name,
		UpdateTime: time.Now().Unix(),
	}
}
