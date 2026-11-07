package ops

import (
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// commitFlags take the next argument as their value.
var commitFlags = map[string]bool{
	"-m": true, "--message": true, "-F": true, "--file": true,
	"--author": true, "--date": true, "-t": true, "--template": true,
	"-C": true, "-c": true,
}

// Commit records the named changes, or every dirty path when none is named.
// The index stays an implementation detail: no verb asks the user to stage.
func Commit(args []string) int {
	flags, paths := splitCommitArgs(args)
	if gitx.InWip() {
		return finalizeWip(flags, paths)
	}
	if len(paths) == 0 {
		if rc := gitx.Run("add", "-A"); rc != 0 {
			return rc
		}
		return gitx.Run(append([]string{"commit"}, flags...)...)
	}
	markIntent(paths)
	argv := append([]string{"commit"}, flags...)
	return gitx.Run(append(append(argv, "--"), paths...)...)
}

// finalizeWip turns the wip commit into the real one and drops the marker.
func finalizeWip(flags, paths []string) int {
	if len(paths) > 0 {
		ui.Errf("gg c: HEAD is a wip commit. Name no path, or run gg uncommit first.")
		return 1
	}
	if !hasMessage(flags) {
		ui.Errf("gg c: a wip commit needs a message. Use gg c -m \"msg\", or gg uncommit to take it back.")
		return 2
	}
	if rc := gitx.Run("add", "-A"); rc != 0 {
		return rc
	}
	rc := gitx.Run(append([]string{"commit", "--amend"}, flags...)...)
	if rc == 0 {
		gitx.DropWip()
	}
	return rc
}

// hasMessage reports whether the flags already carry a commit message.
func hasMessage(flags []string) bool {
	for _, f := range flags {
		switch f {
		case "-m", "--message", "-F", "--file", "-C", "--no-edit", "--reuse-message":
			return true
		}
	}
	return false
}

// splitCommitArgs separates flags from paths.
func splitCommitArgs(args []string) (flags, paths []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return flags, append(paths, args[i+1:]...)
		case strings.HasPrefix(a, "-"):
			flags = append(flags, a)
			if commitFlags[a] && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
		default:
			paths = append(paths, a)
		}
	}
	return flags, paths
}

// markIntent makes an untracked path visible to `git commit -- <path>`.
func markIntent(paths []string) {
	for _, p := range paths {
		if tracked(p) {
			continue
		}
		_, _ = gitx.Output("add", "-N", "--", p)
	}
}

// tracked reports whether git already knows the path.
func tracked(path string) bool {
	_, err := gitx.Output("ls-files", "--error-unmatch", "--", path)
	return err == nil
}

// Discard drops edits to tracked files, and keeps new files: nothing in a
// commit holds them, so deleting one is not a discard. It is the but verb of
// the same name; `gg clear` owns the new files.
func Discard(args []string) int {
	var paths []string
	for _, a := range args {
		switch a {
		case "-a", "--all":
			ui.Errf("gg discard: no -a. A bare discard drops every edit already.")
			return 2
		case "-f", "--force":
			ui.Errf("gg discard: -f deletes nothing here. Use gg clear.")
			return 2
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		return gitx.Run("reset", "--hard", "--quiet")
	}

	var known, fresh []string
	for _, p := range paths {
		if tracked(p) {
			known = append(known, p)
			continue
		}
		fresh = append(fresh, p)
	}
	if len(fresh) > 0 {
		ui.Errf("gg discard: %d path(s) are new, so no change is there to drop:", len(fresh))
		for _, p := range fresh {
			ui.Errf("    %s", p)
		}
		ui.Errf("gg discard: nothing touched. gg clear deletes new files.")
		return 1
	}
	argv := append([]string{"restore", "--source=HEAD", "--staged", "--worktree", "--"}, known...)
	return gitx.Run(argv...)
}

// Clear drops every edit and deletes the new files, so the tree matches HEAD.
// No commit holds a new file, so git cannot bring one back: it asks first,
// and -f skips the question.
func Clear(args []string) int {
	force := false
	for _, a := range args {
		switch a {
		case "-f", "--force":
			force = true
		case "-a", "--all":
			ui.Errf("gg clear: no -a. It already means everything.")
			return 2
		default:
			ui.Errf("gg clear: it takes no path and no changeId, only -f to skip the prompt")
			return 2
		}
	}
	fresh := untracked()
	if len(fresh) > 0 && !force && !confirmClear(fresh) {
		return 1
	}
	if rc := gitx.Run("reset", "--hard", "--quiet"); rc != 0 {
		return rc
	}
	return gitx.Run("clean", "-f", "-d")
}

// untracked lists the new files git would delete, one per line.
func untracked() []string {
	out, err := gitx.Output("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil
	}
	return strings.Fields(out)
}

// confirmClear asks before deleting new paths, which no commit holds.
func confirmClear(paths []string) bool {
	ui.Errf("gg clear: %d new path(s) would be deleted, and git cannot restore them:", len(paths))
	for _, p := range paths {
		ui.Errf("    %s", p)
	}
	choice, ok := ui.Prompt("gg clear: delete them? [y/N] > ")
	if !ok {
		ui.Errf("gg clear: nothing deleted. Pass -f to drop new paths.")
		return false
	}
	return choice == "y" || choice == "yes"
}
