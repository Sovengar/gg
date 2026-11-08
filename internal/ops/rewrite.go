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
		ui.Errf("gg describe <commit>")
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
// or revert. With --keep it stashes the live edits of an aborted rebase and
// restores them afterwards.
func Abort(args []string) int {
	keep := false
	for _, a := range args {
		if a == "--keep" || a == "-k" {
			keep = true
		}
	}
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
		if !keep || k.reason != "rebase" || !gitx.Dirty() {
			return gitx.Run(k.abort...)
		}
		if rc := gitx.Run("stash", "push", "-u", "-m", "gg abort --keep"); rc != 0 {
			return rc
		}
		rc := gitx.Run(k.abort...)
		if popRC := gitx.Run("stash", "pop"); popRC != 0 {
			ui.Errf("gg: your changes are in the stash (git stash list), the pop needs a manual fix")
		}
		return rc
	}
	ui.Errf("gg: nothing to abort")
	return 1
}

// Continue resumes a rebase stopped by gg edit.
func Continue(args []string) int {
	return gitx.Run("rebase", "--continue")
}

// StopAmend folds uncommitted edits into the commit where a rebase stopped for
// gg edit. It does nothing when no rebase is in progress, when the stop is not
// an edit point, when the index has conflicts, or when the tree is clean.
func StopAmend() {
	if !inSequence() {
		return
	}
	path, err := gitx.Output("rev-parse", "--git-path", "rebase-merge/done")
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if !stoppedOnEdit(string(data)) {
		return
	}
	if gitx.Unmerged() || !gitx.Dirty() {
		return
	}
	if rc := gitx.Run("add", "-A"); rc != 0 {
		return
	}
	if rc := gitx.Run("commit", "--amend", "--no-edit"); rc != 0 {
		return
	}
}

// stoppedOnEdit reports whether the rebase todo last ran an edit point: an
// edit command for a commit, or a break for right after a merge.
func stoppedOnEdit(done string) bool {
	lines := strings.Split(done, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line == "break" || strings.HasPrefix(line, "edit ")
	}
	return false
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

	merge := isMerge(commit)
	if merge && action != "edit" {
		ui.Errf("gg: %s is a merge commit; a merge cannot fold into its parent", commit)
		return 1
	}

	self, err := os.Executable()
	if err != nil {
		ui.Errf("gg: cannot find my own binary: %v", err)
		return 1
	}
	editor := fmt.Sprintf("%s __seq-edit %s %s", self, action, commit)
	argv := []string{"rebase", "-i", "--autostash", "--empty=drop"}
	if merge {
		// A plain rebase drops merge commits, so the target would never
		// appear in the plan. Keeping the merges puts it there.
		argv = append(argv, "--rebase-merges")
	}
	argv = append(argv, commit+"^")
	cmd := exec.Command("git", argv...)
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
	out := make([]string, 0, len(lines)+1)
	matched := false
	for _, line := range lines {
		out = append(out, line)
		if matched {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "merge" {
			// A merge line reads "merge -C <commit> <label>". git has no
			// edit for it, so stop the rebase right after the merge.
			if len(fields) < 3 || (fields[1] != "-C" && fields[1] != "-c") || !sameCommit(fields[2], commit) {
				continue
			}
			if action != "edit" {
				ui.Errf("gg: %s is a merge commit; a merge cannot fold into its parent", commit)
				return 1
			}
			out = append(out, "break")
			matched = true
			continue
		}
		if !sameCommit(fields[1], commit) {
			continue
		}
		fields[0] = action
		out[len(out)-1] = strings.Join(fields, " ")
		matched = true
	}
	if !matched {
		ui.Errf("gg: %s is not inside the rebase range", commit)
		return 1
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		ui.Errf("gg: cannot write the rebase plan: %v", err)
		return 1
	}
	return 0
}

// isMerge reports whether a commit has more than one parent.
func isMerge(sha string) bool {
	out, err := gitx.Output("rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return false
	}
	return len(strings.Fields(out)) > 2
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
