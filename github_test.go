package main

import (
	"testing"
	"time"
)

func TestCheckTokenRejectsAResponseWithoutExpiry(t *testing.T) {
	ok := &token{Token: "ghs_x", ExpiresAt: time.Now().Add(time.Hour)}
	if err := checkToken(ok); err != nil {
		t.Errorf("a usable token was rejected: %v", err)
	}

	if err := checkToken(&token{Token: "ghs_x"}); err == nil {
		t.Error("a token with no expiry was accepted")
	}
}
