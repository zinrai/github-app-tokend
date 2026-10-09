package main

import (
	"errors"
	"flag"
	"io"
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
	if err := refuseRepeats(fs, args); err != nil {
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

// refuseRepeats parses args again only to count each flag, as flag lets a
// repeated one silently replace the first. Wrapping each flag.Value instead
// would lose the type names and defaults that -h prints.
func refuseRepeats(fs *flag.FlagSet, args []string) error {
	count := flag.NewFlagSet(fs.Name(), flag.ContinueOnError)
	count.SetOutput(io.Discard)
	seen := make(map[string]bool)
	fs.VisitAll(func(f *flag.Flag) {
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			count.BoolFunc(f.Name, "", onlyOnce(seen, f.Name))
		} else {
			count.Func(f.Name, "", onlyOnce(seen, f.Name))
		}
	})
	return count.Parse(args)
}

func onlyOnce(seen map[string]bool, name string) func(string) error {
	return func(string) error {
		if seen[name] {
			return errors.New("given more than once")
		}
		seen[name] = true
		return nil
	}
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
