package ops

import (
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Merge merges a branch into the current one. gg asks git for a fast-forward
// unless the call carries its own fast-forward flag or the repository sets
// merge.ff: a merge commit is a decision, never a default.
func Merge(args []string) int {
	argv, ffOnly := mergeArgs(args)
	branch := mergeBranch(args)
	if branch != "<branch>" && !revExists(branch) {
		ui.Errf("gg merge: no branch or commit named %s", branch)
		return 2
	}
	rc := gitx.Run(append([]string{"merge"}, argv...)...)
	if rc == 0 {
		return 0
	}
	if ffOnly {
		ui.Errf("gg merge: not a fast-forward, so nothing merged.")
		ui.Errf("gg merge: %s --no-ff keeps a merge commit, gg rebase %s replays it, gg land %s pushes it.",
			branch, branch, branch)
		return 1
	}
	return rc
}

// mergeArgs adds --ff-only when the call says nothing about fast-forwarding,
// and leaves the arguments alone when they do or when merge.ff is configured:
// the repository then owns the policy.
func mergeArgs(args []string) (argv []string, ffOnly bool) {
	if len(args) == 0 || hasMergeFlag(args) {
		return args, false
	}
	if _, err := gitx.Output("config", "--get", "merge.ff"); err == nil {
		return args, false
	}
	return append([]string{"--ff-only"}, args...), true
}

// mergeFlags are the arguments that already decide how the merge behaves, so
// gg must not add a fast-forward flag of its own.
var mergeFlags = map[string]bool{
	"--ff": true, "--no-ff": true, "--ff-only": true, "--squash": true,
	"--abort": true, "--continue": true, "--quit": true,
	"-m": true, "--message": true, "-F": true, "--file": true,
	"-e": true, "--edit": true, "--no-edit": true,
	"-s": true, "--strategy": true, "-X": true, "--strategy-option": true,
	"--no-commit": true, "--commit": true, "--no-verify": true,
	"--allow-unrelated-histories": true, "--log": true, "--no-log": true,
}

// hasMergeFlag reports whether the call carries a flag of its own.
func hasMergeFlag(args []string) bool {
	for _, a := range args {
		if mergeFlags[a] || strings.HasPrefix(a, "--ff=") || strings.HasPrefix(a, "--strategy=") {
			return true
		}
	}
	return false
}

// mergeBranch names the branch the call mentions, for the refusal message.
func mergeBranch(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return "<branch>"
}
