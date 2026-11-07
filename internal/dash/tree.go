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

// maxLanes caps how many branch lanes the tree draws.
const maxLanes = 4

// lane is one branch drawn in the tree. Parallel branches are real worktrees:
// every branch a worktree holds gets its own lane.
type lane struct {
	name    string
	tip     string
	commits []string
}

// Tree prints the but-style tree: the uncommitted files, one lane per branch,
// the common base and two hints. An optional argument sets the commits per lane.
func Tree(args []string) int {
	n := defaultCommits
	if len(args) > 0 {
		if v, err := parsePositive(args[0]); err == nil {
			n = v
		}
	}

	sha, ok := gitx.Head()
	if !ok {
		ui.Errf("gg: no commits yet")
		return 1
	}

	lanes, base := lanesOf(sha, n)

	fileLane(files.Load())
	fmt.Println(ui.Dim("┊"))

	for i, l := range lanes {
		head := "┊├┄"
		if i == 0 {
			head = "┊╭┄"
		}
		fmt.Println(ui.Dim(head) + " " + ui.Red(branchHandle(l.name)) + " " + ui.Dim("["+l.name+"]"))
		for _, row := range l.commits {
			fmt.Println(ui.Dim("┊") + row)
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
	fmt.Println(ui.Dim("╭┄") + " " + ui.Red(uncommittedHandle()) + " " + ui.Dim("[uncommitted]") + emptyNote(len(rows)))
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

// lanesOf builds the branch lanes and the commit they share as a base.
func lanesOf(sha string, n int) ([]lane, string) {
	current, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if current == "HEAD" {
		current = ""
	}
	base := baseFor(current)

	names := []string{current}
	if current == "" {
		names = []string{"detached"}
	}
	if current != "" {
		trees := map[string]bool{}
		for _, b := range ops.WorktreeBranches() {
			trees[b] = true
		}
		for _, b := range localBranches() {
			if len(names) >= maxLanes {
				break
			}
			if b.name != current && trees[b.name] {
				names = append(names, b.name)
			}
		}
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
		lanes = append(lanes, lane{name: name, tip: tip, commits: hist.BranchLines(tip, base, n)})
	}
	return lanes, base
}

// baseFor is the commit every lane branches from: the merge base of the
// current branch and the target, or the merge base with the upstream.
func baseFor(current string) string {
	for _, target := range []string{"gitbutler/target", "main", "master"} {
		if target == current {
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
		if name != "" {
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

// uncommittedHandle is the stable changeId of the uncommitted lane.
func uncommittedHandle() string {
	return ids.RawChangeID("branch:uncommitted")[:ids.MinLen]
}
