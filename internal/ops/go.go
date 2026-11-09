package ops

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gg/internal/gitx"
	"gg/internal/oplog"
	"gg/internal/ui"
)

// Go moves to where a branch lives: its worktree when one holds it, this
// checkout when none does, and a new branch asks where it should live. It
// prints the directory on stdout, because a program cannot change the shell
// that started it; `gg shell install` gives fish a wrapper that cds there.
func Go(args []string) int {
	branch := ""
	var exe []string
	mode := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			exe = args[i+1:]
			i = len(args)
		case a == "-w" || a == "--worktree":
			mode = "worktree"
		case a == "-i" || a == "--in-place":
			mode = "in-place"
		case strings.HasPrefix(a, "-"):
			ui.Errf("gg go: unknown flag %s", a)
			return 2
		default:
			if branch != "" {
				ui.Errf("gg go: name one branch")
				return 2
			}
			branch = a
		}
	}
	if branch == "" {
		return goPrompt(exe)
	}

	root, ok := repoRoot()
	if !ok {
		ui.Errf("gg go: not inside a work tree")
		return 1
	}
	if wt, found := findWorktree(worktrees(), branch); found && wt.branch != "" {
		return goEnter(wt.path, exe)
	}
	base := ConventionBranch()
	if branchExists(branch) {
		if mode == "worktree" {
			if rc := forkFrom(branch, base); rc != 0 {
				return rc
			}
			return goEnter(worktreePath(root, branch), exe)
		}
		if rc := switchTo(branch); rc != 0 {
			return rc
		}
		return goEnter(root, exe)
	}

	if base == "" {
		ui.Errf("gg go: %s does not exist, and this repository names no default branch to start it from", branch)
		return 1
	}
	if mode == "" {
		choice, offered := ui.Prompt(fmt.Sprintf("gg go: %s does not exist. Put it in its own worktree? [y/N]: ", branch))
		if offered && (choice == "y" || choice == "yes") {
			mode = "worktree"
		} else {
			mode = "in-place"
		}
	}
	if mode == "worktree" {
		oplog.Record("go")
		if rc := forkFrom(branch, base); rc != 0 {
			return rc
		}
		return goEnter(worktreePath(root, branch), exe)
	}
	oplog.Record("go")
	WipAmend()
	if rc := gitx.Run("switch", "-c", branch, base); rc != 0 {
		return rc
	}
	return goEnter(root, exe)
}

// goPrompt lists what can be reached and asks for one of them.
func goPrompt(exe []string) int {
	rows := WorktreeLines()
	if len(rows) == 0 {
		ui.Errf("gg go: this repository has a single worktree; name a branch")
		return 2
	}
	ui.Errf("gg go: name a branch, or one of these worktrees:")
	for _, row := range rows {
		ui.Errf("    %s", row)
	}
	choice, ok := ui.Prompt("gg go: branch? ")
	if !ok {
		ui.Errf("gg go: no terminal to ask on; name the branch")
		return 2
	}
	if choice == "" {
		return 2
	}
	return Go(append([]string{choice, "--"}, exe...))
}

// switchTo changes this checkout to an existing branch, folding a wip commit
// first the way every history-moving verb does.
func switchTo(branch string) int {
	cur, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if cur == branch {
		return 0
	}
	oplog.Record("go")
	WipAmend()
	return gitx.Run("switch", branch)
}

// forkFrom makes the sibling worktree of a branch, on that base.
func forkFrom(branch, base string) int {
	return worktreeNew([]string{branch, base}, "gg go")
}

// goEnter prints the directory, or runs the command inside it when the caller
// passed one after --.
func goEnter(path string, exe []string) int {
	if len(exe) == 0 {
		fmt.Println(path)
		return 0
	}
	cmd := exec.Command(exe[0], exe[1:]...)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		ui.Errf("gg go: %v", err)
		return 1
	}
	return 0
}
