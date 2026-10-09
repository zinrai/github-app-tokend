package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// A JWT that does not verify, or one GitHub rejects for exceeding the
// ten-minute window between iat and exp, makes every request fail.
func TestSignedJWTVerifiesAndFitsGitHubsWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key.pem")
	key := writeKey(t, path)
	c := newClient(defaultAPIBase, "Iv1.abc123", path)
	tok, err := c.signJWT()
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("want 3 segments, got %d", len(parts))
	}
	enc := base64.RawURLEncoding

	hdr, err := enc.DecodeString(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(hdr) != `{"alg":"RS256","typ":"JWT"}` {
		t.Errorf("header = %s", hdr)
	}

	raw, err := enc.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		Iss      string
		Iat, Exp int64
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Iss != "Iv1.abc123" {
		t.Errorf("iss = %q", claims.Iss)
	}
	if span := claims.Exp - claims.Iat; span > 600 {
		t.Errorf("iat..exp = %ds, exceeds GitHub's 600s limit", span)
	}
	if claims.Exp <= time.Now().Unix() {
		t.Error("JWT is already expired when issued")
	}

	if err := verifyJWT(tok, &key.PublicKey); err != nil {
		t.Fatalf("signature does not verify: %v", err)
	}
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
