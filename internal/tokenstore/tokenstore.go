// Package tokenstore stores, loads, and deletes the GitHub access token
// produced by the OAuth device flow.
//
// Storage is hybrid: the OS keychain (macOS Keychain, Windows Credential
// Manager, Linux Secret Service via go-keyring) is tried first, falling back
// to a protected local file when the keychain is unavailable or errors. This
// is deliberate rather than keychain-only: many Linux setups -- notably
// WSL2, which has no Secret Service provider running by default -- would
// leave `relio auth login` broken if the keychain were the only option. The
// hybrid approach keeps the "OS keychain" promise where a keychain exists,
// while still working everywhere else.
package tokenstore

import (
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// serviceName and accountName identify the credential within the OS
// keychain.
const (
	serviceName = "relio"
	accountName = "github"
)

// dirName is the subdirectory created under os.UserConfigDir() for the file
// fallback.
const dirName = "relio"

// fileName is the token file within Dir().
const fileName = "github_token"

// keyringBackend abstracts the OS keychain so tests never touch a real one.
type keyringBackend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// osKeyringBackend wraps go-keyring's package-level functions, which talk to
// the real OS keychain.
type osKeyringBackend struct{}

func (osKeyringBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (osKeyringBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (osKeyringBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

// backend is the keyring implementation in use; overridable in tests so they
// can simulate an unavailable or erroring keychain without touching a real
// one.
var backend keyringBackend = osKeyringBackend{}

// Dir returns the directory the file fallback lives in. It does not create
// the directory.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, dirName), nil
}

// Path returns the full path to the fallback token file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// Store saves token, preferring the OS keychain. If the keychain is
// unavailable or returns any error, Store falls back to writing the
// protected local file instead.
//
// Whichever backend ends up holding the token is the only one that does: a
// successful keychain write removes any stale fallback file left over from a
// prior run where the keychain was unavailable, so the token is never live
// in both places where the copies could drift out of sync. The cleanup is
// best-effort -- Store has already succeeded once the keychain write lands,
// so a failure to remove the stale file does not fail Store.
func Store(token string) error {
	if err := backend.Set(serviceName, accountName, token); err == nil {
		_ = removeFile()
		return nil
	}
	return storeFile(token)
}

// Load resolves the current token. It checks the OS keychain first; if
// found, it returns (token, "keychain", nil) without touching the fallback
// file at all. If the keychain errors or has nothing stored, Load tries the
// fallback file next. If neither backend has a token, Load returns
// ("", "", nil) -- "not authenticated" is not an error condition, mirroring
// ghrelease.Token()'s convention of signaling "nothing found" with zero
// values rather than an error.
//
// A real failure reading the fallback file (e.g. a permissions problem, as
// opposed to the file simply not existing) is surfaced as err, since that
// indicates something is wrong rather than "the user hasn't logged in yet".
func Load() (token, source string, err error) {
	if v, kerr := backend.Get(serviceName, accountName); kerr == nil && v != "" {
		return v, "keychain", nil
	}

	data, ferr := loadFile()
	if ferr != nil {
		if os.IsNotExist(ferr) {
			return "", "", nil
		}
		return "", "", ferr
	}
	if data == "" {
		return "", "", nil
	}
	return data, "file", nil
}

// Delete clears the token from both backends. Keychain errors -- including
// "not found" and a keychain backend that is unavailable entirely, as on
// many WSL2/headless Linux setups -- are expected in a hybrid store and
// never fail Delete; they simply mean that backend had nothing to clear.
// Only a genuine failure removing the fallback file (something other than
// the file not existing) is treated as a real, reportable error.
func Delete() error {
	_ = backend.Delete(serviceName, accountName)

	if err := removeFile(); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// storeFile writes token to the protected fallback file, creating the
// directory (mode 0700) on first use. It writes to a temp file in the same
// directory and renames it into place (mode 0600), mirroring
// userconfig.Save's atomic-write technique so an interrupted write can never
// leave a partial or corrupt token behind.
func storeFile(token string) error {
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

	tmp, err := os.CreateTemp(dir, ".github_token-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(token); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, p); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// loadFile reads the raw token from the fallback file. Unlike
// userconfig.Load, this does not swallow every error into a zero value: a
// missing file (os.IsNotExist) is "not authenticated" and is handled by the
// caller, but any other error (corruption is not applicable to a plain-text
// file, but permission issues are) is returned so it isn't mistaken for
// "never logged in".
func loadFile() (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// removeFile deletes the fallback file if present. A missing file is not an
// error.
func removeFile() error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
