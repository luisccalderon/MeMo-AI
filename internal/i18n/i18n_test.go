package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLanguagesDefineTheSameKeys(t *testing.T) {
	for _, lang := range []string{English, Spanish} {
		other := English
		if lang == English {
			other = Spanish
		}
		for key := range messages[lang] {
			if _, ok := messages[other][key]; !ok {
				t.Errorf("key %q exists in %s but not in %s", key, lang, other)
			}
		}
	}
}

// T falls back to the raw key, so a missing entry would reach users as
// "agents.subagents_unsupported". Every literal key passed to T or Tf must
// exist in every language.
func TestEveryKeyUsedInCodeIsDefined(t *testing.T) {
	call := regexp.MustCompile(`i18n\.Tf?\(\s*[\w.]+\s*,\s*"([^"]+)"`)
	root := filepath.Join("..", "..")
	seen := 0
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range call.FindAllStringSubmatch(string(raw), -1) {
			seen++
			for _, lang := range []string{English, Spanish} {
				if _, ok := messages[lang][match[1]]; !ok {
					t.Errorf("%s uses i18n key %q, which %s does not define", path, match[1], lang)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen < 20 {
		t.Fatalf("scanner found only %d i18n calls; the pattern is probably stale", seen)
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"en":      English,
		"EN":      English,
		"en-US":   English,
		"English": English,
		"es":      Spanish,
		"es-MX":   Spanish,
		"Español": Spanish,
		"unknown": Spanish,
		"":        Spanish,
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTranslateFallback(t *testing.T) {
	if got := T("en", "app.tagline"); got != "Your coding agent, bringing your changes, tasks, and engineering workflow together." {
		t.Fatalf("tagline = %q", got)
	}
	if got := T("en", "missing.key"); got != "missing.key" {
		t.Fatalf("fallback = %q", got)
	}
	if got := Tf("es", "error.unknown_command", "foo"); got != "Comando desconocido: foo" {
		t.Fatalf("formatted = %q", got)
	}
}
