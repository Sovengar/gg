package verbs

import (
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	e, ok := Lookup("st")
	if !ok {
		t.Fatal("st not found")
	}
	if e.IsFunc() {
		t.Error("st should be a git argv verb")
	}
	got := e.Argv([]string{"-u"})
	want := []string{"status", "--short", "-u"}
	if len(got) != len(want) {
		t.Fatalf("argv = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv = %q, want %q", got, want)
		}
	}
}

func TestFuncVerbs(t *testing.T) {
	for _, name := range []string{"log", "dash", "push", "update", "sync", "uncommit"} {
		e, ok := Lookup(name)
		if !ok || !e.IsFunc() || e.Fn == nil {
			t.Errorf("%s should be a Go function", name)
		}
	}
}

func TestRegistryComplete(t *testing.T) {
	names := Names()
	if len(names) < 30 {
		t.Fatalf("only %d verbs registered", len(names))
	}
	for _, name := range names {
		if _, ok := Lookup(name); !ok {
			t.Errorf("%s is listed but does not resolve", name)
		}
	}
}

func TestMutating(t *testing.T) {
	if !IsMutating("c") || !IsMutating("push") {
		t.Error("commit and push must be mutating")
	}
	if IsMutating("st") {
		t.Error("st must not be mutating")
	}
}

func TestToolEntries(t *testing.T) {
	e, ok := Lookup("story")
	if !ok || !e.IsTool() {
		t.Fatalf("story should be a tool verb, got ok=%v entry=%+v", ok, e)
	}
	if e.Tool != "gitlogue" {
		t.Errorf("tool = %q, want gitlogue", e.Tool)
	}
	if e.IsFunc() || e.IsTool() && e.Tool == "" {
		t.Error("story must resolve to a tool binary")
	}
	if got := e.Describe(); !strings.Contains(got, "gitlogue") {
		t.Errorf("describe = %q", got)
	}
}
