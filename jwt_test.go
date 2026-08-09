package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T) *client {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &client{
		base:  defaultAPIBase,
		appID: "12345",
		key:   key,
		http:  &http.Client{Timeout: 5 * time.Second},
	}
}

// A JWT that does not verify, or one GitHub rejects for exceeding the
// ten-minute window between iat and exp, makes every request fail.
func TestSignedJWTVerifiesAndFitsGitHubsWindow(t *testing.T) {
	c := testClient(t)
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
	if claims.Iss != "12345" {
		t.Errorf("iss = %q", claims.Iss)
	}
	if span := claims.Exp - claims.Iat; span > 600 {
		t.Errorf("iat..exp = %ds, exceeds GitHub's 600s limit", span)
	}
	if claims.Exp <= time.Now().Unix() {
		t.Error("JWT is already expired when issued")
	}

	sig, err := enc.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&c.key.PublicKey, crypto.SHA256, sum[:], sig); err != nil {
		t.Fatalf("signature does not verify: %v", err)
	}
}
