package ops

import (
	"strings"
	"testing"
)

func TestLandArgs(t *testing.T) {
	cases := []struct {
		args   []string
		branch string
		target string
		yes    bool
		stack  bool
		bad    bool
	}{
		{nil, "", "", false, false, false},
		{[]string{"side"}, "side", "", false, false, false},
		{[]string{"side", "-y"}, "side", "", true, false, false},
		{[]string{"--yes", "side"}, "side", "", true, false, false},
		{[]string{"side", "-b", "main"}, "side", "main", false, false, false},
		{[]string{"-b", "release", "-y", "side"}, "side", "release", true, false, false},
		{[]string{"--onto", "main", "side"}, "side", "main", false, false, false},
		{[]string{"base", "--stack"}, "base", "", false, true, false},
		{[]string{"-s", "base", "-y"}, "base", "", true, true, false},
		{[]string{"side", "other"}, "", "", false, false, true},
		{[]string{"side", "-b"}, "", "", false, false, true},
		{[]string{"side", "-b", "a", "--onto", "b"}, "", "", false, false, true},
		{[]string{"--nope"}, "", "", false, false, true},
	}
	for _, c := range cases {
		branch, target, yes, stack, err := landArgs(c.args)
		if c.bad {
			if err == nil {
				t.Errorf("landArgs(%q) = %q, %q, %v, %v; want an error", c.args, branch, target, yes, stack)
			}
			continue
		}
		if err != nil {
			t.Errorf("landArgs(%q) failed: %v", c.args, err)
			continue
		}
		if branch != c.branch || target != c.target || yes != c.yes || stack != c.stack {
			t.Errorf("landArgs(%q) = %q, %q, %v, %v; want %q, %q, %v, %v",
				c.args, branch, target, yes, stack, c.branch, c.target, c.yes, c.stack)
		}
	}
}

func TestUnlandedChildrenBlocksABase(t *testing.T) {
	dir := prRepo(t)
	prInDir(t, dir)
	prGit(t, dir, "switch", "-q", "-c", "base")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "base")
	prGit(t, dir, "switch", "-q", "-c", "child")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "child")

	kids := stackedChildren("base", "main")
	if len(kids) != 1 || kids[0] != "child" {
		t.Fatalf("stackedChildren(base) = %q; want [child]", kids)
	}
	if unlanded := unlandedChildren(kids, "main"); len(unlanded) != 1 || unlanded[0] != "child" {
		t.Fatalf("unlandedChildren = %q; want [child]", unlanded)
	}
	if kids := stackedChildren("child", "main"); len(kids) != 0 {
		t.Fatalf("stackedChildren(child) = %q; want none", kids)
	}
	// once the child lands, its base rides along and no longer blocks
	if unlanded := unlandedChildren(kids, "child"); len(unlanded) != 0 {
		t.Fatalf("a landed child no longer blocks: %q", unlanded)
	}
}

func TestStackedChildrenIgnoresSiblings(t *testing.T) {
	dir := prRepo(t)
	prInDir(t, dir)
	prGit(t, dir, "switch", "-q", "-c", "side/a")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "a")
	prGit(t, dir, "switch", "-q", "main")
	prGit(t, dir, "switch", "-q", "-c", "side/b")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "b")

	if kids := stackedChildren("side/a", "main"); len(kids) != 0 {
		t.Fatalf("a sibling is not a child: %q", kids)
	}
}

func TestStackTopPublishesTheWholeChain(t *testing.T) {
	dir := prRepo(t)
	prInDir(t, dir)
	prGit(t, dir, "switch", "-q", "-c", "base")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "base")
	prGit(t, dir, "switch", "-q", "-c", "mid")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "mid")
	prGit(t, dir, "switch", "-q", "-c", "leaf")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "leaf")

	top, chain, err := stackTop("base")
	if err != nil {
		t.Fatalf("stackTop(base) failed: %v", err)
	}
	if top != "leaf" {
		t.Fatalf("stackTop(base) = %q; want leaf", top)
	}
	if strings.Join(chain, ",") != "base,mid,leaf" {
		t.Fatalf("stackTop(base) chain = %q; want base,mid,leaf", chain)
	}
	if top, chain, err := stackTop("leaf"); err != nil || top != "leaf" || len(chain) != 1 {
		t.Fatalf("stackTop(leaf) = %q, %q, %v; want leaf alone", top, chain, err)
	}
}

func TestStackTopRefusesAFork(t *testing.T) {
	dir := prRepo(t)
	prInDir(t, dir)
	prGit(t, dir, "switch", "-q", "-c", "base")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "base")
	prGit(t, dir, "switch", "-q", "-c", "left")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "left")
	prGit(t, dir, "switch", "-q", "base")
	prGit(t, dir, "switch", "-q", "-c", "right")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "right")

	if _, _, err := stackTop("base"); err == nil {
		t.Fatal("stackTop(base) accepted a fork; want an error")
	}
}
