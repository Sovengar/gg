package ops

import (
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/ui"
)

// Branches lists branches newest first, each with its own branch changeId, or
// switches to one. It never creates a branch by accident: only "branch new"
// and "cbr" create, so a mistyped name fails with a direction instead of
// silently branching off HEAD.
func Branches(args []string) int {
	if len(args) == 0 {
		return listBranches()
	}
	if strings.HasPrefix(args[0], "-") {
		if deleteFlag(args[0]) {
			return DeleteBranch(args)
		}
		// other flags belong to git branch: -a, -v, --merged and friends
		return gitx.Run(append([]string{"branch"}, args...)...)
	}
	if args[0] == "new" {
		if len(args) < 2 {
			ui.Errf("gg branch new <name>: name the branch to create")
			return 2
		}
		WipAmend()
		return gitx.Run(append([]string{"switch", "-c"}, args[1:]...)...)
	}

	target := args[0]
	if !branchExists(target) && !revExists(target) {
		ui.Errf("gg branch: no branch or commit named %s. gg cbr %s creates it.", target, target)
		return 2
	}
	WipAmend()
	return gitx.Run(append([]string{"switch"}, args...)...)
}

func listBranches() int {
	lines, err := gitx.OutputAll("for-each-ref",
		"--sort=-committerdate", "refs/heads",
		"--format=%(refname:short)%1f%(objectname)%1f%(upstream:short)%1f%(HEAD)")
	if err != nil || len(lines) == 0 {
		ui.Errf("gg branch: no branches yet")
		return 1
	}

	bidx, err := ids.LoadBranches()
	if err != nil {
		bidx = &ids.BranchIndex{}
	}

	nameWidth := 0
	for _, l := range lines {
		if name, _, _ := strings.Cut(l, "\x1f"); len(name) > nameWidth {
			nameWidth = len(name)
		}
	}

	for _, l := range lines {
		parts := strings.Split(l, "\x1f")
		name, sha, upstream := parts[0], parts[1], parts[2]
		marker := " "
		if len(parts) > 3 && parts[3] == "*" {
			marker = "*"
		}
		line := fmt.Sprintf("  %s %s %-*s", ui.Green(marker), ui.Red(bidx.ChangeID(name)), nameWidth, name)
		if upstream == "" {
			line += "  " + ui.Dim("no upstream")
		} else {
			ahead, _ := gitx.Count(upstream + ".." + sha)
			behind, _ := gitx.Count(sha + ".." + upstream)
			line += "  " + ui.Dim(fmt.Sprintf("↑%d ↓%d", ahead, behind))
		}
		fmt.Println(line)
	}
	return 0
}

// revExists reports whether the token resolves to any commit.
func revExists(token string) bool {
	_, err := gitx.Output("rev-parse", "--verify", "--quiet", token)
	return err == nil
}

// DeleteBranch drops a branch by name or changeId. It deletes the safe way
// unless asked to force, and it keeps the tip in refs/gg/deleted/<name>, so a
// branch that was not meant to go can be recreated from the changeId it prints.
func DeleteBranch(args []string) int {
	force := false
	var names []string
	for _, a := range args {
		switch a {
		case "-d", "--delete":
		case "-D", "-f", "--force":
			force = true
		default:
			if strings.HasPrefix(a, "-") {
				ui.Errf("gg dbr: unknown flag %s. Use -d (safe) or -D (force).", a)
				return 2
			}
			names = append(names, a)
		}
	}
	if len(names) == 0 {
		ui.Errf("gg dbr <branch>: name the branch to delete")
		return 2
	}

	current, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	rc := 0
	for _, name := range names {
		if name == current {
			ui.Errf("gg dbr: %s is the current branch; switch away first", name)
			rc = 1
			continue
		}
		if !branchExists(name) {
			ui.Errf("gg dbr: no branch named %s", name)
			rc = 1
			continue
		}
		sha, err := gitx.Output("rev-parse", "--verify", "refs/heads/"+name)
		if err != nil {
			ui.Errf("gg dbr: cannot read the tip of %s", name)
			rc = 1
			continue
		}
		flag := "-d"
		if force {
			flag = "-D"
		}
		if code := gitx.Run("branch", flag, name); code != 0 {
			if !force {
				ui.Errf("gg dbr: %s holds commits that are not merged. -D drops them.", name)
			}
			rc = code
			continue
		}
		gitx.Output("update-ref", "refs/gg/deleted/"+name, sha)
		ui.Errf("gg: %s is deleted. Its tip stays in refs/gg/deleted/%s", name, name)
		ui.Errf("gg: to bring it back: gg cbr %s %s", name, sha)
	}
	return rc
}

// deleteFlag reports whether the first argument asks for a deletion.
func deleteFlag(a string) bool {
	switch a {
	case "-d", "-D", "-f", "--delete", "--force":
		return true
	}
	return false
}
