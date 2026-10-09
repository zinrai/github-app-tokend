package main

import (
	"testing"
	"time"
)

func TestCheckExpiryRejectsAResponseWithoutExpiry(t *testing.T) {
	ok := &token{Token: "ghs_x", ExpiresAt: time.Now().Add(time.Hour)}
	if err := checkExpiry(ok); err != nil {
		t.Errorf("a usable token was rejected: %v", err)
	}

	if err := checkExpiry(&token{Token: "ghs_x"}); err == nil {
		t.Error("a token with no expiry was accepted")
	}
}
