package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
)

func InStringArray(array []string, str string) bool {
	for _, item := range array {
		if str == item {
			return true
		}
	}

	return false
}

func IsFileExist(path string) (bool, error) {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func ReadFile(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)

	defer func() {
		_ = f.Close()
	}()

	if err != nil {
		return nil, err
	} else {
		return ioutil.ReadAll(f)
	}
}

func GetUuid() string {
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return GetMd5String(base64.URLEncoding.EncodeToString(b))
}

// 获取md5方法
func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
