package verbs

import (
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	e, ok := Lookup("log")
	if !ok {
		t.Fatal("log not found")
	}
	if !e.IsFunc() {
		t.Error("log should be a Go function")
	}

	git, ok := Lookup("reflog")
	if !ok {
		t.Fatal("reflog not found")
	}
	got := git.Argv([]string{"--oneline"})
	want := []string{"reflog", "--oneline"}
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
	for _, name := range []string{"log", "dash", "status", "diff", "push", "update", "sync", "uncommit", "init", "clone", "rebase", "tag", "remote", "brlm", "land"} {
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

func TestReviewToolVerbs(t *testing.T) {
	tools := map[string]string{
		"hunk": "hunk", "tuicr": "tuicr", "but": "but", "jj": "jj",
		"lazygit": "lazygit", "prdash": "prdash", "gitdash": "gitdash",
	}
	for name, binary := range tools {
		e, ok := Lookup(name)
		if !ok || !e.IsTool() {
			t.Errorf("%s should be a tool verb, got ok=%v entry=%+v", name, ok, e)
			continue
		}
		if e.Tool != binary {
			t.Errorf("%s tool = %q, want %q", name, e.Tool, binary)
		}
	}
}

func TestConflictVerb(t *testing.T) {
	for _, name := range []string{"conflict", "cf"} {
		e, ok := Lookup(name)
		if !ok || e.IsFunc() || e.IsTool() {
			t.Fatalf("%s should be a git verb, got ok=%v entry=%+v", name, ok, e)
		}
		if got := e.Argv(nil); len(got) != 1 || got[0] != "mergetool" {
			t.Errorf("%s argv = %q, want [mergetool]", name, got)
		}
	}
}

func TestPRVerbsRegistered(t *testing.T) {
	for _, name := range []string{"pr", "rpr", "epr"} {
		e, ok := Lookup(name)
		if !ok || !e.IsFunc() || e.Fn == nil {
			t.Errorf("%s should be a Go function", name)
		}
	}
	for _, name := range []string{"pr", "rpr", "epr"} {
		if !IsMutating(name) {
			t.Errorf("%s should be mutating", name)
		}
	}
	// the raw reset verb is gone: it must not dispatch to an entry of its own
	if _, ok := Lookup("re"); ok {
		t.Error("re should no longer be a verb")
	}
	if IsMutating("re") {
		t.Error("re should no longer be mutating")
	}
}
