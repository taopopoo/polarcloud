package utils

import (
	"bytes"
	"encoding/binary"
)

//uint64转byte
func Uint64ToBytes(n uint64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

//byte转uint64
func BytesToUint64(b []byte) uint64 {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp uint64
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	return tmp
}

//int64转byte
func Int64ToBytes(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

//byte转int64
func BytesToInt64(b []byte) int64 {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int64
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	return tmp
}
