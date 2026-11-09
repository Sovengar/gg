package ops

import "testing"

// wtPayload is the shape wt list --format json prints for two worktrees.
const wtPayload = `[
  {"branch":"main","path":"/r","is_main":true,"is_current":true,
   "main_state":"is_main","working_tree":{"modified":true}},
  {"branch":"side","path":"/r.side","is_main":false,"is_current":false,
   "main":{"ahead":2,"behind":1},
   "working_tree":{"untracked":true},"worktree":{"state":"branch_worktree_mismatch"}}
]`

func TestParseWtTrees(t *testing.T) {
	trees := parseWtTrees([]byte(wtPayload))
	if len(trees) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(trees))
	}
	side := trees[1]
	if side.Branch != "side" || side.Path != "/r.side" {
		t.Fatalf("wrong worktree: %+v", side)
	}
	if side.Main.Ahead != 2 || side.Main.Behind != 1 {
		t.Fatalf("wrong drift: %+v", side.Main)
	}
	if !side.Dirty() {
		t.Fatal("an untracked path makes the worktree dirty")
	}
	if trees[0].Dirty() != true {
		t.Fatal("a modified path makes the worktree dirty")
	}
}

func TestParseWtTreesRefusesGarbage(t *testing.T) {
	if got := parseWtTrees([]byte("not json")); got != nil {
		t.Fatalf("garbage must yield nil, got %+v", got)
	}
}
