package ident

import (
	"crypto/rand"
	"fmt"
)

const textLength = 36

func New() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("génération d'identifiant : %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func Valid(candidate string) bool {
	if len(candidate) != textLength {
		return false
	}
	for i := range len(candidate) {
		if isSeparatorPosition(i) {
			if candidate[i] != '-' {
				return false
			}
			continue
		}
		if !isHex(candidate[i]) {
			return false
		}
	}
	return true
}

func isSeparatorPosition(i int) bool {
	return i == 8 || i == 13 || i == 18 || i == 23
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
