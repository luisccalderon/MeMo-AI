package promptblock

import (
	"strings"
	"testing"
)

func TestUpsertAppendsAndPreservesUserText(t *testing.T) {
	user := "# My rules\n\nAlways use tabs.\n"
	got := Upsert(user, "orchestrator v1")
	if !strings.HasPrefix(got, user) {
		t.Fatalf("user text not preserved:\n%s", got)
	}
	if !Has(got) || !strings.Contains(got, "orchestrator v1") {
		t.Fatalf("block missing:\n%s", got)
	}
}

func TestUpsertReplacesExistingBlockInPlace(t *testing.T) {
	first := Upsert("before\n", "v1") + "after\n"
	second := Upsert(first, "v2")
	if strings.Contains(second, "v1") || !strings.Contains(second, "v2") {
		t.Fatalf("block not replaced:\n%s", second)
	}
	if strings.Count(second, Begin) != 1 {
		t.Fatalf("expected one block:\n%s", second)
	}
	if !strings.HasPrefix(second, "before\n") || !strings.HasSuffix(second, "after\n") {
		t.Fatalf("surrounding text changed:\n%s", second)
	}
	if Upsert(second, "v2") != second {
		t.Fatal("upsert should be idempotent")
	}
}

func TestStripRestoresUserText(t *testing.T) {
	user := "# My rules\n\nAlways use tabs.\n"
	got, found := Strip(Upsert(user, "body"))
	if !found || got != user {
		t.Fatalf("found = %v, got %q, want %q", found, got, user)
	}
}

func TestStripOnlyBlockLeavesEmpty(t *testing.T) {
	got, found := Strip(Upsert("", "body"))
	if !found || got != "" {
		t.Fatalf("found = %v, got %q", found, got)
	}
}

func TestStripWithoutBlockIsNoop(t *testing.T) {
	const content = "plain\n" + beginPrefix + " unterminated\n"
	got, found := Strip(content)
	if found || got != content {
		t.Fatalf("found = %v, got %q", found, got)
	}
}
