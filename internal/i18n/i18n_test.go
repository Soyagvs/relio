package i18n

import (
	"sync"
	"testing"
)

// swapCatalogs temporarily replaces the package's active catalog set with fake
// so tests can exercise T's fallback chain without depending on real catalog
// content (and without risking corruption of the real English reference set).
// The caller MUST call the returned restore func, typically via defer.
func swapCatalogs(t *testing.T, fake map[string]map[MessageID]string) (restore func()) {
	t.Helper()
	oldCatalogs := catalogs
	oldActive := active.Load()
	oldCurrent := currentLang.Load()

	catalogs = fake
	active.Store(fake["en"])
	currentLang.Store("en")

	return func() {
		catalogs = oldCatalogs
		if oldActive != nil {
			active.Store(oldActive)
		}
		if oldCurrent != nil {
			currentLang.Store(oldCurrent)
		}
	}
}

func TestTReturnsActiveLanguageString(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"greet": "Hello, %s"},
		"es": {"greet": "Hola, %s"},
	})
	defer restore()

	if _, ok := SetLanguage("es"); !ok {
		t.Fatal("SetLanguage(es) rejected a registered language")
	}
	if got := T("greet", "Ana"); got != "Hola, Ana" {
		t.Errorf("T(greet, Ana) = %q, want %q", got, "Hola, Ana")
	}
}

func TestTFallsBackToEnglishWhenActiveMissesKey(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"greet": "Hello"},
		"es": {}, // deliberately missing "greet"
	})
	defer restore()

	SetLanguage("es")
	if got := T("greet"); got != "Hello" {
		t.Errorf("T(greet) = %q, want English fallback %q", got, "Hello")
	}
}

func TestTFallsBackToRawIDWhenEverywhereMisses(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {},
		"es": {},
	})
	defer restore()

	SetLanguage("es")
	if got := T("nope"); got != "nope" {
		t.Errorf("T(nope) = %q, want raw id %q", got, "nope")
	}
}

func TestTWithNoArgsReturnsTemplateVerbatim(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"pct": "100%!"},
		"es": {"pct": "100%!"},
	})
	defer restore()

	if got := T("pct"); got != "100%!" {
		t.Errorf("T(pct) = %q, want the template verbatim (no Sprintf when there are no args)", got)
	}
}

func TestSetLanguageValidCode(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"k": "v-en"},
		"es": {"k": "v-es"},
	})
	defer restore()

	resolved, ok := SetLanguage("es")
	if !ok || resolved != "es" {
		t.Fatalf("SetLanguage(es) = (%q, %v), want (es, true)", resolved, ok)
	}
	if Current() != "es" {
		t.Errorf("Current() = %q, want es", Current())
	}
}

func TestSetLanguageUnknownCodeFallsBackToEnglish(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"k": "v-en"},
		"es": {"k": "v-es"},
	})
	defer restore()

	SetLanguage("es")

	resolved, ok := SetLanguage("fr")
	if ok {
		t.Fatal("SetLanguage(fr) ok = true, want false (fr is not registered)")
	}
	if resolved != "en" {
		t.Errorf("resolved = %q, want en", resolved)
	}
	if Current() != "en" {
		t.Errorf("Current() = %q, want en after an unknown language id", Current())
	}
}

func TestSetLanguageNormalizesCase(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"k": "v-en"},
		"es": {"k": "v-es"},
	})
	defer restore()

	if resolved, ok := SetLanguage("  ES  "); !ok || resolved != "es" {
		t.Fatalf("SetLanguage(\"  ES  \") = (%q, %v), want (es, true)", resolved, ok)
	}
}

// TestSetLanguageRaceSafe swaps the active language on one goroutine while
// another reads through T, run under `go test -race`. Because the Settings
// screen mutates language while Bubble Tea is mid-render, writer and readers
// genuinely race in production; this proves the atomic.Value swap never
// produces a torn read.
func TestSetLanguageRaceSafe(t *testing.T) {
	restore := swapCatalogs(t, map[string]map[MessageID]string{
		"en": {"k": "v-en"},
		"es": {"k": "v-es"},
	})
	defer restore()

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				SetLanguage("es")
				SetLanguage("en")
			}
		}
	}()

	for i := 0; i < 2000; i++ {
		if got := T("k"); got != "v-en" && got != "v-es" {
			t.Fatalf("T(k) = %q, want v-en or v-es (torn read)", got)
		}
	}
	close(done)
	wg.Wait() // ensure the writer goroutine has fully stopped before restore() runs
}
