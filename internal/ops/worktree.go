package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// worktree is one entry of `git worktree list --porcelain`.
type worktree struct {
	path   string
	branch string // empty when detached
}

// Worktree manages one worktree per branch: new, ls, rm.
func Worktree(args []string) int {
	if len(args) == 0 {
		return worktreeUsage()
	}
	switch args[0] {
	case "new":
		return worktreeNew(args[1:])
	case "ls", "list":
		return worktreeList()
	case "rm", "remove":
		return worktreeRemove(args[1:])
	default:
		return worktreeUsage()
	}
}

func worktreeUsage() int {
	ui.Errf("gg wt new <branch> [base] | gg wt ls | gg wt rm <name|path> [--force]")
	return 2
}

// WorktreeLines describes each worktree for the dashboard: the branch, the
// path and how dirty it is, the current one marked with "*". It returns
// nothing when the repository has a single worktree, where the view adds noise.
func WorktreeLines() []string {
	all := worktrees()
	if len(all) < 2 {
		return nil
	}
	root, _ := repoRoot()
	out := make([]string, 0, len(all))
	for _, w := range all {
		marker := " "
		if w.path == root {
			marker = "*"
		}
		state := "clean"
		if n := dirty(w.path); n > 0 {
			state = strconv.Itoa(n) + " dirty"
		}
		out = append(out, fmt.Sprintf("%s %-14s %s  %s", marker, worktreeName(w), w.path, state))
	}
	return out
}

// WorktreeBranches lists the branches a worktree holds, so the dashboard can
// draw one lane per parallel branch. The main worktree is included; callers
// skip the branch they are already on.
func WorktreeBranches() []string {
	var out []string
	for _, w := range worktrees() {
		if w.branch != "" {
			out = append(out, w.branch)
		}
	}
	return out
}

// repoRoot returns the top level of the current work tree.
func repoRoot() (string, bool) {
	root, err := gitx.Output("rev-parse", "--show-toplevel")
	if err != nil || root == "" {
		return "", false
	}
	return root, true
}

// worktreePath is the sibling directory that holds one branch.
func worktreePath(root, branch string) string {
	slug := strings.ReplaceAll(branch, "/", "-")
	return filepath.Join(filepath.Dir(root), filepath.Base(root)+"."+slug)
}

func worktreeNew(args []string) int {
	if len(args) < 1 {
		return worktreeUsage()
	}
	// fold a wip commit first: the new worktree must not start on top of it
	WipAmend()
	branch := args[0]
	base := "HEAD"
	if len(args) > 1 {
		base = args[1]
	}
	root, ok := repoRoot()
	if !ok {
		ui.Errf("gg wt new: not inside a work tree")
		return 1
	}
	path := worktreePath(root, branch)
	if _, err := os.Stat(path); err == nil {
		ui.Errf("gg wt new: %s already exists", path)
		return 1
	}

	gitArgs := []string{"worktree", "add"}
	if _, err := gitx.Output("show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		if w, busy := findWorktree(worktrees(), branch); busy {
			ui.Errf("gg wt new: branch %s is checked out in %s", branch, w.path)
			return 1
		}
		gitArgs = append(gitArgs, path, branch)
	} else {
		gitArgs = append(gitArgs, "-b", branch, path, base)
	}
	if rc := gitx.Run(gitArgs...); rc != 0 {
		return rc
	}
	ui.Errf("gg wt new: %s", path)
	return 0
}

// worktrees parses the porcelain listing.
func worktrees() []worktree {
	lines, err := gitx.OutputAll("worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}
	var out []worktree
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "worktree "):
			out = append(out, worktree{path: strings.TrimPrefix(line, "worktree ")})
		case strings.HasPrefix(line, "branch "):
			if len(out) > 0 {
				out[len(out)-1].branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
			}
		}
	}
	return out
}

// dirty counts the uncommitted files of one worktree.
func dirty(path string) int {
	out, err := gitx.Output("-C", path, "status", "--porcelain")
	if err != nil || out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

func worktreeList() int {
	all := worktrees()
	if len(all) == 0 {
		ui.Errf("gg wt ls: no worktrees")
		return 1
	}
	root, _ := repoRoot()
	width := 0
	for _, w := range all {
		if n := len(worktreeName(w)); n > width {
			width = n
		}
	}
	for _, w := range all {
		mark := " "
		if w.path == root {
			mark = ui.Cyan("*")
		}
		name := worktreeName(w)
		state := ui.Dim("clean")
		if n := dirty(w.path); n > 0 {
			state = ui.Yellow(strconv.Itoa(n) + " dirty")
		}
		fmt.Printf("%s %s  %s  %s\n",
			mark,
			ui.Cyan(fmt.Sprintf("%-*s", width, name)),
			ui.Dim(w.path),
			state)
	}
	return 0
}

// worktreeName is the branch, or the last path element when detached.
func worktreeName(w worktree) string {
	if w.branch != "" {
		return w.branch
	}
	return "detached"
}

func worktreeRemove(args []string) int {
	force := false
	var name string
	for _, a := range args {
		switch a {
		case "--force", "-f":
			force = true
		default:
			name = a
		}
	}
	if name == "" {
		return worktreeUsage()
	}
	all := worktrees()
	target, ok := findWorktree(all, name)
	if !ok {
		ui.Errf("gg wt rm: no worktree named %q", name)
		return 1
	}
	if len(all) > 0 && target.path == all[0].path {
		ui.Errf("gg wt rm: %s is the main worktree. Run git worktree remove from another directory.", target.path)
		return 1
	}
	if n := dirty(target.path); n > 0 && !force {
		ui.Errf("gg wt rm: %s has %d dirty file. Use --force to remove it.", name, n)
		return 1
	}
	gitArgs := []string{"worktree", "remove"}
	if force {
		gitArgs = append(gitArgs, "--force")
	}
	return gitx.Run(append(gitArgs, target.path)...)
}

// findWorktree resolves a branch name, a path or a directory name.
func findWorktree(all []worktree, name string) (worktree, bool) {
	for _, w := range all {
		switch {
		case w.path == name,
			w.branch == name,
			filepath.Base(w.path) == name:
			return w, true
		}
	}
	return worktree{}, false
}
