package main

import (
	"errors"
	"flag"
)

// config is everything the process was asked to do, settled before the private
// key is read and before the first request goes out.
type config struct {
	clientID       string
	privateKey     string
	apiBase        string
	installationID string
	out            string
	oneshot        bool
	showVersion    bool
}

// parseFlags reads the command line and nowhere else, so what a running process
// was asked to do is visible in the process listing rather than in whatever
// happened to be exported around it.
func parseFlags() (*config, error) {
	var cfg config
	flag.StringVar(&cfg.clientID, "client-id", "", "GitHub App client ID")
	flag.StringVar(&cfg.privateKey, "private-key", "", "path to the App private key in PEM form")
	flag.StringVar(&cfg.apiBase, "api-base", defaultAPIBase, "REST API root, used as written")
	flag.StringVar(&cfg.installationID, "installation-id", "", "installation to mint tokens for")
	flag.StringVar(&cfg.out, "out", "", "file to write the token to")
	flag.BoolVar(&cfg.oneshot, "oneshot", false, "write one token and exit instead of renewing it")
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

// validate reports every missing flag before anything has an effect. Checked
// where each is used instead, a missing -out would surface only after a token
// had been requested from GitHub.
func (c *config) validate() error {
	switch {
	case c.clientID == "":
		return errors.New("-client-id is required")
	case c.privateKey == "":
		return errors.New("-private-key is required")
	case c.installationID == "":
		return errors.New("-installation-id is required")
	case c.out == "":
		return errors.New("-out is required")
	}
	return nil
}
