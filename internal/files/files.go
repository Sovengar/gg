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

// Index maps changeIDs to paths for the current working tree.
type Index struct {
	changeIDs map[string]string   // path -> changeId
	raws      map[string]string   // path -> full reverse-hex identity
	byToken   map[string]string   // unique token -> path
	ambig     map[string][]string // token shared by several paths
}

// Load reads the dirty paths once.
func Load() *Index {
	ix := &Index{changeIDs: map[string]string{}, raws: map[string]string{}, byToken: map[string]string{}, ambig: map[string][]string{}}

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
		raw := fileRaw(p)
		ix.raws[p] = raw
		ix.changeIDs[p] = raw
		for n := ids.MintLen(); n <= ids.MaxLen && n <= len(raw); n++ {
			owners[raw[:n]] = append(owners[raw[:n]], p)
		}
	}
	for token, ps := range owners {
		if len(ps) != 1 {
			sorted := append([]string(nil), ps...)
			sort.Strings(sorted)
			ix.ambig[token] = sorted
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

// fileRaw is the reverse-hex identity of a path. A path made only of k..z is
// spelled in front, the way but keeps a name-derived id readable.
func fileRaw(path string) string {
	raw := ids.RawChangeID("file:" + path)
	for _, r := range path {
		if r < 'k' || r > 'z' {
			return raw
		}
	}
	return path + raw
}

// Raws is the full identity of every dirty path in the index.
func (ix *Index) Raws() map[string]string { return ix.raws }

// ChangeID returns the changeId for a path, or "" when the path is clean.
func (ix *Index) ChangeID(path string) string { return ix.changeIDs[path] }

// Resolve turns a token into a dirty path.
func (ix *Index) Resolve(token string) (string, bool) {
	p, ok := ix.byToken[token]
	return p, ok
}

// Ambiguous lists the paths a token could name, when there is more than one.
func (ix *Index) Ambiguous(token string) []string {
	return ix.ambig[token]
}

// Display is the changeId plus the path.
func (ix *Index) Display(path string) string {
	if h := ix.ChangeID(path); h != "" {
		return h + " " + path
	}
	return path
}
