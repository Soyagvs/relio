package semver

import (
	"testing"

	"github.com/soyagvs/go-release/internal/conventional"
)

func TestParseAndString(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"1.3.2", "1.3.2", false},
		{"v1.3.2", "v1.3.2", false},
		{"  v0.0.1 ", "v0.0.1", false},
		{"1.2", "", true},
		{"1.2.3.4", "", true},
		{"v1.2.3-rc1", "", true},
		{"vx.y.z", "", true},
		{"1.2.-3", "", true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("Parse(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got.String() != tt.want {
			t.Errorf("Parse(%q).String() = %q, want %q", tt.in, got.String(), tt.want)
		}
	}
}

func TestNext(t *testing.T) {
	v := Version{1, 3, 2, "v"}
	if got := v.Next(Patch).String(); got != "v1.3.3" {
		t.Errorf("patch = %q", got)
	}
	if got := v.Next(Minor).String(); got != "v1.4.0" {
		t.Errorf("minor = %q", got)
	}
	if got := v.Next(Major).String(); got != "v2.0.0" {
		t.Errorf("major = %q", got)
	}
	if got := v.Next(None).String(); got != "v1.3.2" {
		t.Errorf("none = %q", got)
	}
}

func TestBumpFor(t *testing.T) {
	mk := func(typ string, breaking bool) conventional.Commit {
		return conventional.Commit{Type: typ, Breaking: breaking}
	}
	tests := []struct {
		name    string
		commits []conventional.Commit
		want    Bump
	}{
		{"feat wins over fix", []conventional.Commit{mk("fix", false), mk("feat", false)}, Minor},
		{"breaking wins over feat", []conventional.Commit{mk("feat", false), mk("chore", true)}, Major},
		{"only fixes", []conventional.Commit{mk("fix", false), mk("fix", false)}, Patch},
		{"perf and refactor are patch", []conventional.Commit{mk("perf", false), mk("refactor", false)}, Patch},
		{"only chores", []conventional.Commit{mk("chore", false), mk("docs", false)}, None},
		{"empty", nil, None},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BumpFor(tt.commits); got != tt.want {
				t.Errorf("BumpFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZero(t *testing.T) {
	if Zero().Next(Minor).String() != "v0.1.0" {
		t.Errorf("zero minor = %q", Zero().Next(Minor).String())
	}
}
