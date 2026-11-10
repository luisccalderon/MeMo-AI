package cli

import "testing"

func TestParseFlagsAndCommand(t *testing.T) {
	inv, err := parse([]string{"install", "--lang=en", "--model=ollama/qwen3:8b", "--no-subagents", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.command != "install" {
		t.Fatalf("command = %q", inv.command)
	}
	if inv.opts.lang != "en" || inv.opts.model != "ollama/qwen3:8b" {
		t.Fatalf("unexpected options: %#v", inv.opts)
	}
	if !inv.opts.noSubagents || !inv.opts.yes {
		t.Fatalf("boolean flags not set: %#v", inv.opts)
	}
}

func TestParseSubcommandArgs(t *testing.T) {
	inv, err := parse([]string{"engram", "install", "--install-engram"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.command != "engram" || len(inv.args) != 1 || inv.args[0] != "install" {
		t.Fatalf("unexpected invocation: %#v", inv)
	}
	if !inv.opts.installEngram {
		t.Fatal("install-engram should be set")
	}
}

func TestParseSubagentFlagsStayAcceptedForExistingScripts(t *testing.T) {
	inv, err := parse([]string{"install", "--yes", "--no-subagents", "--subagent-model", "ollama/qwen3:8b"})
	if err != nil {
		t.Fatalf("subagent flags must not break existing scripts: %v", err)
	}
	if !inv.opts.noSubagents || inv.opts.subagentModel != "ollama/qwen3:8b" {
		t.Fatalf("subagent flags not parsed: %#v", inv.opts)
	}
}

func TestParsePiFlags(t *testing.T) {
	inv, err := parse([]string{"install", "--no-orchestrator", "--no-extension"})
	if err != nil {
		t.Fatal(err)
	}
	if !inv.opts.noOrchestrator || !inv.opts.noExtension {
		t.Fatalf("pi flags not set: %#v", inv.opts)
	}
	if _, err := parse([]string{"install", "--agent", "build"}); err == nil {
		t.Fatal("--agent was removed with the OpenCode backend and must be rejected")
	}
}

func TestParseQuietStartupFlags(t *testing.T) {
	inv, err := parse([]string{"install", "--quiet-startup", "--no-quiet-startup"})
	if err != nil {
		t.Fatal(err)
	}
	if !inv.opts.quietStartup || !inv.opts.noQuietStartup {
		t.Fatalf("quiet startup flags not set: %#v", inv.opts)
	}
}

func TestParseAtelierFlags(t *testing.T) {
	inv, err := parse([]string{"install", "--atelier", "--no-atelier"})
	if err != nil {
		t.Fatal(err)
	}
	if !inv.opts.atelier || !inv.opts.noAtelier {
		t.Fatalf("atelier flags not set: %#v", inv.opts)
	}
}

func TestParseAtelierConfigFlag(t *testing.T) {
	inv, err := parse([]string{"install", "--atelier-config", "C:/tmp/mine.json"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.opts.atelierConfig != "C:/tmp/mine.json" {
		t.Fatalf("atelier-config = %q", inv.opts.atelierConfig)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := parse([]string{"--unknown"}); err == nil {
		t.Fatal("expected unknown flag error")
	}
	if _, err := parse([]string{"--model"}); err == nil {
		t.Fatal("expected missing value error")
	}
}
