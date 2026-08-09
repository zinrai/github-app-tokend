// Command github-app-tokend keeps a GitHub App installation access token in a
// file, rewriting it before it expires.
//
// Applications read the file every time they need the token and do nothing
// else. No JWT signing, no GitHub API code and no private key on their side.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseFlags()
	if err != nil {
		return err
	}
	if cfg.showVersion {
		printVersion()
		return nil
	}

	client, err := newClient(cfg.apiBase, cfg.appID, cfg.privateKey)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The installation is fixed for the life of the process, so every request
	// the loop makes is this same one.
	mint := func(ctx context.Context) (*token, error) {
		return client.mint(ctx, cfg.installationID)
	}

	// Mint before entering the loop so that a misconfiguration fails the start
	// rather than leaving a process running with no file to show for it.
	tok, err := mint(ctx)
	if err != nil {
		return fmt.Errorf("initial token request: %w", err)
	}
	if err := writeToken(cfg.out, tok.Token); err != nil {
		return err
	}
	slog.Info("token written",
		"path", cfg.out,
		"installation_id", cfg.installationID,
		"expires_at", tok.ExpiresAt,
		"repository_selection", tok.RepositorySelection)

	return loop(ctx, mint, cfg.out, tok, retryInterval)
}
