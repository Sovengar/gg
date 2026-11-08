package ops

import (
	"gg/internal/gitx"
)

// Remote lists the configured remotes, or hands the arguments to git remote so
// add, rename, rm and set-url keep working. The remotes live in git config, not
// in a ref, so no oplog entry can bring a removed one back.
func Remote(args []string) int {
	if len(args) == 0 {
		return gitx.Run("remote", "-v")
	}
	return gitx.Run(append([]string{"remote"}, args...)...)
}
