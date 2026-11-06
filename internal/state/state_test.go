package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadAndRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memo-ai", "state.json")
	st := &State{Language: "en", PrimaryAgent: "memo-orchestrator"}
	st.Record("/tmp/a.md", "abc", "agent")
	st.Record("/tmp/a.md", "def", "agent")
	st.Record("/tmp/b.md", "ghi", "command")
	if len(st.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(st.Files))
	}
	if err := st.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PrimaryAgent != "memo-orchestrator" || len(loaded.Files) != 2 {
		t.Fatalf("unexpected state: %#v", loaded)
	}
	record, ok := loaded.Find("/tmp/a.md")
	if !ok || record.SHA256 != "def" {
		t.Fatalf("record = %#v, ok = %v", record, ok)
	}
	if loaded.InstalledAt == "" || loaded.UpdatedAt == "" {
		t.Fatal("timestamps should be set")
	}
}

func TestLoadMissing(t *testing.T) {
	st, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st == nil || st.Version != Version || st.InstalledAt != "" {
		t.Fatalf("unexpected state: %#v", st)
	}
}

func TestHashFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
	if _, err := HashFile(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
