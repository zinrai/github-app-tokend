package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeKey(t *testing.T, path string) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return key
}

func verifyJWT(jwt string, pub *rsa.PublicKey) error {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return fmt.Errorf("want 3 segments, got %d", len(parts))
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig)
}

func TestSignJWTUsesTheKeyFileAsItIsNow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key.pem")
	c := newClient(defaultAPIBase, "Iv1.abc123", path)

	for i := range 2 {
		key := writeKey(t, path)
		tok, err := c.signJWT()
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyJWT(tok, &key.PublicKey); err != nil {
			t.Errorf("JWT %d was not signed with the key then in the file: %v", i+1, err)
		}
	}
}
