// Package dash renders the default gg view: branch, dirty files, commits.
package dash

import (
	"fmt"
	"strings"

	"gg/internal/files"
	"gg/internal/gitx"
	"gg/internal/hist"
	"gg/internal/ids"
	"gg/internal/ui"
)

const (
	defaultCommits  = 6
	defaultBranches = 5
)

// Run prints the dashboard. An optional argument limits the commit list.
func Run(args []string) int {
	commits := defaultCommits
	if len(args) > 0 {
		if n, err := parsePositive(args[0]); err == nil {
			commits = n
		}
	}

	sha, ok := gitx.Head()
	if !ok {
		ui.Errf("gg: no commits yet")
		return 1
	}
	fidx := files.Load()
	_ = fidx
	idx, err := ids.Load()
	if err != nil {
		idx = &ids.Index{}
	}
	fidx = files.Load()

	header(idx, sha)
	fileList(fidx)
	commitList(commits)
	branchList(idx)
	return 0
}

func parsePositive(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("not a positive number")
	}
	return n, nil
}

func header(idx *ids.Index, sha string) {
	branch, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		branch = "detached"
	}

	ahead, behind := "-", "-"
	if up, ok := gitx.Upstream(); ok {
		if n, ok := gitx.Count(up + "..HEAD"); ok {
			ahead = fmt.Sprint(n)
		}
		if n, ok := gitx.Count("HEAD.." + up); ok {
			behind = fmt.Sprint(n)
		}
	}

	line := fmt.Sprintf("%s  %s %s  %s %s",
		ui.Cyan(branch),
		ui.Red(idx.Handle(sha)), ui.Dim(sha[:7]),
		ui.Green("↑"+ahead), ui.Yellow("↓"+behind))
	if gitx.InWip() {
		line += "  " + ui.Magenta("wip")
	}
	fmt.Println(line)
}

func fileList(fidx *files.Index) {
	lines, err := gitx.OutputAll("status", "--short")
	if err != nil || len(lines) == 0 {
		fmt.Println(ui.Dim("\nclean"))
		return
	}
	fmt.Println()
	fmt.Println(ui.Dim("files"))
	for _, l := range lines {
		if strings.HasPrefix(l, "## ") {
			continue
		}
		code, path := l[:2], strings.TrimSpace(l[3:])
		handle := ui.Red(fmt.Sprintf("%-2s", fidx.Handle(path)))
		fmt.Printf("  %s %s %s\n", handle, paint(code), path)
	}
}

// paint colours the two status columns.
func paint(line string) string {
	if len(line) < 2 {
		return line
	}
	code, rest := line[:2], line[2:]
	if strings.Contains(code, "?") {
		return ui.Dim(code) + rest
	}
	return ui.Cyan(code) + rest
}

func commitList(n int) {
	lines := hist.Block(n, false)
	if len(lines) == 0 {
		return
	}
	fmt.Println()
	fmt.Println(ui.Dim("commits"))
	for _, l := range lines {
		fmt.Println("  " + l)
	}
}

func branchList(idx *ids.Index) {
	lines, err := gitx.OutputAll("branch",
		"--format=%(refname:short)|%(objectname)|%(upstream:short)|%(HEAD)")
	if err != nil || len(lines) == 0 {
		return
	}

	// current branch first, then the rest in git order
	var current, others []string
	for _, l := range lines {
		if strings.HasSuffix(l, "|*") {
			current = append(current, l)
		} else {
			others = append(others, l)
		}
	}
	ordered := append(current, others...)
	if len(ordered) > defaultBranches {
		ordered = ordered[:defaultBranches]
	}

	fmt.Println()
	fmt.Println(ui.Dim("branches"))
	for _, l := range ordered {
		parts := strings.Split(l, "|")
		name, sha, upstream := parts[0], parts[1], parts[2]
		marker := " "
		if len(parts) > 3 && parts[3] == "*" {
			marker = "*"
		}
		line := fmt.Sprintf("  %s %s %-14s", marker, ui.Red(idx.Handle(sha)), name)
		switch {
		case upstream == "":
			line += ui.Dim("no upstream")
		default:
			ahead, _ := gitx.Count(upstream + ".." + sha)
			behind, _ := gitx.Count(sha + ".." + upstream)
			line += ui.Dim(fmt.Sprintf("↑%d ↓%d", ahead, behind))
		}
		fmt.Println(line)
	}
}
