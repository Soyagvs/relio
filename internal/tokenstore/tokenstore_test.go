package tokenstore

import (
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// fakeKeyring simulates the OS keychain for tests, so they never touch a
// real macOS Keychain / Windows Credential Manager / Linux Secret Service.
// Setting err simulates a keychain backend that is unavailable or errors on
// every call (e.g. no Secret Service provider, as on many WSL2/headless
// Linux setups).
type fakeKeyring struct {
	stored map[string]string
	err    error
	// deleteErr, when set, makes Delete fail without removing the entry --
	// simulating a genuine keychain deletion failure (as opposed to err,
	// which simulates the backend being unavailable for every call).
	deleteErr error
}

func (f *fakeKeyring) key(service, user string) string { return service + "\x00" + user }

func (f *fakeKeyring) Set(service, user, password string) error {
	if f.err != nil {
		return f.err
	}
	f.stored[f.key(service, user)] = password
	return nil
}

func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.stored[f.key(service, user)]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	if f.err != nil {
		return f.err
	}
	if f.deleteErr != nil {
		return f.deleteErr
	}
	k := f.key(service, user)
	if _, ok := f.stored[k]; !ok {
		return keyring.ErrNotFound
	}
	delete(f.stored, k)
	return nil
}

// withFakeBackend swaps the package-level keyring backend for f for the
// duration of the test.
func withFakeBackend(t *testing.T, f *fakeKeyring) {
	t.Helper()
	prev := backend
	backend = f
	t.Cleanup(func() { backend = prev })
}

func TestStoreFallsBackToFileWhenKeyringUnavailable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakeBackend(t, &fakeKeyring{stored: map[string]string{}, err: errors.New("no keyring backend available")})

	if err := Store("tok-123"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("expected fallback file to exist: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 0600", perm)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "tok-123" {
		t.Errorf("file content = %q, want %q", data, "tok-123")
	}

	tok, source, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok != "tok-123" || source != "file" {
		t.Errorf("Load() = (%q, %q), want (%q, %q)", tok, source, "tok-123", "file")
	}
}

func TestStoreUsesKeyringDoesNotCreateFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakeBackend(t, &fakeKeyring{stored: map[string]string{}})

	if err := Store("tok-789"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("expected no fallback file to be created, stat err = %v", err)
	}

	tok, source, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok != "tok-789" || source != "keychain" {
		t.Errorf("Load() = (%q, %q), want (%q, %q)", tok, source, "tok-789", "keychain")
	}
}

func TestStoreUsesKeyringAndCleansUpStaleFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	fk := &fakeKeyring{stored: map[string]string{}}
	withFakeBackend(t, fk)

	// Simulate a stale fallback file left from a prior keyring-unavailable run.
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Store("tok-456"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("expected stale fallback file to be removed, stat err = %v", err)
	}

	tok, source, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok != "tok-456" || source != "keychain" {
		t.Errorf("Load() = (%q, %q), want (%q, %q)", tok, source, "tok-456", "keychain")
	}
}

func TestLoadWithNothingStoredReturnsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakeBackend(t, &fakeKeyring{stored: map[string]string{}})

	tok, source, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok != "" || source != "" {
		t.Errorf(`Load() = (%q, %q), want ("", "")`, tok, source)
	}
}

func TestDeleteClearsBothBackends(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	fk := &fakeKeyring{stored: map[string]string{}}
	withFakeBackend(t, fk)

	if err := fk.Set(serviceName, accountName, "tok"); err != nil {
		t.Fatal(err)
	}
	if err := storeFile("tok"); err != nil {
		t.Fatal(err)
	}

	if err := Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := fk.Get(serviceName, accountName); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("expected keyring entry to be deleted, got err = %v", err)
	}
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("expected fallback file to be deleted, stat err = %v", err)
	}
}

func TestDeleteReturnsErrorWhenKeychainDeletionActuallyFails(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	fk := &fakeKeyring{stored: map[string]string{}, deleteErr: errors.New("transient D-Bus failure")}
	withFakeBackend(t, fk)
	if err := fk.Set(serviceName, accountName, "tok"); err != nil {
		t.Fatal(err)
	}

	if err := Delete(); err == nil {
		t.Fatal("Delete() = nil, want an error: the token is still present in the keychain after the failed deletion")
	}

	if v, err := fk.Get(serviceName, accountName); err != nil || v != "tok" {
		t.Fatalf("expected the keychain entry to remain untouched, got (%q, %v)", v, err)
	}
}

func TestDeleteWithNothingStoredDoesNotError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakeBackend(t, &fakeKeyring{stored: map[string]string{}})

	if err := Delete(); err != nil {
		t.Errorf("Delete() with nothing stored = %v, want nil", err)
	}
}
