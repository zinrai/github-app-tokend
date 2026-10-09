package main

import "testing"

func TestParseFlagsRefusesARepeatedFlag(t *testing.T) {
	base := []string{"-client-id", "Iv1.abc123", "-private-key", "key.pem", "-installation-id", "87654321", "-out", "token"}
	if _, err := parseFlags(base); err != nil {
		t.Fatalf("flags given once were rejected: %v", err)
	}

	for name, extra := range map[string][]string{
		"-out":     {"-out", "other"},
		"-out=":    {"-out=other"},
		"-oneshot": {"-oneshot", "-oneshot=false"},
	} {
		if _, err := parseFlags(append(base, extra...)); err == nil {
			t.Errorf("repeated %s was accepted", name)
		}
	}
}
