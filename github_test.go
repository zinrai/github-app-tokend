package main

import (
	"testing"
	"time"
)

// Without an expiry there is nothing to schedule renewal from, so the token
// would be served until it silently stopped working.
func TestCheckTokenRejectsUnusableResponses(t *testing.T) {
	ok := &token{Token: "ghs_x", ExpiresAt: time.Now().Add(time.Hour)}
	if err := checkToken(ok); err != nil {
		t.Errorf("a usable token was rejected: %v", err)
	}

	if err := checkToken(&token{ExpiresAt: time.Now().Add(time.Hour)}); err == nil {
		t.Error("an empty token was accepted")
	}

	if err := checkToken(&token{Token: "ghs_x"}); err == nil {
		t.Error("a token with no expiry was accepted")
	}
}
