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

// A failed renewal leaves the file untouched and is retried.
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
	calls := 0
	mint := func(context.Context) (*token, error) {
		calls++
		select {
		case minted <- struct{}{}:
		default:
		}
		// The first is past its expiry, so the loop comes straight back and the
		// second call proves the first write finished. Later ones last, so the
		// loop then waits instead of spinning until stopped.
		expiresAt := time.Now().Add(time.Hour)
		if calls == 1 {
			expiresAt = time.Now().Add(-time.Minute)
		}
		return &token{Token: "ghs_new", ExpiresAt: expiresAt}, nil
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
