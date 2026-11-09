package ops

import (
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Reap drops a worktree, its branch, or both. A branch that still has a
// worktree is asked about; a plain branch is deleted the same way, with the
// remote question when origin holds it too.
func Reap(args []string) int {
	var names []string
	keep, drop, force, merged, yes, remote := false, false, false, false, false, false
	for _, a := range args {
		switch a {
		case "-k", "--keep-branch":
			keep = true
		case "-d", "--delete-branch":
			drop = true
		case "-D", "-f", "--force":
			force = true
		case "-m", "--merged":
			merged = true
		case "-y", "--yes":
			yes = true
		case "-r", "--remote":
			remote = true
		default:
			if strings.HasPrefix(a, "-") {
				ui.Errf("gg reap: unknown flag %s", a)
				return 2
			}
			names = append(names, a)
		}
	}
	if keep && drop {
		ui.Errf("gg reap: -k keeps the branch and -d deletes it; name one")
		return 2
	}
	if merged {
		if len(names) > 0 {
			ui.Errf("gg reap --merged: it takes no name")
			return 2
		}
		return reapMerged(force, yes, remote)
	}
	if len(names) == 0 {
		ui.Errf("gg reap <branch|path>... [--keep-branch|--delete-branch] [--force] [--merged]")
		return 2
	}
	rc := 0
	st := &reapStats{}
	for _, name := range names {
		if code := reapOne(name, keep, drop, force, yes, remote, false, st); code != 0 {
			rc = code
		}
	}
	if st.branches > 0 {
		recentBranches(5)
	}
	return rc
}

// reapStats counts what one reap run deleted, so the branch list prints once.
type reapStats struct{ branches int }

// reapOne drops one worktree and/or branch. asked reports whether the caller
// already answered the prompts, so a sweep does not ask once per branch.
func reapOne(name string, keep, drop, force, yes, remote, asked bool, st *reapStats) int {
	all := worktrees()
	wt, hasWt := findWorktree(all, name)
	branch := name
	if hasWt {
		branch = wt.branch
		if len(all) > 0 && wt.path == all[0].path {
			ui.Errf("gg reap: %s is the main worktree. Run git worktree remove from another directory.", wt.path)
			return 1
		}
	}
	if branch != "" {
		cur, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
		if branch == cur {
			ui.Errf("gg reap: %s is the current branch; switch away first", branch)
			return 1
		}
	}
	exists := branch != "" && branchExists(branch)
	if !hasWt && !exists {
		ui.Errf("gg reap: no worktree and no branch named %s", name)
		return 1
	}

	if hasWt {
		if n := dirty(wt.path); n > 0 && !force {
			ui.Errf("gg reap: %s has %d dirty file(s). Use --force to drop them.", name, n)
			return 1
		}
		if exists && !keep && !drop && !asked {
			choice, ok := ui.Prompt(fmt.Sprintf("gg reap: keep branch %s? [y/N]: ", branch))
			if !ok {
				ui.Errf("gg reap: refusing non-interactively; pass -k to keep %s or -d to delete it", branch)
				return 1
			}
			keep = choice == "y" || choice == "yes"
		}
	}

	if hasWt {
		argv := []string{"worktree", "remove"}
		if force {
			argv = append(argv, "--force")
		}
		if rc := gitx.Run(append(argv, wt.path)...); rc != 0 {
			return rc
		}
		ui.Errf("gg reap: removed %s", wt.path)
	}
	if exists && !keep {
		if rc := dropBranch(branch, force); rc != 0 {
			return rc
		}
		st.branches++
		if !revExists("refs/remotes/origin/" + branch) {
			return 0
		}
		switch {
		case remote:
			deleteRemote(branch)
		case !asked:
			choice, ok := ui.Prompt(fmt.Sprintf("gg reap: delete origin/%s too? [y/N]: ", branch))
			if ok && (choice == "y" || choice == "yes") {
				deleteRemote(branch)
			} else if !ok {
				ui.Errf("gg reap: origin/%s stays; pass -r to delete it", branch)
			}
		default:
			ui.Errf("gg reap: origin/%s stays", branch)
		}
	}
	return 0
}

// reapMerged sweeps every worktree and branch already merged into the default
// branch. It lists them before it asks, because a sweep can remove several
// directories at once.
func reapMerged(force, yes, remote bool) int {
	target := ConventionBranch()
	if target == "" {
		target = "HEAD"
	}
	out, err := gitx.OutputAll("branch", "--merged", target, "--format=%(refname:short)")
	if err != nil {
		ui.Errf("gg reap --merged: cannot list the branches merged into %s", target)
		return 1
	}
	cur, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	var names []string
	for _, line := range out {
		name := strings.TrimSpace(line)
		if name == "" || name == target || name == cur {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		ui.Errf("gg reap --merged: nothing is merged into %s", target)
		return 0
	}
	ui.Errf("gg reap: %d branch(es) merged into %s:", len(names), target)
	for _, name := range names {
		ui.Errf("    %s", name)
	}
	if !yes {
		choice, ok := ui.Prompt(fmt.Sprintf("Reap %d branch(es)? [y/N]: ", len(names)))
		if !ok {
			ui.Errf("gg reap: refusing non-interactively; add -y to sweep")
			return 1
		}
		if choice != "y" && choice != "yes" {
			ui.Errf("gg reap: aborted, nothing reaped")
			return 1
		}
	}
	rc := 0
	st := &reapStats{}
	for _, name := range names {
		if code := reapOne(name, false, true, force, true, remote, true, st); code != 0 {
			rc = code
		}
	}
	if st.branches > 0 {
		recentBranches(5)
	}
	return rc
}

// dropBranch deletes one branch and keeps its tip reachable in
// refs/gg/deleted, so the changeId printed by gg br brings it back.
func dropBranch(name string, force bool) int {
	sha, err := gitx.Output("rev-parse", "--verify", "refs/heads/"+name)
	if err != nil {
		ui.Errf("gg reap: cannot read the tip of %s", name)
		return 1
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	if rc := gitx.Run("branch", flag, name); rc != 0 {
		if !force {
			ui.Errf("gg reap: %s holds commits that are not merged. -D drops them.", name)
		}
		return rc
	}
	_, _ = gitx.Output("update-ref", "refs/gg/deleted/"+name, sha)
	ui.Errf("gg: %s is deleted. Its tip stays in refs/gg/deleted/%s", name, name)
	return 0
}

// deleteRemote deletes origin's copy of a branch, only when it exists there.
func deleteRemote(branch string) {
	if !revExists("refs/remotes/origin/" + branch) {
		return
	}
	if rc := gitx.Run("push", "origin", "--delete", branch); rc != 0 {
		ui.Errf("gg reap: origin/%s stays", branch)
		return
	}
	ui.Errf("gg reap: deleted origin/%s", branch)
}
