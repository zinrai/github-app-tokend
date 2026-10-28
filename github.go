package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	httpTimeout      = 30 * time.Second
	maxResponseBytes = 1 << 20

	// defaultAPIBase is the REST API root of github.com. Enterprise Server
	// serves the API somewhere else, and the operator passes that root as
	// written rather than a hostname this program would have to map.
	defaultAPIBase = "https://api.github.com"
)

type token struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	// RepositorySelection is "all" or "selected", the reach of the installation
	// as GitHub reports it. It is logged because no narrowing is requested here,
	// so the installation alone decides that reach.
	RepositorySelection string `json:"repository_selection"`
}

type client struct {
	base     string
	clientID string
	keyPath  string
	http     *http.Client
}

func newClient(apiBase, clientID, keyPath string) *client {
	return &client{
		base:     apiBase,
		clientID: clientID,
		keyPath:  keyPath,
		http:     &http.Client{Timeout: httpTimeout},
	}
}

func (c *client) mint(ctx context.Context, installationID string) (*token, error) {
	jwt, err := c.signJWT()
	if err != nil {
		return nil, err
	}

	u := c.base + "/app/installations/" + url.PathEscape(installationID) + "/access_tokens"
	// Sent without a body. Asking for fewer repositories or permissions would
	// put into this program a decision the installation already makes.
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

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
	if err := checkExpiry(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func checkExpiry(t *token) error {
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
