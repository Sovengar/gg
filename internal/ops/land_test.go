package ops

import "testing"

func TestLandArgs(t *testing.T) {
	cases := []struct {
		args   []string
		branch string
		target string
		yes    bool
		bad    bool
	}{
		{nil, "", "", false, false},
		{[]string{"side"}, "side", "", false, false},
		{[]string{"side", "-y"}, "side", "", true, false},
		{[]string{"--yes", "side"}, "side", "", true, false},
		{[]string{"side", "-b", "main"}, "side", "main", false, false},
		{[]string{"-b", "release", "-y", "side"}, "side", "release", true, false},
		{[]string{"--onto", "main", "side"}, "side", "main", false, false},
		{[]string{"side", "other"}, "", "", false, true},
		{[]string{"side", "-b"}, "", "", false, true},
		{[]string{"side", "-b", "a", "--onto", "b"}, "", "", false, true},
		{[]string{"--nope"}, "", "", false, true},
	}
	for _, c := range cases {
		branch, target, yes, err := landArgs(c.args)
		if c.bad {
			if err == nil {
				t.Errorf("landArgs(%q) = %q, %q, %v; want an error", c.args, branch, target, yes)
			}
			continue
		}
		if err != nil {
			t.Errorf("landArgs(%q) failed: %v", c.args, err)
			continue
		}
		if branch != c.branch || target != c.target || yes != c.yes {
			t.Errorf("landArgs(%q) = %q, %q, %v; want %q, %q, %v",
				c.args, branch, target, yes, c.branch, c.target, c.yes)
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
