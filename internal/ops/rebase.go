package ops

import (
	"gg/internal/gitx"
)

// Rebase replays the current branch onto another base. It passes every flag
// through, so -i, --onto and --autosquash keep working; the wip commit is
// folded first by the caller, because a rebase rewrites HEAD.
func Rebase(args []string) int {
	return gitx.Run(append([]string{"rebase"}, args...)...)
}
