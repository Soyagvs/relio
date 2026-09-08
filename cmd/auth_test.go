package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// authStatus is the shared body of `relio auth status` and the menu's Auth →
// "Show sign-in status" entry. With no token resolvable it must say so without
// touching the network.
func TestAuthStatusSharedBySubcommandAndMenu(t *testing.T) {
	orig := lookupToken
	lookupToken = func() (string, string) { return "", "" }
	t.Cleanup(func() { lookupToken = orig })

	var buf bytes.Buffer
	if err := authStatus(&buf); err != nil {
		t.Fatalf("authStatus: %v", err)
	}
	if !strings.Contains(buf.String(), "not authenticated") {
		t.Errorf("authStatus with no token = %q, want it to mention 'not authenticated'", buf.String())
	}
}
