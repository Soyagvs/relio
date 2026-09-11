package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"testing"
)

// TestCatalogKeyParity asserts every non-English catalog defines exactly the
// same key set as en, the reference set. A key present in en but missing
// elsewhere silently falls back at runtime (by design) — this test is what
// actually catches the drift.
func TestCatalogKeyParity(t *testing.T) {
	want := keySet(en)
	for lang, cat := range catalogs {
		if lang == "en" {
			continue
		}
		got := keySet(cat)
		for k := range want {
			if _, ok := got[k]; !ok {
				t.Errorf("catalog %q is missing key %q, present in en", lang, k)
			}
		}
		for k := range got {
			if _, ok := want[k]; !ok {
				t.Errorf("catalog %q has key %q, not present in en", lang, k)
			}
		}
	}
}

func keySet(cat map[MessageID]string) map[MessageID]struct{} {
	s := make(map[MessageID]struct{}, len(cat))
	for k := range cat {
		s[k] = struct{}{}
	}
	return s
}

var (
	literalPercentRE = regexp.MustCompile(`%%`)
	verbRE           = regexp.MustCompile(`%[^%]`)
)

// countVerbs counts fmt verbs in s, ignoring the literal "%%" escape.
func countVerbs(s string) int {
	stripped := literalPercentRE.ReplaceAllString(s, "")
	return len(verbRE.FindAllString(stripped, -1))
}

// TestCatalogVerbArityParity asserts every non-English translation of a key
// references the same number of fmt verbs as the English template. A
// translation that drops or adds an argument would panic or silently mis-render
// at Sprintf time; this test catches it before it ships.
func TestCatalogVerbArityParity(t *testing.T) {
	for id, enTmpl := range en {
		want := countVerbs(enTmpl)
		for lang, cat := range catalogs {
			if lang == "en" {
				continue
			}
			tmpl, ok := cat[id]
			if !ok {
				continue // TestCatalogKeyParity already reports the missing key
			}
			if got := countVerbs(tmpl); got != want {
				t.Errorf("catalog %q key %q has %d fmt verbs, en has %d\n  en: %q\n  %s: %q",
					lang, id, got, want, enTmpl, lang, tmpl)
			}
		}
	}
}

// TestKeysDeclaredInASTExistInEnglishCatalog parses keys.go directly (not
// through the compiled package) and asserts every MessageID constant's
// literal string value is a key in the English catalog. This closes the
// residual hole plain review can't: a const declared but never added to any
// catalog compiles fine and silently falls back to its raw id at runtime.
func TestKeysDeclaredInASTExistInEnglishCatalog(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "keys.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing keys.go: %v", err)
	}

	var ids []string
	ast.Inspect(file, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, v := range vs.Values {
			lit, ok := v.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				continue
			}
			ids = append(ids, val)
		}
		return true
	})

	if len(ids) == 0 {
		t.Fatal("no MessageID constants found in keys.go — the AST coverage check found nothing to check")
	}
	for _, id := range ids {
		if _, ok := en[MessageID(id)]; !ok {
			t.Errorf("keys.go declares MessageID %q, not present in the English catalog", id)
		}
	}
}
