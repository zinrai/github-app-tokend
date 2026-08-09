package main

import (
	"errors"
	"flag"
)

// config is everything the process was asked to do, settled before the private
// key is read and before the first request goes out.
type config struct {
	appID          string
	privateKey     string
	apiBase        string
	installationID string
	out            string
	showVersion    bool
}

// parseFlags reads the configuration from the command line and nowhere else.
// What a running process was asked to do, including how far the token it hands
// out reaches, is then visible in the process listing rather than in whatever
// happened to be exported around it.
func parseFlags() (*config, error) {
	var cfg config
	flag.StringVar(&cfg.appID, "app-id", "", "GitHub App ID or client ID")
	flag.StringVar(&cfg.privateKey, "private-key", "", "path to the App private key in PEM form")
	flag.StringVar(&cfg.apiBase, "api-base", defaultAPIBase, "REST API root, used as written")
	flag.StringVar(&cfg.installationID, "installation-id", "", "installation to mint tokens for")
	flag.StringVar(&cfg.out, "out", "", "file to write the token to")
	flag.BoolVar(&cfg.showVersion, "version", false, "print version information and exit")
	flag.Parse()

	if cfg.showVersion {
		return &cfg, nil
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validate rejects every combination the process cannot act on, in one place
// and before any of them has an effect. Checking a flag where it happens to be
// used instead would report a missing key after the daemon had already opened
// the key file.
func (c *config) validate() error {
	switch {
	case c.appID == "":
		return errors.New("-app-id is required")
	case c.privateKey == "":
		return errors.New("-private-key is required")
	case c.installationID == "":
		return errors.New("-installation-id is required")
	case c.out == "":
		return errors.New("-out is required")
	}
	return nil
}
