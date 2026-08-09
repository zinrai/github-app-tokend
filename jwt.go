package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"
)

const (
	// clockSkew backdates iat so GitHub does not reject a JWT issued by a
	// machine whose clock runs slightly fast. It counts against the ten
	// minutes GitHub allows between iat and exp.
	clockSkew = 30 * time.Second

	// The JWT signs one request and is then discarded, so there is no reason
	// to approach that ceiling.
	jwtLifetime = 5 * time.Minute
)

// signJWT builds the JWT that authenticates as the App itself, which is what
// GitHub accepts in exchange for an installation token.
func (c *client) signJWT() (string, error) {
	now := time.Now()
	claims := struct {
		Iss string `json:"iss"`
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
	}{
		Iss: c.appID,
		Iat: now.Add(-clockSkew).Unix(),
		Exp: now.Add(jwtLifetime).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) +
		"." + enc.EncodeToString(claimsJSON)

	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}
	return signingInput + "." + enc.EncodeToString(sig), nil
}

// GitHub hands out PKCS#1, and nothing in this program converts a key, so
// nothing here produces any other encoding.
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes.TrimSpace(pemBytes))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not the PKCS#1 private key GitHub hands out: %w", err)
	}
	return key, nil
}
