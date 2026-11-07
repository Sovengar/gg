package ops

import (
	"os"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// sequencePaths are the files git writes while an operation owns the tree.
var sequencePaths = []string{
	"rebase-merge", "rebase-apply",
	"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD",
}

// New starts a wip commit that holds the working tree, jj style.
func New(args []string) int {
	msg := wipMessage(args)
	warnPublished()
	if gitx.InWip() {
		// the wip commit is HEAD: fold the edits in, then start a fresh one
		WipAmend()
	}
	if rc := gitx.Run("add", "-A"); rc != 0 {
		return rc
	}
	if rc := gitx.Run("commit", "--allow-empty", "--no-verify", "-m", msg); rc != 0 {
		return rc
	}
	gitx.MarkWip()
	return 0
}

// wipMessage reads -m/--message, defaulting to "wip".
func wipMessage(args []string) string {
	for i, a := range args {
		if (a == "-m" || a == "--message") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return "wip"
}

// WipAmend folds uncommitted edits into the wip commit. It does nothing when
// HEAD is not the wip commit, when the tree is clean, or while git is in the
// middle of an operation that owns the tree.
func WipAmend() {
	sha, ok := gitx.WipSha()
	if !ok {
		return
	}
	head, ok := gitx.Head()
	if !ok || sha != head {
		gitx.DropWip() // stale marker: HEAD moved on
		return
	}
	if !gitx.Dirty() || inSequence() {
		return
	}
	if rc := gitx.Run("add", "-A"); rc != 0 {
		return
	}
	if rc := gitx.Run("commit", "--amend", "--no-edit"); rc != 0 {
		return
	}
	gitx.MarkWip() // the amend rewrote HEAD, so move the marker with it
}

// inSequence reports whether git is inside a rebase, merge or pick.
func inSequence() bool {
	for _, name := range sequencePaths {
		path, err := gitx.Output("rev-parse", "--git-path", name)
		if err != nil || path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// warnPublished warns when the wip commit is already on the upstream, where
// folding it rewrites a published commit. The new wip commit is always
// unpublished, so the check runs before the commit, not after it.
func warnPublished() {
	if !gitx.InWip() {
		return
	}
	up, ok := gitx.Upstream()
	if !ok {
		return
	}
	if _, err := gitx.Output("merge-base", "--is-ancestor", "HEAD", up); err != nil {
		return
	}
	ui.Errf("gg new: this wip commit is already on %s. Take it back with gg uncommit first.", up)
}
