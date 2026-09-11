// Package i18n is a small, zero-dependency translation catalog for the CLI's
// interactive and help output. It is process-global and resolved once at
// startup (see cmd/language.go), then consumed by every rendering package
// through T.
//
// The active catalog lives in an atomic.Value rather than a plain package var
// or a mutex: the Settings screen mutates the language while a Bubble Tea
// program is mid-render, so writer and readers genuinely race. atomic.Value
// gives lock-free reads and a -race-clean swap.
package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// MessageID identifies one translatable string. Constants live in keys.go,
// grouped by the surface that uses them.
type MessageID string

// Language is one entry in the language registry — the single source of
// truth for the Settings screen and for validating a configured language id.
type Language struct {
	ID   string // e.g. "en"
	Name string // the endonym; never translated, e.g. "English", "Español"
}

// languages is the ordered registry. Order controls Settings row order.
var languages = []Language{
	{ID: "en", Name: "English"},
	{ID: "es", Name: "Español"},
}

// catalogs maps a language id to its message catalog. en is the reference
// set: every other catalog MUST define every key en defines (catalog_test.go
// enforces this).
var catalogs = map[string]map[MessageID]string{
	"en": en,
	"es": es,
}

// active holds the currently active catalog (map[MessageID]string). currentLang
// holds its language id (string). Both are atomic.Value so T and SetLanguage
// never take a lock.
var (
	active      atomic.Value
	currentLang atomic.Value
)

func init() {
	active.Store(catalogs["en"])
	currentLang.Store("en")
}

// Languages returns the ordered language registry. The returned slice is a
// copy; callers may not mutate the registry through it.
func Languages() []Language {
	out := make([]Language, len(languages))
	copy(out, languages)
	return out
}

// Current returns the active language id.
func Current() string {
	if v, ok := currentLang.Load().(string); ok && v != "" {
		return v
	}
	return "en"
}

// SetLanguage normalizes id (trim + lowercase), swaps the active catalog, and
// returns the id actually applied. ok is false when id is not registered; the
// active language is then "en". Safe to call concurrently with T.
func SetLanguage(id string) (resolved string, ok bool) {
	norm := strings.ToLower(strings.TrimSpace(id))
	cat, found := catalogs[norm]
	if !found {
		active.Store(catalogs["en"])
		currentLang.Store("en")
		return "en", false
	}
	active.Store(cat)
	currentLang.Store(norm)
	return norm, true
}

// T looks up id in the active catalog, falls back to the English catalog,
// then falls back to string(id) — never a panic, never an empty string. With
// no args the template is returned verbatim, so a stray '%' in a translation
// can never become "%!s(MISSING)"; with args it is fmt.Sprintf'd.
//
// A missing key falls back silently: writing to stderr mid-frame would
// corrupt Bubble Tea's rendering and pollute piped output. Coverage is
// enforced by catalog_test.go instead.
func T(id MessageID, args ...any) string {
	cat, _ := active.Load().(map[MessageID]string)
	tmpl, ok := cat[id]
	if !ok {
		tmpl, ok = catalogs["en"][id]
	}
	if !ok {
		return string(id)
	}
	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}
