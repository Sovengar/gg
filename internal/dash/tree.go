// Package dash renders the branch tree: the default gg view and the two
// status views built on it.
package dash

import (
	"fmt"
	"strings"

	"gg/internal/files"
	"gg/internal/gitx"
	"gg/internal/hist"
	"gg/internal/ids"
	"gg/internal/ops"
	"gg/internal/ui"
)

const (
	// maxLanes caps how many branch lanes `gg st` draws.
	maxLanes = 4
	// dashLanes caps how many the default view draws: the newest branches.
	dashLanes = 5
	// maxFiles caps how many changed paths a commit lists in the full view.
	maxFiles = 5
	// defaultCommits is how many commits each lane shows.
	defaultCommits = 6
)

// lane is one branch drawn in the tree. Parallel branches are real worktrees:
// every branch a worktree holds gets its own lane.
type lane struct {
	name    string
	tip     string
	commits []hist.Line
}

// parsePositive reads a commit count from the first argument.
func parsePositive(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("not a positive number")
	}
	return n, nil
}

// view says which tree to draw: how many lanes, how they are picked and whether
// every commit lists the paths it changes.
type view struct {
	lanes  int
	newest bool // pick the newest branches, one of them the default
	files  bool
	all    bool // list every changed path, not just the first maxFiles
}

// Tree prints the but-style tree for `gg st`: the uncommitted files, one lane
// per worktree branch, the common base and two hints. An optional argument sets
// the commits per lane.
func Tree(args []string) int { return tree(args, view{lanes: maxLanes}) }

// TreeFull prints the same tree with the changed paths under every commit.
func TreeFull(args []string) int { return tree(args, view{lanes: maxLanes, files: true}) }

// TreeAll prints the full tree with every changed path under every commit.
func TreeAll(args []string) int {
	return tree(args, view{lanes: maxLanes, files: true, all: true})
}

// Run prints the default gg view: the tree over the newest branch lanes. The
// branch header is left to starship, which already shows it.
func Run(args []string) int { return tree(args, view{lanes: dashLanes, newest: true}) }

func tree(args []string, opt view) int {
	n := defaultCommits
	for _, a := range args {
		switch a {
		case "-a", "--all":
			opt.files, opt.all = true, true
			continue
		case "-f", "--files":
			opt.files = true
			continue
		}
		if v, err := parsePositive(a); err == nil {
			n = v
		}
	}

	sha, ok := gitx.Head()
	if !ok {
		ui.Errf("gg: no commits yet")
		return 1
	}

	lanes, base := lanesOf(sha, n, opt)

	fileLane(files.Load())
	fmt.Println(ui.Dim("┊"))

	for i, l := range lanes {
		head := "┊├┄"
		if i == 0 {
			head = "┊╭┄"
		}
		fmt.Println(ui.Dim(head) + " " + ui.Red(branchHandle(l.name)) + " " + ui.Dim("["+l.name+"]"))
		for _, row := range l.commits {
			fmt.Println(ui.Dim("┊") + row.Text)
			if opt.files {
				for _, frow := range changedFiles(row.Sha, opt.all) {
					fmt.Println(ui.Dim("┊") + "   " + frow)
				}
			}
		}
		if opt.newest && len(l.commits) == 0 {
			fmt.Println(ui.Dim("┊") + "   " + ui.Dim("(at the common base)"))
		}
		if i < len(lanes)-1 {
			fmt.Println(ui.Dim("┊│"))
		}
	}
	if len(lanes) > 0 {
		fmt.Println(ui.Dim("├╯"))
		fmt.Println(ui.Dim("┊"))
	}

	if base != "" {
		fmt.Println(baseLine(base))
	}

	fmt.Println()
	hints()
	return 0
}

// fileLane prints the uncommitted section: one row per dirty path, every path
// carrying the changeId its verb takes.
func fileLane(fidx *files.Index) {
	rows := fileRows(fidx)
	head := ui.Dim("╭┄") + " " + ui.Red(uncommittedHandle()) + " " + ui.Dim("[uncommitted]")
	if gitx.InWip() {
		head += " " + ui.Yellow("wip")
	}
	fmt.Println(head + emptyNote(len(rows)))
	for _, row := range rows {
		fmt.Println(ui.Dim("┊") + "   " + row)
	}
}

func emptyNote(n int) string {
	if n > 0 {
		return ""
	}
	return " " + ui.Dim("(no changes)")
}

// fileRows renders every dirty path with a changeId and one status letter.
// Untracked paths read A, like a branch that adds the file.
func fileRows(fidx *files.Index) []string {
	lines, err := gitx.OutputAll("status", "--short", "--untracked-files=all")
	if err != nil {
		return nil
	}

	type row struct{ handle, status, path string }
	var rows []row
	width := ids.MinLen
	for _, l := range lines {
		if strings.HasPrefix(l, "## ") || len(l) < 4 {
			continue
		}
		handle := fidx.ChangeID(strings.TrimSpace(l[3:]))
		if len(handle) > width {
			width = len(handle)
		}
		rows = append(rows, row{handle, fileStatus(l[:2]), strings.TrimSpace(l[3:])})
	}

	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, ui.Red(fmt.Sprintf("%-*s", width, r.handle))+" "+statusLetter(r.status)+" "+r.path)
	}
	return out
}

// fileStatus folds git's two status columns into one letter: untracked and
// added paths read A, otherwise the index status wins over the worktree one.
func fileStatus(code string) string {
	if strings.ContainsRune(code, '?') {
		return "A"
	}
	if code[0] != ' ' {
		return string(code[0])
	}
	return string(code[1])
}

func statusLetter(s string) string {
	switch s {
	case "A":
		return ui.Green(s)
	case "D":
		return ui.Red(s)
	}
	return ui.Cyan(s)
}

// changedFiles lists the paths one commit touches. Without all it stops at
// maxFiles and closes with the count of the paths the cap hides.
func changedFiles(sha string, all bool) []string {
	lines, err := gitx.OutputAll("show", "--name-status", "--no-renames", "--format=", sha)
	if err != nil {
		return nil
	}
	cap := maxFiles
	if all {
		cap = len(lines)
	}
	rows := make([]string, 0, cap)
	hidden := 0
	for _, l := range lines {
		code, path, ok := strings.Cut(l, "\t")
		code = strings.TrimSpace(code)
		if !ok || path == "" || code == "" {
			continue
		}
		if len(rows) >= cap {
			hidden++
			continue
		}
		rows = append(rows, statusLetter(code[:1])+" "+path)
	}
	if hidden > 0 {
		rows = append(rows, ui.Dim(fmt.Sprintf("+%d more", hidden)))
	}
	return rows
}

// lanesOf builds the branch lanes and the commit they share as a base.
func lanesOf(sha string, n int, opt view) ([]lane, string) {
	current, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if current == "HEAD" {
		current = ""
	}
	base := baseFor()

	var names []string
	switch {
	case current == "":
		names = []string{"detached"}
	case opt.newest:
		names = newestLanes(current, opt.lanes)
	default:
		names = worktreeLanes(current, opt.lanes)
	}

	lanes := make([]lane, 0, len(names))
	for i, name := range names {
		tip := sha
		if name != current && name != "detached" {
			if t, err := gitx.Output("rev-parse", "--verify", "refs/heads/"+name); err == nil {
				tip = t
			}
		}
		if i == 0 && name == "detached" {
			lanes = append(lanes, lane{name: name, tip: tip, commits: hist.BranchLines(tip, "", n)})
			continue
		}
		commits := hist.BranchLines(tip, base, n)
		// a branch already sitting on the base still shows its own last commits
		if len(commits) == 0 && i == 0 {
			commits = hist.BranchLines(tip, "", n)
		}
		lanes = append(lanes, lane{name: name, tip: tip, commits: commits})
	}
	// a lane with nothing above the base adds no row: but prints only the base.
	// the default view keeps it, so the default branch always has its lane.
	kept := lanes[:0]
	for _, l := range lanes {
		if len(l.commits) > 0 || opt.newest {
			kept = append(kept, l)
		}
	}
	return kept, base
}

// worktreeLanes picks the lanes `gg st` draws: the current branch, then the
// branches a sibling worktree holds, newest first.
func worktreeLanes(current string, limit int) []string {
	names := []string{current}
	trees := map[string]bool{}
	for _, b := range ops.WorktreeBranches() {
		trees[b] = true
	}
	for _, b := range localBranches() {
		if len(names) >= limit {
			break
		}
		if b.name != current && trees[b.name] {
			names = append(names, b.name)
		}
	}
	return names
}

// newestLanes picks the lanes the default view draws: the current branch first,
// then the most recently committed branches, always including the default one.
func newestLanes(current string, limit int) []string {
	def := defaultBranch()
	names := []string{current}
	for _, b := range localBranches() {
		if len(names) >= limit {
			break
		}
		if b.name == current {
			continue
		}
		// keep the last slot for the default branch when it is not the current one
		if len(names) == limit-1 && def != "" && def != current && b.name != def {
			continue
		}
		names = append(names, b.name)
	}
	if def != "" && def != current && !hasName(names, def) && len(names) < limit {
		names = append(names, def)
	}
	return names
}

// hasName reports whether names already holds the branch.
func hasName(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// defaultBranch names the branch every lane branches from.
func defaultBranch() string { return ops.DefaultBranch() }

// baseFor is the commit every lane branches from: the merge base of the
// current branch with but's target or the default branch, else with the
// upstream.
func baseFor() string {
	for _, target := range []string{"gitbutler/target", defaultBranch()} {
		if target == "" {
			continue
		}
		if _, err := gitx.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+target); err != nil {
			continue
		}
		if out, err := gitx.Output("merge-base", "HEAD", target); err == nil {
			return out
		}
	}
	if up, ok := gitx.Upstream(); ok {
		if out, err := gitx.Output("merge-base", "HEAD", up); err == nil {
			return out
		}
	}
	// no target and no upstream: the first commit is the base, so the tree
	// still closes on a base line instead of an unbounded lane
	if roots, err := gitx.OutputAll("rev-list", "--max-parents=0", "HEAD"); err == nil && len(roots) > 0 {
		return strings.TrimSpace(roots[len(roots)-1])
	}
	return ""
}

// baseLine is the closing line: the shared commit, its date and its subject.
func baseLine(sha string) string {
	out, err := gitx.Output("log", "-1", "--format=%h%x1f%cs%x1f%s", sha)
	if err != nil {
		return ui.Dim("┴ " + sha)
	}
	short, rest, _ := strings.Cut(out, "\x1f")
	date, subject, _ := strings.Cut(rest, "\x1f")
	return ui.Dim("┴ ") + ui.Dim(short) + ui.Dim(" (common base) ") + date + " " + subject
}

// hints is the two line trailer but prints after status.
func hints() {
	fmt.Println(ui.Dim("Hint: commits are listed newest first. The first token on each line is the changeId to use in commands."))
	fmt.Println(ui.Dim("Hint: run `gg diff` to see uncommitted changes and `gg c -m \"message\"` to commit them"))
}

type branch struct{ name, tip string }

// localBranches lists the local branches, newest first.
func localBranches() []branch {
	lines, err := gitx.OutputAll("for-each-ref", "--sort=-committerdate",
		"--format=%(refname:short)%1f%(objectname)", "refs/heads")
	if err != nil {
		return nil
	}
	out := make([]branch, 0, len(lines))
	for _, l := range lines {
		name, tip, _ := strings.Cut(l, "\x1f")
		// gitbutler's own refs are bookkeeping, not branches to draw
		if name != "" && !strings.HasPrefix(name, "gitbutler/") {
			out = append(out, branch{name, tip})
		}
	}
	return out
}

func branchHandle(name string) string {
	ix, err := ids.LoadBranches()
	if err != nil {
		return ids.RawChangeID("branch:" + name)[:ids.MinLen]
	}
	return ix.ChangeID(name)
}

// uncommittedHandle is the fixed changeId of the uncommitted lane, the same
// "zz" every but workspace prints.
func uncommittedHandle() string {
	return "zz"
}
