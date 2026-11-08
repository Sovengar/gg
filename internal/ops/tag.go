package ops

import (
	"gg/internal/gitx"
)

// Tag lists tags newest version first, or hands the arguments to git tag so
// create, -d, -s and -a keep working. Only the listing is read-only; the
// caller records an oplog entry for the rest, and refs/tags is in the
// snapshot, so gg undo brings a deleted tag back.
func Tag(args []string) int {
	if len(args) == 0 {
		return gitx.Run("tag", "--sort=-v:refname", "-n", "-l")
	}
	return gitx.Run(append([]string{"tag"}, args...)...)
}

// IsTagList reports whether the arguments only list tags.
func IsTagList(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, a := range args {
		switch a {
		case "-l", "--list", "-n", "--verify", "-v":
		default:
			return false
		}
	}
	return true
}
