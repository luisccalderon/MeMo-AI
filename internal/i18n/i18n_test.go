package i18n

import "testing"

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
