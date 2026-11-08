// Package ops holds gg verbs that need more than one git call.
package ops

import (
	"fmt"
	"os"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Update fetches every remote and rebases the current branch on its upstream.
func Update(args []string) int {
	check := false
	if len(args) > 0 && (args[0] == "--check" || args[0] == "-c") {
		check = true
		args = args[1:]
	}
	if rc := gitx.Run("fetch", "--prune", "--all"); rc != 0 {
		return rc
	}
	if check {
		return updateCheck()
	}
	return gitx.Run(append([]string{"pull", "--rebase", "--autostash"}, args...)...)
}

// updateCheck reports what update would do, the way but pull --check does:
// how far the upstream is, and whether the rebase replays without conflict.
func updateCheck() int {
	up, ok := gitx.Upstream()
	if !ok {
		ui.Errf("gg update --check: this branch has no upstream")
		return 1
	}
	behind, _ := gitx.Count("HEAD.." + up)
	ahead, _ := gitx.Count(up + "..HEAD")
	state := ui.Green("clean")
	if _, err := gitx.Output("merge-tree", "--write-tree", "HEAD", up); err != nil {
		state = ui.Red("conflicts")
	}
	fmt.Printf("  %s  %d behind · %d ahead   rebase: %s\n", up, behind, ahead, state)
	return 0
}

// Sync rebases the current branch onto a named remote branch, then publishes it.
func Sync(args []string) int {
	if wipGuard() {
		return 1
	}
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

// Push pushes. When the upstream holds commits HEAD does not, gg forces on its
// own only when the force would drop nothing but the user's own commits and no
// other remote branch holds them. Anything else asks, the way it always has.
func Push(args []string) int {
	if wipGuard() {
		return 1
	}
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
		if why, safe := safeForce(up); safe {
			ui.Errf("gg: %s holds %d commit(s) HEAD rewrote; retrying with lease.", up, behind)
			if rc := forcePush(args); rc == 0 {
				return 0
			}
			// the lease or --force-if-includes refused: the rewrite is not a pure one
		} else if why != "" {
			ui.Errf("gg: not forcing: %s", why)
		}
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

// safeForce reports whether a force push would drop only the user's own commits,
// on no other remote branch. The lease already covers races; this covers
// ownership, which no force flag can see.
func safeForce(up string) (why string, safe bool) {
	lines, err := gitx.OutputAll("log", "--format=%H %ae", "HEAD.."+up)
	if err != nil || len(lines) == 0 {
		return "", false
	}
	me, _ := gitx.Output("config", "user.email")
	upRef := "refs/remotes/" + up
	for _, line := range lines {
		sha, mail, _ := strings.Cut(line, " ")
		if me == "" || mail != me {
			return "it would drop " + abbrev(sha) + " by " + mail, false
		}
		refs, _ := gitx.Output("for-each-ref", "--format=%(refname)", "--contains", sha, "refs/remotes")
		for _, ref := range strings.Split(refs, "\n") {
			if ref == "" || ref == upRef || strings.HasSuffix(ref, "/HEAD") {
				continue
			}
			return ref + " also holds " + abbrev(sha), false
		}
	}
	return "", true
}

// forcePush rewrites the upstream, but only while the lease still holds.
func forcePush(args []string) int {
	return gitx.Run(append([]string{"push", "--force-with-lease", "--force-if-includes"}, args...)...)
}

// abbrev shortens a sha for a one-line message.
func abbrev(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// wipGuard refuses to publish the wip commit.
func wipGuard() bool {
	if !gitx.InWip() {
		return false
	}
	ui.Errf("gg: HEAD is a wip commit. Run gg c -m \"msg\" to finish it, or gg uncommit to drop it.")
	return true
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
		return forcePush(args)
	default:
		ui.Errf("gg: aborted, nothing pushed")
		return 1
	}
}

// Uncommit moves commits back into the working tree: the change stays, the
// commit goes, and its descendants are replayed on top. With no argument it
// uncommits HEAD, the way but does.
func Uncommit(args []string) int {
	defer gitx.ClearStaleWip()
	if len(args) == 0 {
		return uncommitHead()
	}
	for _, arg := range args {
		if rc := uncommitOne(arg); rc != 0 {
			return rc
		}
	}
	return 0
}

// uncommitHead drops HEAD and leaves its change in the tree, unstaged: gg has
// no staging step, so nothing may arrive in the index.
func uncommitHead() int {
	if _, err := gitx.Output("rev-parse", "--verify", "--quiet", "HEAD~1"); err != nil {
		ui.Errf("gg uncommit: HEAD has no parent to uncommit into")
		return 1
	}
	return gitx.Run("reset", "--quiet", "HEAD~1")
}

// uncommitOne replays the history without one commit and hands its diff back
// to the tree. The patch is applied with three-way merge, so a later conflict
// stops there instead of losing the change.
func uncommitOne(arg string) int {
	sha, err := gitx.Output("rev-parse", "--verify", "--quiet", arg+"^{commit}")
	if err != nil {
		ui.Errf("gg uncommit: %s is not a commit any more; rerun with a fresh changeId", arg)
		return 1
	}
	head, err := gitx.Output("rev-parse", "HEAD")
	if err != nil {
		ui.Errf("gg uncommit: cannot read HEAD")
		return 1
	}
	if sha == head {
		return uncommitHead()
	}
	if _, err := gitx.Output("rev-parse", "--verify", "--quiet", sha+"^2"); err == nil {
		ui.Errf("gg uncommit: %s is a merge, uncommit one side instead", arg)
		return 1
	}
	if rc := gitx.Run("merge-base", "--is-ancestor", sha, head); rc != 0 {
		ui.Errf("gg uncommit: %s is not in the history of HEAD", arg)
		return 1
	}
	if n := dirty("."); n > 0 {
		ui.Errf("gg uncommit: the tree is dirty, commit or discard first")
		return 1
	}

	patch, err := gitx.Output("diff", sha+"^", sha)
	if err != nil || patch == "" {
		ui.Errf("gg uncommit: %s carries no change to move back", arg)
		return 1
	}
	file, err := os.CreateTemp("", "gg-uncommit-*.patch")
	if err != nil {
		ui.Errf("gg uncommit: cannot write a patch")
		return 1
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(patch + "\n"); err != nil {
		file.Close()
		ui.Errf("gg uncommit: cannot write a patch")
		return 1
	}
	file.Close()

	if rc := gitx.Run("rebase", "--onto", sha+"^", sha); rc != 0 {
		keepPatch(patch, file.Name())
		return rc
	}
	if rc := gitx.Run("apply", "--3way", file.Name()); rc != 0 {
		ui.Errf("gg uncommit: the change of %s did not apply cleanly; fix it by hand", arg)
		return rc
	}
	return gitx.Run("reset", "--quiet")
}

// keepPatch leaves the dropped change inside .git when a rebase stopped on a
// conflict, so the work survives the resolution.
func keepPatch(patch, tmp string) {
	path, err := gitx.Output("rev-parse", "--git-path", "gg-uncommit.patch")
	if err != nil {
		path = tmp
	} else if writeErr := os.WriteFile(path, []byte(patch+"\n"), 0o644); writeErr != nil {
		path = tmp
	}
	ui.Errf("gg uncommit: the change is kept in %s; apply it after the rebase", path)
}
