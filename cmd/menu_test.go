package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
)

func TestRunMenuSetupOnExistingConfig(t *testing.T) {
	_, dir := initTempRepo(t)
	if err := config.Default("proj").Save(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetIn(strings.NewReader(""))

	if err := runMenuSetup(cmd, &releaseFlags{dir: dir}); err != nil {
		t.Fatalf("runMenuSetup: %v", err)
	}
	if !strings.Contains(buf.String(), "already exists") {
		t.Errorf("Setup on an existing config = %q, want it to mention 'already exists'", buf.String())
	}
}
