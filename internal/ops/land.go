package ops

import (
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Land fast-forwards the target branch on the remote to a branch tip and
// pushes it, the way `but land` does. It never creates a merge commit: a
// branch that is not strictly ahead of the target cannot land, because a real
// merge there would land without the review and the CI a pull request runs.
func Land(args []string) int {
	branch, target, yes, err := landArgs(args)
	if err != nil {
		ui.Errf("gg land: %v", err)
		return 2
	}
	if wipGuard() {
		return 1
	}
	if _, err := gitx.Output("remote", "get-url", "origin"); err != nil {
		ui.Errf("gg land: no origin remote to land on")
		return 1
	}
	if branch == "" {
		cur, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil || cur == "HEAD" {
			ui.Errf("gg land: detached HEAD, name the branch to land")
			return 2
		}
		branch = cur
	}
	if target == "" {
		target = ConventionBranch()
		if target == "" {
			ui.Errf("gg land: this repository names no default branch; pass -b <target>")
			return 2
		}
	}
	if branch == target {
		ui.Errf("gg land: %s is the target branch itself", branch)
		return 2
	}
	if !branchExists(branch) {
		ui.Errf("gg land: no branch named %s", branch)
		return 2
	}
	if rc := gitx.Run("fetch", "--quiet", "--prune", "origin"); rc != 0 {
		return rc
	}

	remoteRef := "origin/" + target
	if !revExists(remoteRef) {
		ui.Errf("gg land: origin has no branch %s", target)
		return 1
	}
	sha, err := gitx.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		ui.Errf("gg land: cannot read %s", branch)
		return 1
	}
	base, err := gitx.Output("rev-parse", "--verify", "--quiet", remoteRef)
	if err != nil {
		ui.Errf("gg land: cannot read %s", remoteRef)
		return 1
	}
	if sha == base {
		ui.Errf("gg land: %s is already landed on %s", branch, remoteRef)
		return 0
	}
	if rc := gitx.Run("merge-base", "--is-ancestor", remoteRef, "refs/heads/"+branch); rc != 0 {
		ui.Errf("gg land: %s is not ahead of %s, so it cannot fast-forward.", branch, remoteRef)
		ui.Errf("gg land: gg pr opens a pull request instead, with review and CI.")
		return 1
	}
	if !yes && !landConfirm(branch, remoteRef) {
		return 1
	}

	note := ffLocalTarget(target, sha)
	if rc := gitx.Run("push", "origin", sha+":refs/heads/"+target); rc != 0 {
		return rc
	}
	ui.Errf("gg land: landed %s onto %s.", branch, remoteRef)
	ui.Errf("gg land: pushed to %s; gg undo cannot un-push it.", remoteRef)
	ui.Errf("gg land: to revert, gg revert %s then gg push, or use your forge.", abbrev(sha))
	if note != "" {
		ui.Errf("gg land: %s", note)
	}
	return 0
}

// landArgs reads the branch, the target and the accept flag. A branch is
// optional, HEAD's branch when missing; the target falls back to the
// repository's default branch.
func landArgs(args []string) (branch, target string, yes bool, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-y", "--yes":
			yes = true
			continue
		case "-b", "--onto", "--base":
			if i+1 >= len(args) {
				return "", "", false, fmt.Errorf("%s needs a target branch", a)
			}
			if target != "" {
				return "", "", false, fmt.Errorf("name one target branch")
			}
			target = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			return "", "", false, fmt.Errorf("unknown flag %s", a)
		}
		if branch != "" {
			return "", "", false, fmt.Errorf("name one branch to land")
		}
		branch = a
	}
	return branch, target, yes, nil
}

// landConfirm prints the price of landing before it asks: no pull request, so
// no review and no CI.
func landConfirm(branch, remoteRef string) bool {
	ui.Errf("gg land: this lands %s directly onto %s, without a pull request:", branch, remoteRef)
	ui.Errf("gg land: review, CI checks and branch protection are skipped. gg pr opens a PR instead.")
	choice, ok := ui.Prompt(fmt.Sprintf("Land %s onto %s? [y/N]: ", branch, remoteRef))
	if !ok {
		ui.Errf("gg land: refusing to land non-interactively; add -y to accept the risk")
		return false
	}
	if choice != "y" && choice != "yes" {
		ui.Errf("gg land: aborted, nothing landed")
		return false
	}
	return true
}

// ffLocalTarget moves the local target branch to sha when that is still a
// fast-forward and no worktree holds the branch. A skipped move returns a
// note, never an error: the remote is what landed.
func ffLocalTarget(target, sha string) string {
	if !branchExists(target) {
		return ""
	}
	cur, err := gitx.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+target)
	if err != nil || cur == sha {
		return ""
	}
	if rc := gitx.Run("merge-base", "--is-ancestor", cur, sha); rc != 0 {
		return fmt.Sprintf("local %s keeps commits %s lacks, so it stays at %s", target, abbrev(sha), abbrev(cur))
	}
	for _, w := range worktrees() {
		if w.branch == target {
			return fmt.Sprintf("local %s stays at %s: %s holds it", target, abbrev(cur), w.path)
		}
	}
	if rc := gitx.Run("update-ref", "-m", "gg land", "refs/heads/"+target, sha); rc != 0 {
		return ""
	}
	return fmt.Sprintf("local %s moved to %s", target, abbrev(sha))
}
