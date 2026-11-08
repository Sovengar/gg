package ops

import (
	"fmt"
	"os"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// sequenceSteps are the files git writes while an operation owns the tree.
var sequenceSteps = []struct{ path, label string }{
	{"rebase-merge", "a rebase"},
	{"rebase-apply", "a rebase"},
	{"MERGE_HEAD", "a merge"},
	{"CHERRY_PICK_HEAD", "a cherry-pick"},
	{"REVERT_HEAD", "a revert"},
}

// New starts a wip commit that holds the working tree, jj style.
func New(args []string) int {
	msg, force, err := wipArgs(args)
	if err != nil {
		ui.Errf("gg new: %v", err)
		return 2
	}
	if step := sequenceOwner(); step != "" {
		ui.Errf("gg new: git is in the middle of %s. Finish or abort it first.", step)
		return 1
	}
	if gitx.Unmerged() {
		ui.Errf("gg new: the tree has conflicts. Resolve them first; gg new would commit the markers.")
		return 1
	}
	if up, published := publishedWip(); published && !force {
		ui.Errf("gg new: this wip commit is already on %s. Take it back with gg uncommit first, or pass --force.", up)
		return 1
	}
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

// wipArgs reads gg new's own flags. gg new commits the whole tree, so a
// mistyped argument must be an error, never something to ignore.
func wipArgs(args []string) (msg string, force bool, err error) {
	msg = "wip"
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-m" || a == "--message":
			if i+1 >= len(args) {
				return "", false, fmt.Errorf("%s needs a message", a)
			}
			msg = args[i+1]
			i++
		case strings.HasPrefix(a, "--message="):
			msg = strings.TrimPrefix(a, "--message=")
		case a == "-f" || a == "--force":
			force = true
		default:
			return "", false, fmt.Errorf("unknown argument %q: gg new takes -m <msg> and --force", a)
		}
	}
	return msg, force, nil
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
func inSequence() bool { return sequenceOwner() != "" }

// sequenceOwner names the operation that owns the tree, or "" when none does.
func sequenceOwner() string {
	for _, step := range sequenceSteps {
		path, err := gitx.Output("rev-parse", "--git-path", step.path)
		if err != nil || path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return step.label
		}
	}
	return ""
}

// publishedWip names the upstream that already holds the wip commit. Folding
// it there rewrites a published commit, so gg new asks for --force.
func publishedWip() (string, bool) {
	if !gitx.InWip() {
		return "", false
	}
	up, ok := gitx.Upstream()
	if !ok {
		return "", false
	}
	if _, err := gitx.Output("merge-base", "--is-ancestor", "HEAD", up); err != nil {
		return "", false
	}
	return up, true
}
