package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	//"crypto/sha256"
	//"errors"
)

const checkSum = 4

//加密（主要用于私钥存储时加密）
func Encrypt(plantText, key []byte) ([]byte, error) {
	key = KeyPadding(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//chksum := sha256.Sum256(plantText)
	//vcode := chksum[:checkSum]
	plantText = PKCS7Padding(plantText, block.BlockSize())
	blockModel := cipher.NewCBCEncrypter(block, key)
	ciphertext := make([]byte, len(plantText))
	blockModel.CryptBlocks(ciphertext, plantText)
	//return append(vcode, ciphertext...), nil
	return ciphertext, nil
}

//解密
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	key = KeyPadding(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockModel := cipher.NewCBCDecrypter(block, key)
	/*if len(ciphertext) <= checkSum {
		return nil, errors.New("Decryption failed")
	}*/
	//vcode := ciphertext[:checkSum]
	//ciphertext = ciphertext[checkSum:]
	plantText := make([]byte, len(ciphertext))
	blockModel.CryptBlocks(plantText, ciphertext)
	plantText = PKCS7UnPadding(plantText, block.BlockSize())
	/*chksum := sha256.Sum256(plantText)
	var errs error
	if bytes.Compare(vcode, chksum[:checkSum]) == 0 {
		errs = nil
	} else {
		errs = errors.New("Decryption failed")
	}*/
	return plantText, err
}
func KeyPadding(key []byte) []byte {
	if len(key) > 16 {
		return key
	}
	newkey := make([]byte, 16)
	copy(newkey, key)
	return newkey
}
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
func PKCS7UnPadding(plantText []byte, blockSize int) []byte {
	length := len(plantText)
	unpadding := int(plantText[length-1])
	var end int
	if length > unpadding {
		end = length - unpadding
	} else {
		end = length
	}
	return plantText[:end]
}
