package ops

import (
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/oplog"
	"gg/internal/ui"
)

// Branches lists branches newest first, each with its own branch changeId, or
// switches to one. It never creates a branch by accident: only "branch -c"
// (or its old "branch new" form) and "cbr" create, so a mistyped name fails
// with a direction instead of silently branching off HEAD.
func Branches(args []string) int {
	if len(args) == 0 {
		return listBranches()
	}
	if strings.HasPrefix(args[0], "-") {
		if args[0] == "-d" || args[0] == "-D" || args[0] == "--delete" {
			ui.Errf("gg branch %s is gone. Use gg discard <branch>, with -f to drop unmerged commits.", args[0])
			return 2
		}
		if createFlag(args[0]) {
			if len(args) < 2 {
				ui.Errf("gg branch -c <name>: name the branch to create")
				return 2
			}
			return createBranch(args[1:])
		}
		switch args[0] {
		case "-a", "--stacks", "--stack":
			// -a reads the stacks; the plain local-plus-remote list lives in -la
			return Stack()
		case "-la", "--all":
			return gitx.Run(append([]string{"branch", "-a"}, args[1:]...)...)
		case "-lm", "--merged":
			return MergedBranches(args[1:])
		case "-lr", "--remotes":
			return gitx.Run(append([]string{"branch", "-r"}, args[1:]...)...)
		case "-g", "--global":
			ui.Errf("gg br -g is now -la: local and remote branches in one list")
			return 2
		case "-r":
			ui.Errf("gg br -r is now -lr: remote branches")
			return 2
		}
		// other flags belong to git branch: -r, -v, --merged and friends
		return gitx.Run(append([]string{"branch"}, args...)...)
	}
	if args[0] == "new" {
		if len(args) < 2 {
			ui.Errf("gg branch new <name>: name the branch to create")
			return 2
		}
		return createBranch(args[1:])
	}

	target := args[0]
	if !branchExists(target) && !revExists(target) {
		ui.Errf("gg branch: no branch or commit named %s. gg cbr %s creates it.", target, target)
		return 2
	}
	oplog.Record("branch")
	WipAmend()
	return gitx.Run(append([]string{"switch"}, args...)...)
}

// branchRow is one local branch, parsed from a for-each-ref line.
type branchRow struct {
	name     string
	sha      string
	upstream string
	current  bool
}

// branchRows lists local branches newest first.
func branchRows() []branchRow {
	lines, err := gitx.OutputAll("for-each-ref",
		"--sort=-committerdate", "refs/heads",
		"--format=%(refname:short)%1f%(objectname)%1f%(upstream:short)%1f%(HEAD)")
	if err != nil {
		return nil
	}
	rows := make([]branchRow, 0, len(lines))
	for _, l := range lines {
		parts := strings.Split(l, "\x1f")
		if len(parts) < 3 {
			continue
		}
		rows = append(rows, branchRow{
			name:     parts[0],
			sha:      parts[1],
			upstream: parts[2],
			current:  len(parts) > 3 && parts[3] == "*",
		})
	}
	return rows
}

// branchLine renders one branch with its changeId, its upstream drift and,
// when it is the default branch, a mark for it.
func branchLine(r branchRow, nameWidth int, bidx *ids.BranchIndex, def string) string {
	marker := " "
	if r.current {
		marker = "*"
	}
	line := fmt.Sprintf("  %s %s %-*s", ui.Green(marker), ui.Red(bidx.ChangeID(r.name)), nameWidth, r.name)
	if r.upstream == "" {
		line += "  " + ui.Dim("no upstream")
	} else {
		ahead, _ := gitx.Count(r.upstream + ".." + r.sha)
		behind, _ := gitx.Count(r.sha + ".." + r.upstream)
		line += "  " + ui.Dim(fmt.Sprintf("↑%d ↓%d", ahead, behind))
	}
	if def != "" && r.name == def {
		line += "  " + ui.Dim("default")
	}
	return line
}

func listBranches() int {
	rows := branchRows()
	if len(rows) == 0 {
		ui.Errf("gg branch: no branches yet")
		return 1
	}

	bidx := branchIndexOrEmpty()
	nameWidth := widestName(rows)
	for _, r := range rows {
		fmt.Println(branchLine(r, nameWidth, bidx, ""))
	}
	return 0
}

// recentBranches prints the newest branches and always the default branch, so
// a branch operation ends with the list it just changed.
func recentBranches(limit int) {
	rows := branchRows()
	if len(rows) == 0 {
		return
	}

	def := DefaultBranch()
	shown := rows
	if len(rows) > limit {
		shown = rows[:limit]
	}
	if def != "" && !holdsName(shown, def) {
		for _, r := range rows {
			if r.name == def {
				shown = append(append([]branchRow{}, shown...), r)
				break
			}
		}
	}

	bidx := branchIndexOrEmpty()
	nameWidth := widestName(shown)
	fmt.Println(ui.Dim(fmt.Sprintf("  branches (%d newest, default always):", limit)))
	for _, r := range shown {
		fmt.Println(branchLine(r, nameWidth, bidx, def))
	}
}

// MergedBranches lists the branches already merged into target (HEAD by
// default), with the same changeIds and upstream drift the plain list shows.
func MergedBranches(args []string) int {
	target := "HEAD"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		target = args[0]
	}
	out, err := gitx.OutputAll("branch", "--merged", target, "--format=%(refname:short)")
	if err != nil {
		ui.Errf("gg br -lm: %s is not a commit", target)
		return 1
	}
	merged := make(map[string]bool, len(out))
	for _, name := range out {
		merged[strings.TrimSpace(name)] = true
	}
	rows := branchRows()
	shown := make([]branchRow, 0, len(rows))
	for _, r := range rows {
		if merged[r.name] {
			shown = append(shown, r)
		}
	}
	if len(shown) == 0 {
		ui.Errf("gg br -lm: no branch is merged into %s", target)
		return 0
	}
	bidx := branchIndexOrEmpty()
	nameWidth := widestName(shown)
	fmt.Println(ui.Dim("  merged into " + target + ":"))
	for _, r := range shown {
		fmt.Println(branchLine(r, nameWidth, bidx, ""))
	}
	return 0
}

func branchIndexOrEmpty() *ids.BranchIndex {
	bidx, err := ids.LoadBranches()
	if err != nil {
		return &ids.BranchIndex{}
	}
	return bidx
}

func widestName(rows []branchRow) int {
	width := 0
	for _, r := range rows {
		if len(r.name) > width {
			width = len(r.name)
		}
	}
	return width
}

func holdsName(rows []branchRow, name string) bool {
	for _, r := range rows {
		if r.name == name {
			return true
		}
	}
	return false
}

// DefaultBranch names the branch a repository is expected to keep: the one
// origin points HEAD at, else a conventional name, else the current branch.
func DefaultBranch() string {
	if name := ConventionBranch(); name != "" {
		return name
	}
	if cur, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD"); err == nil && cur != "HEAD" {
		return cur
	}
	return ""
}

// ConventionBranch names the default branch only when the repository says so:
// origin points HEAD at it, or it carries a conventional name. It returns ""
// rather than fall back to the current branch, because a caller that treats
// the default as fixed ground cannot use a name that moves with HEAD.
func ConventionBranch() string {
	if out, err := gitx.Output("symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil && out != "" {
		return strings.TrimPrefix(out, "origin/")
	}
	for _, name := range []string{"main", "master", "trunk", "develop"} {
		if branchExists(name) {
			return name
		}
	}
	return ""
}

// revExists reports whether the token resolves to any commit.
func revExists(token string) bool {
	_, err := gitx.Output("rev-parse", "--verify", "--quiet", token)
	return err == nil
}

// createBranch makes a branch and switches to it, leaving the rest of the
// arguments to git switch -c.
func createBranch(args []string) int {
	oplog.Record("branch")
	WipAmend()
	return gitx.Run(append([]string{"switch", "-c"}, args...)...)
}

// createFlag reports whether the first argument asks for a new branch.
func createFlag(a string) bool {
	switch a {
	case "-c", "--create":
		return true
	}
	return false
}
