package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	httpTimeout = 30 * time.Second

	// defaultAPIBase is the REST API root of github.com. Enterprise Server
	// serves the API somewhere else, and the operator passes that root as
	// written rather than a hostname this program would have to map.
	defaultAPIBase = "https://api.github.com"
)

type token struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	// RepositorySelection is "all" or "selected". It is the confirmation from
	// GitHub that the requested narrowing was applied.
	RepositorySelection string `json:"repository_selection"`
}

type client struct {
	base  string
	appID string
	key   *rsa.PrivateKey
	http  *http.Client
}

// newClient assumes the arguments have already been through config.validate,
// so an empty one here is a programming error rather than a user mistake.
func newClient(apiBase, appID, keyPath string) (*client, error) {
	pemBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	key, err := parseRSAPrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", keyPath, err)
	}
	return &client{
		base:  apiBase,
		appID: appID,
		key:   key,
		http:  &http.Client{Timeout: httpTimeout},
	}, nil
}

// mint is the only request this program makes. It carries no body, so the
// token comes back with everything the installation grants.
func (c *client) mint(ctx context.Context, installationID string) (*token, error) {
	jwt, err := c.signJWT()
	if err != nil {
		return nil, err
	}

	u := c.base + "/app/installations/" + url.PathEscape(installationID) + "/access_tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "github-app-tokend/"+version)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("POST %s: %s: %s", u, resp.Status, apiError(raw))
	}

	var t token
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	if err := checkToken(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func checkToken(t *token) error {
	if t.Token == "" {
		return errors.New("GitHub returned an empty token")
	}
	if t.ExpiresAt.IsZero() {
		return errors.New("GitHub returned no expiry, cannot schedule renewal")
	}
	return nil
}

// What GitHub says here is the whole of what an operator has to work from, so
// it is worth digging out of the payload rather than reporting the status code
// alone.
func apiError(raw []byte) string {
	var payload struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &payload)
	if payload.Message != "" {
		return payload.Message
	}
	return strings.TrimSpace(string(raw))
}
