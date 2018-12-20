package keystore

import (
	"bytes"
	"encoding/binary"

	"fmt"

	"os"
	"path/filepath"
	"polarcloud/core/utils"
	"sync"
)

var (
	lock sync.RWMutex
)

//根据公钥生成mutilhash
func ParseHashByPubkey(pubkey []byte) (*utils.Multihash, error) {
	hash, err := buildAddrinfo(pubkey, Version)
	return hash, err
}

//int转byte
func intToBytes(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

//byte转int
func bytesToInt(b []byte) int64 {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int64
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	return int64(tmp)
}

//保存为文件
func writeToFile(seedpath, filename string, data []byte) error {
	lock.Lock()

	//newfile, err := os.OpenFile(filepath.Join(SeedPath, filename), os.O_RDWR|os.O_CREATE, os.ModePerm)
	ok := pathExists(seedpath)
	if !ok {
		os.Mkdir(seedpath, os.ModePerm)
	}
	newfile, err := os.OpenFile(filepath.Join(seedpath, filename), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	defer newfile.Close()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	_, err = newfile.Write(data)
	lock.Unlock()
	return err
}

//判断文件夹存不存在
func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
