package utils

import (
	"strconv"

	"github.com/cespare/xxhash/v2"
)

func GetHash(b []byte) string {
	hash := xxhash.Sum64(b)
	return strconv.FormatUint(hash, 32)
}
