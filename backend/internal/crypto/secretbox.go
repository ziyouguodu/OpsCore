package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const encryptedPrefix = "enc:v1:"

type SecretBox struct {
	aead cipher.AEAD
}

func NewSecretBox(key string) (SecretBox, error) {
	if len(key) < 32 {
		return SecretBox{}, errors.New("OPSCORE_CREDENTIAL_ENCRYPTION_KEY must be at least 32 bytes")
	}
	block, err := aes.NewCipher([]byte(key[:32]))
	if err != nil {
		return SecretBox{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return SecretBox{}, err
	}
	return SecretBox{aead: aead}, nil
}

func (b SecretBox) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (b SecretBox) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, encryptedPrefix) {
		return ciphertext, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedPrefix))
	if err != nil {
		return "", err
	}
	nonceSize := b.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("invalid encrypted secret")
	}
	plaintext, err := b.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func IsEncryptedSecret(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}
