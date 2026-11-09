package ops

import (
	"fmt"
	"os"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Discard drops whatever the name points at: a dirty path, a commit, a branch
// and its worktree, or zz for every uncommitted change. The resolver in main.go
// types the argument first, so a path changeId, a commit changeId and a branch
// changeId cannot be confused; a plain name is sorted here.
func Discard(args []string) int {
	force, yes, remote := false, false, false
	var items []string
	for _, a := range args {
		switch a {
		case "-a", "--all":
			ui.Errf("gg discard: no -a. A bare discard drops every edit already.")
			return 2
		case "-f", "--force":
			force = true
		case "-y", "--yes":
			yes = true
		case "-r", "--remote":
			remote = true
		default:
			if strings.HasPrefix(a, "-") {
				ui.Errf("gg discard: unknown flag %s", a)
				return 2
			}
			items = append(items, a)
		}
	}
	if len(items) == 0 {
		return gitx.Run("reset", "--hard", "--quiet")
	}

	rc := 0
	var paths []string
	deleted := 0
	for _, item := range items {
		kind, name := discardKind(item)
		switch kind {
		case "all":
			if code := gitx.Run("reset", "--hard", "--quiet"); code != 0 {
				rc = code
			}
		case "commit":
			if _, err := gitx.Output("rev-parse", "--verify", "--quiet", name+"^"); err != nil {
				ui.Errf("gg discard: %s is the root commit; nothing to replay it onto", abbrev(name))
				rc = 1
				continue
			}
			if code := Abandon([]string{name}); code != 0 {
				rc = code
				continue
			}
			ui.Errf("gg discard: dropped commit %s", abbrev(name))
		case "branch":
			code, gone := discardBranch(name, force, yes, remote)
			if code != 0 {
				rc = code
				continue
			}
			if gone {
				deleted++
			}
		default:
			paths = append(paths, name)
		}
	}
	if deleted > 0 {
		recentBranches(5)
	}
	if len(paths) > 0 {
		if code := discardPaths(paths); code != 0 {
			rc = code
		}
	}
	return rc
}

// discardKind reads the typed prefix the resolver wrote, then sorts a plain
// name by what it is: an existing path wins, then a branch, then a commit.
func discardKind(item string) (string, string) {
	for _, prefix := range []string{"path:", "commit:", "branch:"} {
		if rest, ok := strings.CutPrefix(item, prefix); ok {
			return strings.TrimSuffix(prefix, ":"), rest
		}
	}
	if item == "zz" {
		return "all", ""
	}
	if _, err := os.Stat(item); err == nil {
		return "path", item
	}
	if branchExists(item) {
		return "branch", item
	}
	if revExists(item) {
		return "commit", item
	}
	return "path", item
}

// discardPaths drops the edits of the named tracked paths and leaves the rest
// of the tree alone. A new path has no change to drop, so it is never touched.
func discardPaths(paths []string) int {
	var known, fresh []string
	for _, p := range paths {
		if tracked(p) {
			known = append(known, p)
			continue
		}
		fresh = append(fresh, p)
	}
	if len(fresh) > 0 {
		ui.Errf("gg discard: %d path(s) are new, so no change is there to drop:", len(fresh))
		for _, p := range fresh {
			ui.Errf("    %s", p)
		}
		ui.Errf("gg discard: nothing touched. gg clear deletes new files.")
		return 1
	}
	argv := append([]string{"restore", "--source=HEAD", "--staged", "--worktree", "--"}, known...)
	return gitx.Run(argv...)
}

// discardBranch drops a branch, its worktree when one holds it, and origin's
// copy when -r says so. It returns whether a branch went, so the caller prints
// the newest branches once.
func discardBranch(name string, force, yes, remote bool) (int, bool) {
	all := worktrees()
	wt, hasWt := findWorktree(all, name)
	branch := name
	if hasWt {
		branch = wt.branch
		if len(all) > 0 && wt.path == all[0].path {
			ui.Errf("gg discard: %s holds the main worktree. Name the branch it holds instead.", wt.path)
			return 1, false
		}
	}
	if branch != "" {
		if cur, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD"); branch == cur {
			ui.Errf("gg discard: %s is the current branch; switch away first", branch)
			return 1, false
		}
	}
	exists := branch != "" && branchExists(branch)
	if !hasWt && !exists {
		ui.Errf("gg discard: no worktree and no branch named %s", name)
		return 1, false
	}

	if exists && !force && !branchMerged(branch) {
		ui.Errf("gg discard: %s holds commits that are not merged. -f drops them.", branch)
		return 1, false
	}

	if hasWt {
		if n := dirty(wt.path); n > 0 && !force {
			ui.Errf("gg discard: %s has %d dirty file(s). Use -f to drop them.", name, n)
			return 1, false
		}
		if !yes {
			what := fmt.Sprintf("worktree %s", wt.path)
			if exists {
				what = fmt.Sprintf("worktree %s and branch %s", wt.path, branch)
			}
			choice, ok := ui.Prompt(fmt.Sprintf("gg discard: remove %s? [y/N]: ", what))
			if !ok {
				ui.Errf("gg discard: refusing non-interactively; add -y to remove it")
				return 1, false
			}
			if choice != "y" && choice != "yes" {
				ui.Errf("gg discard: aborted, nothing dropped")
				return 1, false
			}
		}
		argv := []string{"worktree", "remove"}
		if force {
			argv = append(argv, "--force")
		}
		if rc := gitx.Run(append(argv, wt.path)...); rc != 0 {
			return rc, false
		}
		ui.Errf("gg discard: removed %s", wt.path)
	}
	if !exists {
		return 0, false
	}
	remoteName, upstream := upstreamOf(branch)
	if rc := dropBranch(branch, force); rc != 0 {
		return rc, false
	}
	if upstream == "" {
		return 0, true
	}
	switch {
	case remote:
		deleteRemote(remoteName, upstream)
	case !yes:
		choice, ok := ui.Prompt(fmt.Sprintf("gg discard: delete %s too? [y/N]: ", remoteName+"/"+upstream))
		if ok && (choice == "y" || choice == "yes") {
			deleteRemote(remoteName, upstream)
		} else if !ok {
			ui.Errf("gg discard: %s stays; pass -r to delete it", remoteName+"/"+upstream)
		}
	default:
		ui.Errf("gg discard: %s stays", remoteName+"/"+upstream)
	}
	return 0, true
}

// upstreamOf names the remote branch the local one tracks. It returns an empty
// name when the branch has no upstream, so an untracked branch never asks about
// a remote copy.
func upstreamOf(branch string) (remoteName, name string) {
	out, err := gitx.Output("rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil || out == "" {
		return "", ""
	}
	remoteName, name, ok := strings.Cut(out, "/")
	if !ok || remoteName == "" || name == "" {
		return "", ""
	}
	return remoteName, name
}

// branchMerged reports whether git would delete the branch without force: it is
// merged into its upstream, or into HEAD when it has none.
func branchMerged(branch string) bool {
	ref := "HEAD"
	if up, err := gitx.Output("rev-parse", "--abbrev-ref", "--symbolic-full-name", branch+"@{upstream}"); err == nil && up != "" {
		ref = up
	}
	return gitx.Run("merge-base", "--is-ancestor", "refs/heads/"+branch, ref) == 0
}

// dropBranch deletes one branch and keeps its tip reachable in
// refs/gg/deleted, so the changeId printed by gg br brings it back.
func dropBranch(name string, force bool) int {
	sha, err := gitx.Output("rev-parse", "--verify", "refs/heads/"+name)
	if err != nil {
		ui.Errf("gg discard: cannot read the tip of %s", name)
		return 1
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	if rc := gitx.Run("branch", flag, name); rc != 0 {
		if !force {
			ui.Errf("gg discard: %s holds commits that are not merged. -f drops them.", name)
		}
		return rc
	}
	_, _ = gitx.Output("update-ref", "refs/gg/deleted/"+name, sha)
	ui.Errf("gg: %s is deleted. Its tip stays in refs/gg/deleted/%s", name, name)
	return 0
}

// deleteRemote deletes the upstream copy of a branch, only when it exists there.
func deleteRemote(remoteName, branch string) {
	if !revExists("refs/remotes/" + remoteName + "/" + branch) {
		return
	}
	if rc := gitx.Run("push", remoteName, "--delete", branch); rc != 0 {
		ui.Errf("gg discard: %s stays", remoteName+"/"+branch)
		return
	}
	ui.Errf("gg discard: deleted %s", remoteName+"/"+branch)
}
