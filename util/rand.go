package util

import mathrand "math/rand"

func GetToken(length int) string {
	token := make([]byte, length)
	bytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!#$%^&*"
	for i:=0; i<length; i++  {
		token[i] = bytes[mathrand.Int()%len(bytes)]
	}
	return string(token)
}
