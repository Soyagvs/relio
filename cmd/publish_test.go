package cmd

import (
	"strings"
	"testing"
)

func TestReadYes(t *testing.T) {
	yes := []string{"y\n", "Y\n", "yes\n", "  yes  \n", "YES"}
	no := []string{"\n", "n\n", "no\n", "nope\n", "later", ""}

	for _, in := range yes {
		if !readYes(strings.NewReader(in)) {
			t.Errorf("readYes(%q) = false, want true", in)
		}
	}
	for _, in := range no {
		if readYes(strings.NewReader(in)) {
			t.Errorf("readYes(%q) = true, want false", in)
		}
	}
}
