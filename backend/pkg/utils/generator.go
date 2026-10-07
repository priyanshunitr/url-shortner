// Package utils contains all reusable code
package utils

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func GenerateShortCode() string {
	//nanoid
	chars := "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789-_"

	id, err := gonanoid.Generate(chars, 8)
	if err != nil {
		panic(err)
	}
	return id
}
