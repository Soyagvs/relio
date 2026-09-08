package semver

import (
	"testing"

	"github.com/soyagvs/relio/internal/conventional"
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
		{"vx.y.z", "", true},
		{"1.2.-3", "", true},
		// Pre-release is now first-class.
		{"v1.6.0-rc.1", "v1.6.0-rc.1", false},
		{"1.6.0-rc.2", "1.6.0-rc.2", false},
		{"v1.2.3-rc1", "v1.2.3-rc1", false},
		{"1.0.0-alpha.beta-3", "1.0.0-alpha.beta-3", false},
		// Build metadata stays rejected; so does a trailing hyphen and a
		// pre-release with characters outside [0-9A-Za-z.-].
		{"1.0.0+build", "", true},
		{"v1.0.0-", "", true},
		{"1.0.0-rc_1", "", true},
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
	v := Version{Major: 1, Minor: 3, Patch: 2, Prefix: "v"}
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

func TestPrereleaseHelpers(t *testing.T) {
	v, err := Parse("v1.6.0-rc.1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !v.IsPrerelease() {
		t.Error("IsPrerelease() = false, want true")
	}
	if v.Pre != "rc.1" {
		t.Errorf("Pre = %q, want rc.1", v.Pre)
	}
	if core := v.Core(); core.IsPrerelease() || core.String() != "v1.6.0" {
		t.Errorf("Core() = %q (prerelease=%v)", core, core.IsPrerelease())
	}
	if got := v.Core().WithPre("beta.2").String(); got != "v1.6.0-beta.2" {
		t.Errorf("WithPre() = %q, want v1.6.0-beta.2", got)
	}

	stable, err := Parse("v1.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if stable.IsPrerelease() {
		t.Error("stable.IsPrerelease() = true")
	}
}

func TestNextPre(t *testing.T) {
	tests := []struct {
		pre  string
		want string
	}{
		{"rc.1", "rc.2"},
		{"rc", "rc.1"},
		{"beta.9", "beta.10"},
		{"alpha", "alpha.1"},
	}
	for _, tt := range tests {
		v := Version{Major: 1, Minor: 6, Prefix: "v", Pre: tt.pre}
		got := v.NextPre()
		if got.Pre != tt.want {
			t.Errorf("NextPre(%q) Pre = %q, want %q", tt.pre, got.Pre, tt.want)
		}
		if got.Major != 1 || got.Minor != 6 || got.Patch != 0 || got.Prefix != "v" {
			t.Errorf("NextPre(%q) disturbed the core/prefix: %+v", tt.pre, got)
		}
	}
}

func TestCompareCoreOnly(t *testing.T) {
	mk := func(s string) Version {
		v, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
		return v
	}
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.6.0", "v1.5.0", 1},
		{"v1.5.0", "v1.6.0", -1},
		{"v1.6.0", "v1.6.0", 0},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.6.1", "v1.6.0", 1},
		// prerelease is ignored: compared cores are equal
		{"v1.6.0-rc.1", "v1.6.0-rc.2", 0},
		{"v1.6.0-rc.1", "v1.6.0", 0},
		{"v2.0.0-rc.1", "v1.6.0-rc.9", 1},
	}
	for _, tt := range tests {
		if got := Compare(mk(tt.a), mk(tt.b)); got != tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
