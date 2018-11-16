package utils

import (
	"golang.org/x/crypto/sha3"
)

func Hash_SHA3_256(bs []byte) []byte {
	hash_sha3 := sha3.New256()
	hash_sha3.Write(bs)
	return hash_sha3.Sum(nil)
}

func Hash_SHA3_512(bs []byte) []byte {
	hash_sha3 := sha3.New512()
	hash_sha3.Write(bs)
	return hash_sha3.Sum(nil)
}
