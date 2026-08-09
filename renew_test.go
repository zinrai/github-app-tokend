package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// runLoop starts loop in the background and returns a function that stops it.
func runLoop(t *testing.T, mint minter, out string, tok *token) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := loop(ctx, mint, out, tok, time.Millisecond); err != nil {
			t.Errorf("loop: %v", err)
		}
	}()
	return func() {
		cancel()
		wg.Wait()
	}
}

// A delay that is too long lets the token expire, and one that is wrongly zero
// spins the loop against the API. The two lifetimes are here because the
// margin has to follow the token rather than a number written down once.
func TestRenewDelayLeavesAThirdOfTheLife(t *testing.T) {
	if d := renewDelay(time.Now().Add(time.Hour)); d < 39*time.Minute || d > 40*time.Minute {
		t.Errorf("delay for an hour = %s, want just under 40m", d)
	}
	if d := renewDelay(time.Now().Add(30 * time.Minute)); d < 19*time.Minute || d > 20*time.Minute {
		t.Errorf("delay for half an hour = %s, want just under 20m", d)
	}
	if d := renewDelay(time.Now().Add(-time.Hour)); d != 0 {
		t.Errorf("delay for an expired token = %s, want 0", d)
	}
}

// An outage at GitHub must not take away a credential that is still valid, so
// a failed renewal leaves the file untouched and keeps retrying.
func TestFailedRenewalKeepsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := writeToken(path, "ghs_current"); err != nil {
		t.Fatal(err)
	}

	attempts := make(chan struct{}, 8)
	mint := func(context.Context) (*token, error) {
		select {
		case attempts <- struct{}{}:
		default:
		}
		return nil, errors.New("GitHub is down")
	}

	// Already past its expiry, so renewal is due immediately.
	tok := &token{Token: "ghs_current", ExpiresAt: time.Now().Add(-time.Minute)}
	stop := runLoop(t, mint, path, tok)
	<-attempts
	<-attempts
	stop()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ghs_current\n" {
		t.Errorf("file = %q, want the previous token to be kept", b)
	}
}

func TestSuccessfulRenewalRewritesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := writeToken(path, "ghs_old"); err != nil {
		t.Fatal(err)
	}

	minted := make(chan struct{}, 8)
	mint := func(context.Context) (*token, error) {
		select {
		case minted <- struct{}{}:
		default:
		}
		// Past its expiry again, so the loop comes straight back and the second
		// call is proof that the first write has finished.
		return &token{Token: "ghs_new", ExpiresAt: time.Now().Add(-time.Minute)}, nil
	}

	tok := &token{Token: "ghs_old", ExpiresAt: time.Now().Add(-time.Minute)}
	stop := runLoop(t, mint, path, tok)
	<-minted
	<-minted
	stop()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ghs_new\n" {
		t.Errorf("file = %q, want the renewed token", b)
	}
}
