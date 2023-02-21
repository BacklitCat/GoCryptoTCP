package GoCryptoTCP

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"math/big"
)

const AesLetterSet = "0123456789abcdefghijklmnopqrstuvwxyz"

var AesKeyLen int = 16

func GenAESKey() []byte {
	key := make([]byte, AesKeyLen)
	for i := range key {
		randNum, _ := rand.Int(rand.Reader, big.NewInt(36))
		key[i] = AesLetterSet[randNum.Int64()]
	}
	return key
}

func AesEncryptCBC(key, plainText []byte) []byte {
	block, err := aes.NewCipher(key)
	CheckFatalErr(err)
	blockMode := cipher.NewCBCEncrypter(block, key[:aes.BlockSize])
	padText := PKCS7Padding(plainText, aes.BlockSize)
	cipherText := make([]byte, len(padText))
	blockMode.CryptBlocks(cipherText, padText)
	return cipherText
}

func AesDecryptCBC(key, cipherText []byte) []byte {
	block, _ := aes.NewCipher(key)
	blockMode := cipher.NewCBCDecrypter(block, key[:aes.BlockSize])
	padText := make([]byte, len(cipherText))
	blockMode.CryptBlocks(padText, cipherText)
	plainText := PKCS7Trimming(padText)
	return plainText
}

func PKCS7Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func PKCS7Trimming(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}
