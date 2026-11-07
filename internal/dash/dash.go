// Package dash renders the default gg view: the header, then the tree.
package dash

import (
	"fmt"

	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/ui"
)

const defaultCommits = 6

// Run prints the dashboard: the branch header, then the tree.
func Run(args []string) int {
	sha, ok := gitx.Head()
	if !ok {
		ui.Errf("gg: no commits yet")
		return 1
	}
	idx, err := ids.Load()
	if err != nil {
		idx = &ids.Index{}
	}

	header(idx, sha)
	return Tree(args)
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
		ui.Red(idx.ChangeID(sha)), ui.Dim(sha[:7]),
		ui.Green("↑"+ahead), ui.Yellow("↓"+behind))
	if gitx.InWip() {
		line += "  " + ui.Magenta("wip")
	}
	fmt.Println(line)
	fmt.Println()
}
