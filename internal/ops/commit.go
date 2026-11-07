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
		flags = append(flags, "--no-edit")
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

// Discard drops the named changes: tracked paths go back to HEAD, untracked
// files are deleted. It is the but verb of the same name.
func Discard(args []string) int {
	var paths []string
	all := false
	for _, a := range args {
		switch a {
		case "-a", "--all":
			all = true
		default:
			paths = append(paths, a)
		}
	}
	if all {
		if rc := gitx.Run("restore", "--source=HEAD", "--staged", "--worktree", "."); rc != 0 {
			return rc
		}
		return gitx.Run("clean", "-f", "-d")
	}
	if len(paths) == 0 {
		ui.Errf("gg discard: name a path or a handle. Use -a to drop all changes.")
		return 2
	}

	var known, fresh []string
	for _, p := range paths {
		if tracked(p) {
			known = append(known, p)
			continue
		}
		fresh = append(fresh, p)
	}
	if len(known) > 0 {
		argv := append([]string{"restore", "--source=HEAD", "--staged", "--worktree", "--"}, known...)
		if rc := gitx.Run(argv...); rc != 0 {
			return rc
		}
	}
	if len(fresh) > 0 {
		if rc := gitx.Run(append([]string{"clean", "-f", "-d", "--"}, fresh...)...); rc != 0 {
			return rc
		}
	}
	return 0
}
