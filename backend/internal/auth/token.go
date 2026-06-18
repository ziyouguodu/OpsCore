package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	UserID    int64    `json:"uid"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	ExpiresAt int64    `json:"exp"`
}

type Signer struct {
	secret []byte
	ttl    time.Duration
}

func NewSigner(secret string, ttl time.Duration) Signer {
	return Signer{secret: []byte(secret), ttl: ttl}
}

func (s Signer) Issue(userID int64, username string, roles []string) (string, error) {
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Roles:     roles,
		ExpiresAt: time.Now().Add(s.ttl).Unix(),
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	signature := s.sign(payload)
	return payload + "." + signature, nil
}

func (s Signer) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Claims{}, errors.New("invalid token")
	}
	if !hmac.Equal([]byte(s.sign(parts[0])), []byte(parts[1])) {
		return Claims{}, errors.New("invalid signature")
	}

	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(body, &claims); err != nil {
		return Claims{}, err
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return Claims{}, errors.New("token expired")
	}
	return claims, nil
}

func (s Signer) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
