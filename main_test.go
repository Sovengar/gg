package main

import (
	"strings"
	"testing"

	"gg/internal/verbs"
)

func TestQuote(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"status", "--short"}, "status --short"},
		{[]string{"commit", "-m", "two words"}, "commit -m 'two words'"},
		{[]string{"log", "--grep=it's"}, `log '--grep=it'\''s'`},
		{nil, ""},
	}
	for _, c := range cases {
		if got := quote(c.args); got != c.want {
			t.Errorf("quote(%q) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestBranchNewIsNotAChangeID(t *testing.T) {
	got, rc := resolveArgs("br", []string{"new", "side"})
	if rc != 0 || len(got) != 2 || got[0] != "new" || got[1] != "side" {
		t.Fatalf("resolveArgs(br, new side) = %q rc %d, want [new side]", got, rc)
	}
}

func TestParseFlags(t *testing.T) {
	m, args := parseFlags([]string{"-p", "st", "--dry"})
	if m != modePrint {
		t.Errorf("mode = %v, want print", m)
	}
	if len(args) != 1 || args[0] != "st" {
		t.Errorf("args = %q, want [st]", args)
	}

	m, args = parseFlags([]string{"st", "-u"})
	if m != modeExec {
		t.Errorf("mode = %v, want exec", m)
	}
	if len(args) != 2 {
		t.Errorf("args = %q, want [st -u]", args)
	}
}

func TestWrapItems(t *testing.T) {
	got := wrapItems([]string{"status (st)", "log", "diff"}, 20)
	want := []string{"status (st) · log", "diff"}
	if len(got) != len(want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}

	// the middle dot is multi-byte: the limit counts runes, not bytes
	got = wrapItems([]string{"status", "log", "br -r", "br -a"}, 28)
	if len(got) != 1 {
		t.Errorf("multi-byte separator wrapped early: %q", got)
	}

	// a single item wider than the limit still prints whole
	got = wrapItems([]string{"reword (rename, describe, desc)", "split"}, 20)
	if got[0] != "reword (rename, describe, desc)" || got[1] != "split" {
		t.Errorf("oversized item = %q", got)
	}
}

func TestTermWidth(t *testing.T) {
	t.Setenv("COLUMNS", "120")
	if n := termWidth(); n != 120 {
		t.Errorf("width = %d, want 120", n)
	}
	t.Setenv("COLUMNS", "junk")
	if n := termWidth(); n != 80 {
		t.Errorf("width = %d, want the 80 fallback", n)
	}
}

func TestHelpLines(t *testing.T) {
	cases := []struct {
		verb    string
		display string
		aliases []string
	}{
		{"status", "status (st)", nil},
		{"branch", "branch (br)", nil},
		{"reword", "reword (desc)", []string{"rename", "describe"}},
	}
	for _, c := range cases {
		e, ok := verbs.Lookup(c.verb)
		if !ok {
			t.Fatalf("%s not found", c.verb)
		}
		if got := e.Display(); got != c.display {
			t.Errorf("%s Display() = %q, want %q", c.verb, got, c.display)
		}
		got := strings.Join(e.Aliases, ",")
		if got != strings.Join(c.aliases, ",") {
			t.Errorf("%s aliases = %q, want %q", c.verb, got, c.aliases)
		}
	}
}
