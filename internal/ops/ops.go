// Package ops holds gg verbs that need more than one git call.
package ops

import (
	"strconv"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Update fetches every remote and rebases the current branch on its upstream.
func Update(args []string) int {
	if rc := gitx.Run("fetch", "--prune", "--all"); rc != 0 {
		return rc
	}
	return gitx.Run(append([]string{"pull", "--rebase", "--autostash"}, args...)...)
}

// Sync rebases the current branch onto a named remote branch, then publishes it.
func Sync(args []string) int {
	if len(args) < 1 {
		ui.Errf("gg sync <branch>: name the branch to rebase against")
		return 2
	}
	target := args[0]
	if rc := gitx.Run("fetch", "--prune", "--all"); rc != 0 {
		return rc
	}
	if rc := gitx.Run("pull", "--rebase", "--autostash", "origin", target); rc != 0 {
		return rc
	}
	return Push(nil)
}

// Push pushes, but stops when the upstream moved and asks before forcing.
func Push(args []string) int {
	up, ok := gitx.Upstream()
	if !ok {
		ui.Errf("gg: no upstream, pushing with -u origin HEAD")
		return gitx.Run(append([]string{"push", "-u", "origin", "HEAD"}, args...)...)
	}

	remote, _, _ := strings.Cut(up, "/")
	_ = gitx.Run("fetch", "--quiet", remote)

	behind, _ := gitx.Count("HEAD.." + up)
	ahead, _ := gitx.Count(up + "..HEAD")

	if behind > 0 {
		ui.Errf("gg: %s has %d commit(s) you don't have (you are %d ahead):", up, behind, ahead)
		printRange("HEAD.." + up)
		return ask(up, args)
	}

	rc := gitx.Run(append([]string{"push"}, args...)...)
	if rc == 0 {
		return 0
	}

	// push failed: maybe the upstream moved between fetch and push
	_ = gitx.Run("fetch", "--quiet", remote)
	if behind, _ := gitx.Count("HEAD.." + up); behind > 0 {
		ui.Errf("gg: push rejected, %s moved while you were working:", up)
		printRange("HEAD.." + up)
		return ask(up, args)
	}
	return rc
}

// printRange lists the incoming commits on stderr, so it stays above the prompt.
func printRange(rng string) {
	lines, err := gitx.OutputAll("log", "--oneline", "--no-decorate", "-n", "10", rng)
	if err != nil {
		return
	}
	for _, line := range lines {
		ui.Errf("    %s", line)
	}
}

func ask(up string, args []string) int {
	choice, ok := ui.Prompt("gg: [u] update and push  [f] force-with-lease  [a] abort > ")
	if !ok {
		ui.Errf("gg: refusing to push non-interactively; run 'gg update' first")
		return 1
	}
	switch choice {
	case "u", "update":
		if rc := gitx.Run("pull", "--rebase", "--autostash"); rc != 0 {
			return rc
		}
		return gitx.Run(append([]string{"push"}, args...)...)
	case "f", "force":
		return gitx.Run(append([]string{"push", "--force-with-lease", "--force-if-includes"}, args...)...)
	default:
		ui.Errf("gg: aborted, nothing pushed")
		return 1
	}
}

// Uncommit moves the last N commits back into the working copy.
func Uncommit(args []string) int {
	n := 1
	if len(args) > 0 {
		parsed, err := strconv.Atoi(args[0])
		if err != nil || parsed < 1 {
			ui.Errf("gg: uncommit takes a positive count, got %q", args[0])
			return 2
		}
		n = parsed
	}
	if _, err := gitx.Output("rev-parse", "--verify", "--quiet", "HEAD~"+strconv.Itoa(n)); err != nil {
		ui.Errf("gg: not enough history to uncommit %d commit(s)", n)
		return 1
	}
	return gitx.Run("reset", "--soft", "HEAD~"+strconv.Itoa(n))
}
