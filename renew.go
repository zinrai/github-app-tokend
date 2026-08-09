package main

import (
	"context"
	"log/slog"
	"time"
)

// A failure leaves the token unchanged, so there is nothing new to compute a
// delay from. Thirty seconds is short enough to catch a brief outage and long
// enough not to hammer GitHub while it is down.
const retryInterval = 30 * time.Second

// minter requests a replacement token. Taking the request rather than the
// client is what the loop actually depends on, and it keeps the failure path
// verifiable without a GitHub standing in for itself.
type minter func(context.Context) (*token, error)

// loop rewrites the file before each token expires. A failed renewal leaves
// the existing file alone: an outage at GitHub must not take away a credential
// that is still valid.
func loop(ctx context.Context, mint minter, out string, tok *token, retry time.Duration) error {
	wait := renewDelay(tok.ExpiresAt)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}

		next, err := mint(ctx)
		// Checked before err so that a normal shutdown is not logged as a
		// failure. A token minted on the way out is not worth writing.
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			slog.Error(renewalFailure(tok.ExpiresAt),
				"error", err, "expires_at", tok.ExpiresAt, "retry_in", retry)
			wait = retry
			continue
		}

		if err := writeToken(out, next.Token); err != nil {
			return err
		}
		tok = next
		wait = renewDelay(tok.ExpiresAt)
		slog.Info("token renewed", "expires_at", tok.ExpiresAt)
	}
}

// renewalFailure separates the failure that needs no attention from the one
// where the file has stopped working. An outage lasting hours produces hundreds
// of these lines, and only the second kind means the applications reading the
// file are getting 401s.
func renewalFailure(expiresAt time.Time) string {
	if time.Now().Before(expiresAt) {
		return "renewal failed, keeping the current token"
	}
	return "renewal failed and the token in the file has expired"
}

// renewDelay leaves a third of the token's remaining life for renewal to
// happen in, which is where Vault and kubelet land on the same problem. Taking
// a fraction rather than a fixed offset keeps any assumption about how long
// GitHub makes a token last out of this program.
func renewDelay(expiresAt time.Time) time.Duration {
	if d := time.Until(expiresAt) * 2 / 3; d > 0 {
		return d
	}
	return 0
}
