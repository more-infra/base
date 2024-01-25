package algoutil

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

func CalcStringMD5(s string) string {
	return CalcMD5([]byte(s))
}

func CalcMD5(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func CalcFileMD5(file string) (string, error) {
	h := md5.New()
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := io.CopyBuffer(h, f, nil); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
