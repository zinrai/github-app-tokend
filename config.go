package main

import (
	"errors"
	"flag"
	"os"
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
func parseFlags(args []string) (*config, error) {
	var cfg config
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&cfg.clientID, "client-id", "", "GitHub App client ID")
	fs.StringVar(&cfg.privateKey, "private-key", "", "path to the App private key in PEM form")
	fs.StringVar(&cfg.apiBase, "api-base", defaultAPIBase, "REST API root, used as written")
	fs.StringVar(&cfg.installationID, "installation-id", "", "installation to mint tokens for")
	fs.StringVar(&cfg.out, "out", "", "file to write the token to")
	fs.BoolVar(&cfg.oneshot, "oneshot", false, "write one token and exit instead of renewing it")
	fs.BoolVar(&cfg.showVersion, "version", false, "print version information and exit")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

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
