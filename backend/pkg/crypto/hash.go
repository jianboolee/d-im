package crypto

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
)

// MD5Hash 计算字符串的MD5哈希
func MD5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// SHA256Hash 计算字符串的SHA256哈希
func SHA256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
