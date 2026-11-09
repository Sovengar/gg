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
// With -e the names are the ones left out, the way `gg ce` runs it.
// The index stays an implementation detail: no verb asks the user to stage.
func Commit(args []string) int {
	flags, paths := splitCommitArgs(args)
	flags, exclude := takeExclude(flags)
	if exclude {
		if gitx.InWip() {
			ui.Errf("gg ce: HEAD is a wip commit. Run gg uncommit first, then gg ce.")
			return 1
		}
		var rc int
		if paths, rc = keepDirty(paths); rc != 0 {
			return rc
		}
	}
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

// takeExclude drops -e from the flags and reports whether it was there.
func takeExclude(flags []string) ([]string, bool) {
	kept := flags[:0]
	exclude := false
	for _, f := range flags {
		switch f {
		case "-e", "--exclude":
			exclude = true
		default:
			kept = append(kept, f)
		}
	}
	return kept, exclude
}

// keepDirty returns every dirty path that the excluded names do not cover.
// A name covers itself and, when it names a directory, what sits under it.
func keepDirty(excluded []string) ([]string, int) {
	all, err := dirtyPaths()
	if err != nil {
		ui.Errf("gg ce: cannot read the dirty paths")
		return nil, 1
	}
	covered := func(p string) bool {
		for _, ex := range excluded {
			if p == ex || strings.HasPrefix(p, strings.TrimSuffix(ex, "/")+"/") {
				return true
			}
		}
		return false
	}
	for _, ex := range excluded {
		found := false
		for _, p := range all {
			if p == ex || strings.HasPrefix(p, strings.TrimSuffix(ex, "/")+"/") {
				found = true
				break
			}
		}
		if !found {
			ui.Errf("gg ce: %s is not a dirty path", ex)
			return nil, 1
		}
	}
	var keep []string
	for _, p := range all {
		if !covered(p) {
			keep = append(keep, p)
		}
	}
	if len(keep) == 0 {
		ui.Errf("gg ce: that leaves no dirty path to commit")
		return nil, 1
	}
	return keep, 0
}

// dirtyPaths lists every changed path, new files included.
func dirtyPaths() ([]string, error) {
	lines, err := gitx.OutputAll("status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") || len(line) < 4 {
			continue
		}
		p := strings.TrimSpace(line[3:])
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		paths = append(paths, p)
	}
	return paths, nil
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
