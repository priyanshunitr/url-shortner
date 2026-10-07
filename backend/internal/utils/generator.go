// Package utils contains all reusable code
package utils

import "crypto/rand"

func GenerateShortCode() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var code [8]byte
	for i := range code {
		// Rejection sampling avoids modulo bias (248 is a multiple of 62).
		var b [1]byte
		for {
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			if b[0] < 248 {
				code[i] = alphabet[int(b[0])%len(alphabet)]
				break
			}
		}
	}
	return string(code[:]), nil
}
