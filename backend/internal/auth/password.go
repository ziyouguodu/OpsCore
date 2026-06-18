package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const passwordIterations = 120000

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := pbkdf2SHA256([]byte(password), salt, passwordIterations, 32)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s",
		passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	var result []byte
	block := uint32(1)
	for len(result) < keyLen {
		u := hmacSHA256(password, append(salt, byte(block>>24), byte(block>>16), byte(block>>8), byte(block)))
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			u = hmacSHA256(password, u)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
		block++
	}
	return result[:keyLen]
}

func hmacSHA256(key, data []byte) []byte {
	blockSize := 64
	if len(key) > blockSize {
		sum := sha256.Sum256(key)
		key = sum[:]
	}

	padded := make([]byte, blockSize)
	copy(padded, key)

	oKeyPad := make([]byte, blockSize)
	iKeyPad := make([]byte, blockSize)
	for i := 0; i < blockSize; i++ {
		oKeyPad[i] = padded[i] ^ 0x5c
		iKeyPad[i] = padded[i] ^ 0x36
	}

	inner := sha256.Sum256(append(iKeyPad, data...))
	outer := sha256.Sum256(append(oKeyPad, inner[:]...))
	return outer[:]
}
