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
	branch, target, yes, stack, err := landArgs(args)
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
	if stack {
		top, chain, err := stackTop(branch)
		if err != nil {
			ui.Errf("gg land: %v", err)
			return 1
		}
		if top != branch {
			ui.Errf("gg land: stack %s; landing %s publishes all of it",
				strings.Join(chain, " <- "), top)
			branch = top
		}
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
	children := stackedChildren(branch, target)
	// a child landed first already carries the branch into the target, so the
	// branch itself has nothing left to publish
	if len(children) > 0 && gitx.Run("merge-base", "--is-ancestor", "refs/heads/"+branch, remoteRef) == 0 {
		ui.Errf("gg land: %s is already landed on %s, carried by %s",
			branch, remoteRef, strings.Join(children, ", "))
		return 0
	}
	if rc := gitx.Run("merge-base", "--is-ancestor", remoteRef, "refs/heads/"+branch); rc != 0 {
		ui.Errf("gg land: %s is not ahead of %s, so it cannot fast-forward.", branch, remoteRef)
		ui.Errf("gg land: gg pr opens a pull request instead, with review and CI.")
		return 1
	}
	if kids := unlandedChildren(children, remoteRef); len(kids) > 0 {
		ui.Errf("gg land: %s has stacked branches on top: %s", branch, strings.Join(kids, ", "))
		ui.Errf("gg land: land the children first, leaf to root, so the base publishes last.")
		ui.Errf("gg land: to drop one instead, gg discard branch:%s", kids[0])
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

// stackedChildren names the branches stacked directly on top of branch. The
// parent is derived, so this reads the same stack gg br -a draws, and the
// target itself never counts as a child.
func stackedChildren(branch, target string) []string {
	rows := branchRows()
	if len(rows) == 0 {
		return nil
	}
	nodes := stackNodes(rows, ConventionBranch())
	kids := make([]string, 0)
	for _, r := range rows {
		if r.name == branch || r.name == target {
			continue
		}
		if nodes[r.name].parent == branch {
			kids = append(kids, r.name)
		}
	}
	return kids
}

// unlandedChildren keeps the children whose tips the target does not hold yet.
// Landing a base while a child still stands on it publishes a base the stack is
// about to move, so the stack lands leaf first. A child the target already
// holds no longer blocks: the base came with it.
func unlandedChildren(kids []string, remoteRef string) []string {
	out := make([]string, 0, len(kids))
	for _, name := range kids {
		sha, err := gitx.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
		if err != nil {
			continue
		}
		if rc := gitx.Run("merge-base", "--is-ancestor", sha, remoteRef); rc == 0 {
			continue
		}
		out = append(out, name)
	}
	return out
}

// stackTop walks the single-child chain up from branch and names its tip.
// Landing that tip publishes every branch under it in one push, because a
// child's history already contains its parent. A fork has no single tip, so
// each line must land on its own.
func stackTop(branch string) (top string, chain []string, err error) {
	rows := branchRows()
	if len(rows) == 0 {
		return "", nil, fmt.Errorf("no branches to stack")
	}
	nodes := stackNodes(rows, ConventionBranch())
	kids := make(map[string][]string, len(rows))
	for _, r := range rows {
		if p := nodes[r.name].parent; p != "" {
			kids[p] = append(kids[p], r.name)
		}
	}
	top, chain = branch, []string{branch}
	for {
		next := kids[top]
		switch len(next) {
		case 0:
			return top, chain, nil
		case 1:
			top = next[0]
			chain = append(chain, top)
		default:
			return "", nil, fmt.Errorf("%s forks into %s; land each top branch on its own",
				top, strings.Join(next, ", "))
		}
	}
}

// landArgs reads the branch, the target, the accept flag and the stack flag.
// A branch is optional, HEAD's branch when missing; the target falls back to
// the repository's default branch.
func landArgs(args []string) (branch, target string, yes, stack bool, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-y", "--yes":
			yes = true
			continue
		case "-s", "--stack":
			stack = true
			continue
		case "-b", "--onto", "--base":
			if i+1 >= len(args) {
				return "", "", false, false, fmt.Errorf("%s needs a target branch", a)
			}
			if target != "" {
				return "", "", false, false, fmt.Errorf("name one target branch")
			}
			target = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			return "", "", false, false, fmt.Errorf("unknown flag %s", a)
		}
		if branch != "" {
			return "", "", false, false, fmt.Errorf("name one branch to land")
		}
		branch = a
	}
	return branch, target, yes, stack, nil
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
