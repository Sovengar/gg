package ops

import (
	"encoding/json"
	"os/exec"

	"gg/internal/gitx"
)

// WtTree is the part of one `wt list --format json` entry that gg draws: the
// branch a worktree holds, how far it drifted from the main branch and whether
// the tree carries changes.
type WtTree struct {
	Branch    string `json:"branch"`
	Path      string `json:"path"`
	IsMain    bool   `json:"is_main"`
	IsCurrent bool   `json:"is_current"`
	Main      struct {
		Ahead  int `json:"ahead"`
		Behind int `json:"behind"`
	} `json:"main"`
	WorkingTree struct {
		Staged    bool `json:"staged"`
		Modified  bool `json:"modified"`
		Untracked bool `json:"untracked"`
		Renamed   bool `json:"renamed"`
		Deleted   bool `json:"deleted"`
	} `json:"working_tree"`
}

// Dirty reports whether the worktree carries changes, staged or not.
func (w WtTree) Dirty() bool {
	t := w.WorkingTree
	return t.Staged || t.Modified || t.Untracked || t.Renamed || t.Deleted
}

// WtTrees reads the worktree list from Worktrunk. It returns nil when the
// repository has one worktree, when the binary is missing or when the call
// fails: the markers compare parallel trees, and a lone tree needs none.
func WtTrees() []WtTree {
	if len(worktrees()) < 2 {
		return nil
	}
	bin, err := exec.LookPath("wt")
	if err != nil {
		return nil
	}
	out, err := gitx.OutputBin(bin, "list", "--format", "json", "--no-progressive")
	if err != nil {
		return nil
	}
	return parseWtTrees([]byte(out))
}

// parseWtTrees reads the JSON payload. A malformed payload yields nil: the
// worktree annotations are optional decoration, never a reason to fail.
func parseWtTrees(data []byte) []WtTree {
	var out []WtTree
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}
