package ops

import (
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Ship turns a branch into one commit, replays it on the target branch,
// fast-forwards the target and removes the branch's worktree. It never talks to
// a remote: `gg land` publishes, `gg ship` composes.
func Ship(args []string) int {
	branch, target, msg, yes, err := shipArgs(args)
	if err != nil {
		ui.Errf("gg ship: %v", err)
		return 2
	}
	if wipGuard() {
		return 1
	}
	if branch == "" {
		cur, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil || cur == "HEAD" {
			ui.Errf("gg ship: detached HEAD, name the branch to ship")
			return 2
		}
		branch = cur
	}
	if target == "" {
		target = ConventionBranch()
		if target == "" {
			ui.Errf("gg ship: this repository names no default branch; pass -b <target>")
			return 2
		}
	}
	if branch == target {
		ui.Errf("gg ship: %s is the target branch itself", branch)
		return 2
	}
	if !branchExists(branch) {
		ui.Errf("gg ship: no branch named %s", branch)
		return 2
	}
	if !branchExists(target) {
		ui.Errf("gg ship: no local branch named %s", target)
		return 2
	}

	old, err := gitx.Output("rev-parse", "--verify", "refs/heads/"+branch)
	if err != nil {
		ui.Errf("gg ship: cannot read %s", branch)
		return 1
	}
	base, err := gitx.Output("merge-base", target, branch)
	if err != nil {
		ui.Errf("gg ship: %s and %s share no base", branch, target)
		return 1
	}
	if base == old {
		ui.Errf("gg ship: %s holds no commit %s lacks", branch, target)
		return 1
	}
	commits, _ := gitx.Count(target + ".." + branch)
	if msg == "" {
		if commits != 1 {
			ui.Errf("gg ship: %s holds %d commits; name the squashed message with -m", branch, commits)
			return 2
		}
		if msg, err = gitx.Output("log", "-1", "--format=%s", branch); err != nil || msg == "" {
			ui.Errf("gg ship: cannot read the message of %s; pass -m", branch)
			return 1
		}
	}

	cur, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	holder := ""
	for _, w := range worktrees() {
		if w.branch == target {
			holder = w.path
		}
	}
	if holder != "" && cur != target {
		ui.Errf("gg ship: %s holds %s; ship from there, or switch this worktree to %s", holder, target, target)
		return 1
	}

	wt, hasWt := findWorktree(worktrees(), branch)
	if hasWt {
		if n := dirty(wt.path); n > 0 {
			ui.Errf("gg ship: %s has %d dirty file(s). Commit or discard them first.", wt.path, n)
			return 1
		}
	}
	if !yes && !shipConfirm(branch, target, commits) {
		return 1
	}

	targetTip, err := gitx.Output("rev-parse", "--verify", "refs/heads/"+target)
	if err != nil {
		ui.Errf("gg ship: cannot read %s", target)
		return 1
	}
	tree, err := gitx.Output("merge-tree", "--write-tree", targetTip, branch)
	if err != nil || tree == "" {
		ui.Errf("gg ship: %s conflicts with %s. Rebase it first: gg rb %s", branch, target, target)
		return 1
	}
	newSha, err := gitx.Output("commit-tree", tree, "-p", targetTip, "-m", msg)
	if err != nil || newSha == "" {
		ui.Errf("gg ship: cannot write the squashed commit")
		return 1
	}
	if rc := gitx.Run("update-ref", "-m", "gg ship", "refs/heads/"+branch, newSha, old); rc != 0 {
		return rc
	}
	if note := ffTarget(target, newSha); note != "" {
		ui.Errf("gg ship: %s", note)
	}
	if hasWt {
		if cur, ok := repoRoot(); ok && cur == wt.path {
			ui.Errf("gg ship: %s stays; run gg reap %s from another worktree to remove it", wt.path, branch)
		} else if rc := gitx.Run("worktree", "remove", "--force", wt.path); rc != 0 {
			return rc
		} else {
			ui.Errf("gg ship: removed %s", wt.path)
		}
	}
	ui.Errf("gg ship: %s is one commit on %s: %s", branch, target, abbrev(newSha))
	return 0
}

// shipArgs reads the branch, the target, the message and the accept flag.
func shipArgs(args []string) (branch, target, msg string, yes bool, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-y", "--yes":
			yes = true
		case "-b", "--onto", "--base":
			if i+1 >= len(args) {
				return "", "", "", false, fmt.Errorf("%s needs a target branch", a)
			}
			target = args[i+1]
			i++
		case "-m", "--message":
			if i+1 >= len(args) {
				return "", "", "", false, fmt.Errorf("%s needs a message", a)
			}
			msg = args[i+1]
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return "", "", "", false, fmt.Errorf("unknown flag %s", a)
			}
			if branch != "" {
				return "", "", "", false, fmt.Errorf("name one branch to ship")
			}
			branch = a
		}
	}
	return branch, target, msg, yes, nil
}

// shipConfirm says what shipping costs: the branch history becomes one commit
// and its worktree goes away.
func shipConfirm(branch, target string, commits int) bool {
	ui.Errf("gg ship: %d commit(s) on %s become one commit on %s.", commits, branch, target)
	ui.Errf("gg ship: this rewrites %s and removes its worktree. gg undo restores refs, not another worktree.", branch)
	choice, ok := ui.Prompt(fmt.Sprintf("Ship %s onto %s? [y/N]: ", branch, target))
	if !ok {
		ui.Errf("gg ship: refusing non-interactively; add -y to ship")
		return false
	}
	if choice != "y" && choice != "yes" {
		ui.Errf("gg ship: aborted, nothing shipped")
		return false
	}
	return true
}

// ffTarget moves the local target onto sha. It merges when the target is the
// current branch, moves the ref when nothing holds it, and reports when another
// worktree owns it.
func ffTarget(target, sha string) string {
	cur, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if cur == target {
		if rc := gitx.Run("merge", "--ff-only", "--quiet", sha); rc != 0 {
			return fmt.Sprintf("local %s did not fast-forward", target)
		}
		return fmt.Sprintf("local %s moved to %s", target, abbrev(sha))
	}
	if rc := gitx.Run("update-ref", "-m", "gg ship", "refs/heads/"+target, sha); rc != 0 {
		return fmt.Sprintf("local %s stays behind", target)
	}
	return fmt.Sprintf("local %s moved to %s", target, abbrev(sha))
}
