package ops

import (
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Copy replays one commit onto a branch, cherry-pick style. With no branch it
// copies onto the current one; with a branch it checks out, copies, and comes
// back, so the working branch stays the one you were on.
func Copy(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		ui.Errf("gg copy <changeId>: name the commit to copy")
		return 2
	}
	commit := args[0]

	if len(args) < 2 {
		return gitx.Run("cherry-pick", commit)
	}
	target := args[1]
	if !branchExists(target) {
		ui.Errf("gg copy: no branch named %s", target)
		return 2
	}
	current, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		ui.Errf("gg copy: cannot read the current branch")
		return 1
	}
	if current == target {
		return gitx.Run("cherry-pick", commit)
	}
	if n := dirty("."); n > 0 {
		ui.Errf("gg copy: the tree is dirty, commit or discard first")
		return 1
	}

	ui.Errf("gg: copying %s onto %s", commit, target)
	if rc := gitx.Run("switch", target); rc != 0 {
		return rc
	}
	if rc := gitx.Run("cherry-pick", commit); rc != 0 {
		ui.Errf("gg: cherry-pick stopped on %s; resolve it, or run gg abort", target)
		return rc
	}
	return gitx.Run("switch", current)
}

// branchExists reports whether a local branch carries that name.
func branchExists(name string) bool {
	_, err := gitx.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}
