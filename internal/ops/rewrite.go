package ops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Move rebases a commit and its descendants onto another commit.
func Move(args []string) int {
	if len(args) < 2 {
		ui.Errf("gg move <commit> <onto>")
		return 2
	}
	what, onto := args[0], args[1]
	return gitx.Run("rebase", "--autostash", "--onto", onto, what+"^")
}

// Revert undoes one commit with a new commit on top. It is the only undo that
// is safe on published history, because it never rewrites: the original commit
// stays reachable. A merge commit needs -m N, which git asks for itself.
func Revert(args []string) int {
	if len(args) == 0 {
		ui.Errf("gg revert: name a commit.")
		return 2
	}
	if wipGuard() {
		return 1
	}
	if inSequence() {
		ui.Errf("gg revert: git is mid-operation. Run gg continue or gg abort first.")
		return 1
	}
	return gitx.Run(append([]string{"revert", "--no-edit"}, args...)...)
}

// Abandon drops one commit and replays its descendants on its parent.
func Abandon(args []string) int {
	if len(args) < 1 {
		ui.Errf("gg abandon <commit>")
		return 2
	}
	what := args[0]
	if HistorySupported() {
		return gitx.Run("history", "drop", what)
	}
	return gitx.Run("rebase", "--autostash", "--onto", what+"^", what)
}

// Amend folds the staged changes into HEAD, or into any commit when git is new
// enough to offer `git history fixup`.
func Amend(args []string) int {
	if len(args) == 0 {
		return gitx.Run("commit", "-a", "--amend", "--no-edit")
	}
	if !HistorySupported() {
		ui.Errf("gg: amending another commit needs git >= 2.56 (git history fixup)")
		return 1
	}
	return gitx.Run(append([]string{"history", "fixup", args[0]}, args[1:]...)...)
}

// Reword edits the message of one commit.
func Reword(args []string) int {
	if len(args) < 1 {
		ui.Errf("gg reword <commit>")
		return 2
	}
	if !HistorySupported() {
		ui.Errf("gg: reword needs git >= 2.56 (git history reword)")
		return 1
	}
	return gitx.Run("history", "reword", args[0])
}

// Split cuts one commit in two: the chosen hunks become its parent.
func Split(args []string) int {
	if len(args) < 1 {
		ui.Errf("gg split <commit> [paths...]")
		return 2
	}
	if !HistorySupported() {
		ui.Errf("gg: split needs git >= 2.56 (git history split)")
		return 1
	}
	argv := []string{"history", "split", args[0]}
	if len(args) > 1 {
		argv = append(argv, "--")
		argv = append(argv, args[1:]...)
	}
	return gitx.Run(argv...)
}

// HistorySupported reports whether git ships the experimental history command.
func HistorySupported() bool {
	out, err := gitx.Output("version")
	if err != nil {
		return false
	}
	fields := strings.Fields(out)
	if len(fields) < 3 {
		return false
	}
	parts := strings.SplitN(fields[2], ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	return major > 2 || (major == 2 && minor >= 56)
}

// Edit stops the rebase at a commit so it can be amended.
func Edit(args []string) int {
	if len(args) < 1 {
		ui.Errf("gg edit <commit>")
		return 2
	}
	return stopAt(args[0], "edit")
}

// Squash folds a commit into its parent, keeping the parent message.
func Squash(args []string) int {
	if len(args) < 1 {
		ui.Errf("gg squash <commit>")
		return 2
	}
	return stopAt(args[0], "fixup")
}

// Abort unsticks whichever operation is in progress: rebase, merge, cherry-pick
// or revert.
func Abort(args []string) int {
	kinds := []struct {
		state  string
		abort  []string
		reason string
	}{
		{"rebase-merge", []string{"rebase", "--abort"}, "rebase"},
		{"rebase-apply", []string{"rebase", "--abort"}, "rebase"},
		{"MERGE_HEAD", []string{"merge", "--abort"}, "merge"},
		{"CHERRY_PICK_HEAD", []string{"cherry-pick", "--abort"}, "cherry-pick"},
		{"REVERT_HEAD", []string{"revert", "--abort"}, "revert"},
	}
	for _, k := range kinds {
		path, err := gitx.Output("rev-parse", "--git-path", k.state)
		if err != nil {
			continue
		}
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}
		ui.Errf("gg: aborting the %s", k.reason)
		return gitx.Run(k.abort...)
	}
	ui.Errf("gg: nothing to abort")
	return 1
}

// Continue resumes a rebase stopped by gg edit.
func Continue(args []string) int {
	return gitx.Run("rebase", "--continue")
}

func stopAt(commit, action string) int {
	dir, err := gitx.Output("rev-parse", "--git-path", "rebase-merge")
	if err == nil {
		if _, statErr := os.Stat(dir); statErr == nil {
			ui.Errf("gg: a rebase is already in progress; use gg continue or gg abort")
			return 1
		}
	}

	sha, err := gitx.Output("rev-parse", "--verify", commit+"^{commit}")
	if err != nil || sha == "" {
		ui.Errf("gg: %s does not name a commit", commit)
		return 1
	}
	commit = sha

	self, err := os.Executable()
	if err != nil {
		ui.Errf("gg: cannot find my own binary: %v", err)
		return 1
	}
	editor := fmt.Sprintf("%s __seq-edit %s %s", self, action, commit)
	cmd := exec.Command("git", "rebase", "-i", "--autostash", "--empty=drop", commit+"^")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GIT_SEQUENCE_EDITOR="+editor)
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

// SeqEdit rewrites a rebase todo list. Git calls it as:
//
//	gg __seq-edit <action> <commit> <todo-file>
func SeqEdit(args []string) int {
	if len(args) < 3 {
		ui.Errf("gg __seq-edit: internal command")
		return 2
	}
	action, commit, path := args[0], args[1], args[2]

	data, err := os.ReadFile(path)
	if err != nil {
		ui.Errf("gg: cannot read the rebase plan: %v", err)
		return 1
	}

	lines := strings.Split(string(data), "\n")
	matched := false
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if !sameCommit(fields[1], commit) {
			continue
		}
		fields[0] = action
		lines[i] = strings.Join(fields, " ")
		matched = true
	}
	if !matched {
		ui.Errf("gg: %s is not inside the rebase range", commit)
		return 1
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		ui.Errf("gg: cannot write the rebase plan: %v", err)
		return 1
	}
	return 0
}

func sameCommit(a, b string) bool {
	n := 7
	if len(a) < n {
		n = len(a)
	}
	if len(b) < n {
		n = len(b)
	}
	return n > 0 && a[:n] == b[:n]
}

const hookMarker = "# gg: keep a stable Change-Id so changeIDs survive rewrites"

const hookScript = `#!/bin/sh
` + hookMarker + `
msg="$1"
grep -q '^Change-Id:' "$msg" && exit 0
printf '\nChange-Id: I%s\n' "$(od -An -tx1 -N20 /dev/urandom | tr -d ' \n')" >> "$msg"
`

// InstallHooks writes the commit-msg hook that gives every new commit a
// Change-Id trailer, which is what makes changeIDs stable across rewrites.
func InstallHooks(args []string) int {
	if !gitx.InRepo() {
		ui.Errf("gg: not inside a git repository")
		return 1
	}
	if custom, err := gitx.Output("config", "--get", "core.hooksPath"); err == nil && custom != "" {
		ui.Errf("gg: core.hooksPath is %q, install the hook there yourself", custom)
		return 1
	}
	dir, err := gitx.Output("rev-parse", "--git-path", "hooks")
	if err != nil {
		ui.Errf("gg: cannot locate the hooks directory")
		return 1
	}
	if !filepath.IsAbs(dir) {
		root, err := gitx.Output("rev-parse", "--show-toplevel")
		if err != nil {
			return 1
		}
		dir = filepath.Join(root, dir)
	}
	path := filepath.Join(dir, "commit-msg")

	if existing, err := os.ReadFile(path); err == nil {
		if !strings.Contains(string(existing), hookMarker) {
			ui.Errf("gg: %s already exists and is not mine; leaving it alone", path)
			return 1
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		ui.Errf("gg: cannot create %s: %v", dir, err)
		return 1
	}
	if err := os.WriteFile(path, []byte(hookScript), 0o755); err != nil {
		ui.Errf("gg: cannot write %s: %v", path, err)
		return 1
	}
	fmt.Println("installed " + path)
	return 0
}
