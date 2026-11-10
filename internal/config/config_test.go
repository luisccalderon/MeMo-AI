package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripJSONC(t *testing.T) {
	raw := []byte(`{
  // line comment
  "url": "https://example.com/a//b",
  "escaped": "quote \" // not a comment",
  /* block
     comment */
  "list": [1, 2, 3,],
  "nested": {"ok": true,},
}`)
	got := string(StripJSONC(raw))
	want := `{
  
  "url": "https://example.com/a//b",
  "escaped": "quote \" // not a comment",
  
  "list": [1, 2, 3],
  "nested": {"ok": true}
}`
	if got != want {
		t.Fatalf("unexpected JSONC output:\n%s\nwant:\n%s", got, want)
	}
}

func TestLoadJSONCAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.jsonc")
	if err := os.WriteFile(path, []byte(`{
  // keep me out
  "model": "ollama/qwen3:8b",
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := GetString(doc.Data, "model"); got != "ollama/qwen3:8b" {
		t.Fatalf("model = %q", got)
	}
	Set(doc.Data, "memo-orchestrator", "agents", "memo-orchestrator", "mode")
	if got := GetString(doc.Data, "agents", "memo-orchestrator", "mode"); got != "memo-orchestrator" {
		t.Fatalf("nested set = %q", got)
	}
	if err := doc.Save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := GetString(reloaded.Data, "agents", "memo-orchestrator", "mode"); got != "memo-orchestrator" {
		t.Fatalf("reloaded nested set = %q", got)
	}
}

func TestLoadMissingReturnsEmptyDocument(t *testing.T) {
	doc, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if doc == nil || doc.Data == nil || len(doc.Data) != 0 {
		t.Fatalf("expected empty document, got %#v", doc)
	}
}

func TestGetDeleteEnsureMap(t *testing.T) {
	root := map[string]any{}
	m := EnsureMap(root, "mcp", "servers")
	m["engram"] = map[string]any{"type": "local"}
	if got := GetString(m, "engram", "type"); got != "local" {
		t.Fatalf("type = %q", got)
	}
	Delete(root, "mcp", "servers", "engram")
	if _, ok := Get(root, "mcp", "servers", "engram"); ok {
		t.Fatal("engram should be deleted")
	}
}

func TestDeepMerge(t *testing.T) {
	dst := map[string]any{"a": map[string]any{"x": 1, "y": 2}}
	src := map[string]any{"a": map[string]any{"y": 3}, "b": true}
	DeepMerge(dst, src)
	if got, _ := Get(dst, "a", "y"); got != 3 {
		t.Fatalf("y = %v", got)
	}
	if got, _ := Get(dst, "a", "x"); got != 1 {
		t.Fatalf("x = %v", got)
	}
	if got, _ := Get(dst, "b"); got != true {
		t.Fatalf("b = %v", got)
	}
}
