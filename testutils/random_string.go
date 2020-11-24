package testutils

import (
	"math/rand"
	"time"
)

func RandomStringWithCharset(length int, charset string) string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789_abcdefghijklmnopqrstuvwxyz0123456789"

func RandomString(length int) string {
	return RandomStringWithCharset(length, charset)
}
