package main

import "testing"

// Each of these is needed before the process can do anything, and reporting it
// at startup is the difference between a failed start and a daemon that runs
// with no file to show for it.
func TestValidateRequiredFlags(t *testing.T) {
	full := config{
		appID:          "12345",
		privateKey:     "/etc/github-app-tokend/key.pem",
		installationID: "87654321",
		out:            "/run/github-app-tokend/token",
	}
	if err := full.validate(); err != nil {
		t.Fatalf("a complete configuration was rejected: %v", err)
	}

	for name, clear := range map[string]func(*config){
		"-app-id":          func(c *config) { c.appID = "" },
		"-private-key":     func(c *config) { c.privateKey = "" },
		"-installation-id": func(c *config) { c.installationID = "" },
		"-out":             func(c *config) { c.out = "" },
	} {
		c := full
		clear(&c)
		if err := c.validate(); err == nil {
			t.Errorf("missing %s was accepted", name)
		}
	}
}
