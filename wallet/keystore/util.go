package keystore

import (
	"bytes"
	"encoding/binary"

	"fmt"

	"os"
	"path/filepath"
	"sync"
	"yunpan/core/utils"
)

var (
	lock sync.RWMutex
)

func ParseHashByPubkey(pubkey []byte) (*utils.Multihash, error) {
	hash, err := buildAddrinfo(pubkey, Version)
	return hash, err
}
func intToBytes(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func bytesToInt(b []byte) int64 {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int64
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	return int64(tmp)
}
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
