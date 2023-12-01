package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Transform(in interface{}, dst interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func Sha256(in []byte) string {
	m := sha256.New()
	m.Write(in)
	res := hex.EncodeToString(m.Sum(nil))
	return res
}

type ListType interface {
	~string | ~int | ~uint32 | ~[]byte | ~rune | ~float64
}

func InList[ListType comparable](list []ListType, val ListType) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
