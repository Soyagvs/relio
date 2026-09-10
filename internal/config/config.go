// Package config loads and writes the project-level .release.yaml file.
//
// The file holds configuration only — never secrets. Credentials live in the OS
// keychain and are handled elsewhere.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/soyagvs/relio/internal/versionfile"
)

// FileName is the config file expected at the repository root.
const FileName = ".release.yaml"

// Config mirrors the .release.yaml schema.
type Config struct {
	Project    string        `yaml:"project"`
	Versioning string        `yaml:"versioning"` // only "semver" for now
	Commits    string        `yaml:"commits"`    // only "conventional" for now
	Release    ReleaseConfig `yaml:"release"`
	GitHub     GitHubConfig  `yaml:"github"`
	Content    ContentConfig `yaml:"content"`
}

// ReleaseConfig controls what a `release` run touches.
//
// Changelog and Tag are pointers so an omitted key can mean "use the default"
// (both on) while an explicit `changelog: false` / `tag: false` is still
// honoured. Read them through ChangelogEnabled / TagEnabled.
type ReleaseConfig struct {
	Changelog     *bool         `yaml:"changelog"`
	ChangelogFile string        `yaml:"changelog_file"`
	Tag           *bool         `yaml:"tag"`
	TagPrefix     string        `yaml:"tag_prefix"`
	CompareLink   bool          `yaml:"compare_link"`
	Contributors  bool          `yaml:"contributors"`
	VersionFiles  []VersionFile `yaml:"version_files,omitempty"`
	Hooks         HooksConfig   `yaml:"hooks,omitempty"`
}

// StringList is a YAML value that accepts either a bare string or a list of
// strings, and always decodes to a slice. It marshals back to a bare string when
// it holds exactly one entry, so a single-command hook round-trips cleanly.
type StringList []string

// UnmarshalYAML accepts a scalar (one command) or a sequence of scalars.
func (s *StringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var one string
		if err := node.Decode(&one); err != nil {
			return err
		}
		*s = StringList{one}
		return nil
	case yaml.SequenceNode:
		var many []string
		if err := node.Decode(&many); err != nil {
			return err
		}
		*s = StringList(many)
		return nil
	default:
		return fmt.Errorf("config: expected a string or a list of strings")
	}
}

// MarshalYAML emits a bare string for a single entry, the slice otherwise, and
// nil for an empty list so `omitempty` drops the key.
func (s StringList) MarshalYAML() (any, error) {
	switch len(s) {
	case 0:
		return nil, nil
	case 1:
		return s[0], nil
	default:
		return []string(s), nil
	}
}

// HooksConfig holds the shell commands run around a release: `before` after the
// user confirms but before anything is written (a failure aborts the release),
// `after` once the tag and any publish step are done (a failure only warns).
type HooksConfig struct {
	Before StringList `yaml:"before,omitempty"`
	After  StringList `yaml:"after,omitempty"`
}

// ChangelogEnabled reports whether a release run updates the changelog file. An
// omitted `changelog:` key defaults to true.
func (c ReleaseConfig) ChangelogEnabled() bool { return c.Changelog == nil || *c.Changelog }

// TagEnabled reports whether a release run commits and tags. An omitted `tag:`
// key defaults to true.
func (c ReleaseConfig) TagEnabled() bool { return c.Tag == nil || *c.Tag }

func boolPtr(b bool) *bool { return &b }

// VersionFile is one entry in release.version_files. In YAML it accepts either a
// bare string (the path, matched by a built-in rule) or a mapping with an
// explicit regexp: `- {path: foo.py, pattern: '__version__ = "([^"]+)"'}`.
type VersionFile struct {
	Path    string
	Pattern string
}

type versionFileAlias struct {
	Path    string `yaml:"path"`
	Pattern string `yaml:"pattern,omitempty"`
}

// UnmarshalYAML accepts a scalar (path only) or a mapping.
func (v *VersionFile) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Decode(&v.Path)
	case yaml.MappingNode:
		var a versionFileAlias
		if err := node.Decode(&a); err != nil {
			return err
		}
		v.Path, v.Pattern = a.Path, a.Pattern
		return nil
	default:
		return fmt.Errorf("config: version_files entry must be a string or a {path, pattern} map")
	}
}

// MarshalYAML emits a bare string when there is no explicit pattern, so a clean
// config round-trips without noise.
func (v VersionFile) MarshalYAML() (any, error) {
	if v.Pattern == "" {
		return v.Path, nil
	}
	return versionFileAlias{Path: v.Path, Pattern: v.Pattern}, nil
}

// VersionTargets converts the configured version_files into versionfile targets.
func (c ReleaseConfig) VersionTargets() []versionfile.Target {
	if len(c.VersionFiles) == 0 {
		return nil
	}
	ts := make([]versionfile.Target, len(c.VersionFiles))
	for i, vf := range c.VersionFiles {
		ts[i] = versionfile.Target{Path: vf.Path, Pattern: vf.Pattern}
	}
	return ts
}

// GitHubConfig controls the opt-in GitHub Release step: pushing the branch and
// tag to origin and creating the Release with the changelog notes as its body.
type GitHubConfig struct {
	Enabled bool   `yaml:"enabled"`
	Repo    string `yaml:"repo"`    // "owner/name" override; empty = derive from origin
	Release bool   `yaml:"release"` // publish a GitHub Release automatically on `relio`
}

// ContentConfig is a placeholder for the v0.3.0 content generator.
type ContentConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Default returns a config suitable for a fresh `release init`.
func Default(project string) Config {
	return Config{
		Project:    project,
		Versioning: "semver",
		Commits:    "conventional",
		Release: ReleaseConfig{
			Changelog:     boolPtr(true),
			ChangelogFile: "CHANGELOG.md",
			Tag:           boolPtr(true),
			TagPrefix:     "v",
		},
		GitHub:  GitHubConfig{Enabled: false},
		Content: ContentConfig{Enabled: false},
	}
}

// withDefaults fills empty fields so older/partial files still work.
func (c Config) withDefaults() Config {
	if c.Versioning == "" {
		c.Versioning = "semver"
	}
	if c.Commits == "" {
		c.Commits = "conventional"
	}
	if c.Release.ChangelogFile == "" {
		c.Release.ChangelogFile = "CHANGELOG.md"
	}
	if c.Release.TagPrefix == "" {
		c.Release.TagPrefix = "v"
	}
	return c
}

// Path returns the config path for a repo root.
func Path(root string) string { return filepath.Join(root, FileName) }

// Exists reports whether a config file is present at root.
func Exists(root string) bool {
	_, err := os.Stat(Path(root))
	return err == nil
}

// ErrNotFound is returned by Load when no config file exists.
var ErrNotFound = errors.New("config: .release.yaml not found (run `release init`)")

// Load reads and validates the config at root.
func Load(root string) (Config, error) {
	data, err := os.ReadFile(Path(root))
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, ErrNotFound
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("config: parsing %s: %w", FileName, err)
	}
	c = c.withDefaults()
	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) validate() error {
	if c.Versioning != "semver" {
		return fmt.Errorf("config: unsupported versioning %q (only \"semver\")", c.Versioning)
	}
	if c.Commits != "conventional" {
		return fmt.Errorf("config: unsupported commits %q (only \"conventional\")", c.Commits)
	}
	return nil
}

// Save writes c to root/.release.yaml. It refuses to overwrite an existing file.
func (c Config) Save(root string) error {
	p := Path(root)
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("config: %s already exists", FileName)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	header := "# release configuration — safe to commit. Never put secrets here.\n"
	return os.WriteFile(p, append([]byte(header), data...), 0o644)
}
