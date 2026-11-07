package main

import "testing"

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
