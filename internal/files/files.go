// Package files gives every dirty path a short changeId, so verbs can take one
// instead of a long path.
package files

import (
	"sort"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ids"
)

// MinLen and MaxLen bound a file changeId.
const (
	MinLen = 2
	MaxLen = 4
)

// Index maps changeIDs to paths for the current working tree.
type Index struct {
	changeIDs map[string]string // path -> changeId
	byToken   map[string]string // unique token -> path
}

// Load reads the dirty paths once.
func Load() *Index {
	ix := &Index{changeIDs: map[string]string{}, byToken: map[string]string{}}

	lines, err := gitx.OutputAll("status", "--short", "--untracked-files=all")
	if err != nil {
		return ix
	}

	var paths []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") || len(line) < 4 {
			continue
		}
		paths = append(paths, strings.TrimSpace(line[3:]))
	}
	sort.Strings(paths)

	owners := map[string][]string{}
	for _, p := range paths {
		raw := ids.RawChangeID("file:" + p)
		ix.changeIDs[p] = raw
		for n := MinLen; n <= MaxLen; n++ {
			owners[raw[:n]] = append(owners[raw[:n]], p)
		}
	}
	for token, ps := range owners {
		if len(ps) != 1 {
			continue
		}
		ix.byToken[token] = ps[0]
		path := ps[0]
		if cur, ok := ix.changeIDs[path]; !ok || len(token) < len(cur) {
			ix.changeIDs[path] = token
		}
	}
	return ix
}

// ChangeID returns the changeId for a path, or "" when the path is clean.
func (ix *Index) ChangeID(path string) string { return ix.changeIDs[path] }

// Resolve turns a token into a dirty path.
func (ix *Index) Resolve(token string) (string, bool) {
	p, ok := ix.byToken[token]
	return p, ok
}

// Display is the changeId plus the path.
func (ix *Index) Display(path string) string {
	if h := ix.ChangeID(path); h != "" {
		return h + " " + path
	}
	return path
}
