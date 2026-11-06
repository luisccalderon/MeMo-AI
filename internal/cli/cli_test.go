package cli

import "testing"

func TestParseFlagsAndCommand(t *testing.T) {
	inv, err := parse([]string{"install", "--lang=en", "--agent", "build", "--model=ollama/qwen3:8b", "--no-subagents", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.command != "install" {
		t.Fatalf("command = %q", inv.command)
	}
	if inv.opts.lang != "en" || inv.opts.agent != "build" || inv.opts.model != "ollama/qwen3:8b" {
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

func TestParseErrors(t *testing.T) {
	if _, err := parse([]string{"--unknown"}); err == nil {
		t.Fatal("expected unknown flag error")
	}
	if _, err := parse([]string{"--agent"}); err == nil {
		t.Fatal("expected missing value error")
	}
}
