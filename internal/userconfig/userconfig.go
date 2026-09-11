// Package userconfig manages Relio's global, per-user preferences: settings
// that apply across every repository and are independent of any git
// repository or .release.yaml.
//
// It resolves under os.UserConfigDir() rather than a hardcoded ~/.config, so
// it works on macOS and Windows too. It is deliberately tolerant: Load never
// fails — a missing directory, missing file, or corrupt file all just yield
// defaults, because a broken preferences file must never block a CLI run the
// user did not start to fix preferences.
package userconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// dirName is the subdirectory created under os.UserConfigDir().
const dirName = "relio"

// fileName is the preferences file within Dir().
const fileName = "config.yaml"

// Config is the global preferences file.
type Config struct {
	Language string `yaml:"language"`
}

// Dir returns the directory Relio's global preferences live in. It does not
// create the directory.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, dirName), nil
}

// Path returns the full path to the preferences file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// Load reads the global preferences. It never returns an error: a missing
// directory, a missing file, an unreadable file, or invalid YAML all yield a
// zero Config.
func Load() Config {
	p, err := Path()
	if err != nil {
		return Config{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Config{}
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}
	}
	return c
}

// Save writes c to the global preferences file, creating the directory
// (mode 0o700) on first use. It writes to a temp file in the same directory
// and renames it into place (mode 0o600), so an interrupted write can never
// truncate existing preferences.
func Save(c Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".config-*.yaml.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, p); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
