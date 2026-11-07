// Package ids gives commits a short handle that survives history rewrites.
//
// The handle comes from the commit's Change-Id trailer when present, so an
// amend, a rebase or a squash keeps the same handle. Commits without that
// trailer fall back to a handle derived from their sha, which does change when
// the commit is rewritten.
//
// Handles are the shortest unique prefix, never shorter than two characters.
package ids

import (
	"crypto/sha256"
	"sort"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// alphabet avoids characters that are easy to confuse when typed.
const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"

// MinLen and MaxLen bound a handle.
const (
	MinLen = 2
	MaxLen = 4
)

// Index maps handles to commits for the current repository.
type Index struct {
	handles map[string]string   // sha -> handle
	byToken map[string]string   // token -> sha, only unique prefixes
	ambig   map[string][]string // token -> shas, when not unique
}

// Load scans the history once and builds the index.
func Load() (*Index, error) {
	// only refs a user can reach: refs/gg/oplog and refs/stash would repeat
	// rewritten commits and split one Change-Id across two shas
	lines, err := gitx.OutputAll("log", "--branches", "--tags", "--remotes", "HEAD",
		"--format=%H%x1f%(trailers:key=Change-Id,valueonly)", "-n", "2000")
	if err != nil {
		return nil, err
	}

	raws := make(map[string]string, len(lines)) // sha -> 4 char raw
	seen := make(map[string]bool, len(lines))   // one sha per identity key
	for _, line := range lines {
		sha, trailer, _ := strings.Cut(line, "\x1f")
		sha = strings.TrimSpace(sha)
		if sha == "" {
			continue
		}
		key := strings.TrimSpace(trailer)
		if key == "" {
			key = "sha:" + strings.ToLower(sha)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		raws[sha] = Raw(key)
	}

	// count how many commits share each prefix, so handles stay unique
	owners := make(map[string]map[string]bool)
	for sha, raw := range raws {
		for n := MinLen; n <= MaxLen; n++ {
			p := raw[:n]
			if owners[p] == nil {
				owners[p] = make(map[string]bool)
			}
			owners[p][sha] = true
		}
	}

	ix := &Index{
		handles: make(map[string]string, len(raws)),
		byToken: make(map[string]string, len(owners)),
		ambig:   make(map[string][]string),
	}
	for sha, raw := range raws {
		handle := raw
		for n := MinLen; n <= MaxLen; n++ {
			if len(owners[raw[:n]]) == 1 {
				handle = raw[:n]
				break
			}
		}
		ix.handles[sha] = handle
	}
	for token, shas := range owners {
		if len(shas) == 1 {
			for sha := range shas {
				ix.byToken[token] = sha
			}
			continue
		}
		for sha := range shas {
			ix.ambig[token] = append(ix.ambig[token], sha)
		}
	}
	return ix, nil
}

// Raw hashes an identity key into alphabet characters.
func Raw(key string) string {
	sum := sha256.Sum256([]byte(key))
	var b strings.Builder
	for i := 0; i < MaxLen; i++ {
		b.WriteByte(alphabet[int(sum[i])%len(alphabet)])
	}
	return b.String()
}

// Handle returns the handle for a sha, or a sha-derived one when the commit is
// not in the index.
func (ix *Index) Handle(sha string) string {
	if h, ok := ix.handles[sha]; ok {
		return h
	}
	return Raw("sha:" + strings.ToLower(sha))
}

// Short returns the handle and the abbreviated sha, for display.
func (ix *Index) Short(sha string) string {
	if len(sha) < 7 {
		return ix.Handle(sha) + " " + sha
	}
	return ix.Handle(sha) + " " + sha[:7]
}

// Ref is the result of resolving a token.
type Ref struct {
	SHA    string
	Branch string // set when the sha is the tip of a local branch
}

// Status is the outcome of a resolution attempt.
type Status int

const (
	Unknown Status = iota
	Resolved
	Ambiguous
)

// Resolve turns a two to four character token, or @token, into a commit.
func (ix *Index) Resolve(token string) (Ref, Status) {
	token = strings.TrimPrefix(token, "@")
	if !Shaped(token) {
		return Ref{}, Unknown
	}
	if sha, ok := ix.byToken[token]; ok {
		return Ref{SHA: sha, Branch: branchTip(sha)}, Resolved
	}
	if shas, ok := ix.ambig[token]; ok {
		sort.Strings(shas)
		var handles []string
		for _, sha := range shas {
			handles = append(handles, ix.Handle(sha))
		}
		ui.Errf("gg: %s is ambiguous: %s", token, strings.Join(handles, " "))
		return Ref{}, Ambiguous
	}
	return Ref{}, Unknown
}

// Shaped reports whether a token could be a handle.
func Shaped(token string) bool {
	if len(token) < MinLen || len(token) > MaxLen {
		return false
	}
	for _, r := range token {
		if !strings.ContainsRune(alphabet, r) {
			return false
		}
	}
	return true
}

func branchTip(sha string) string {
	lines, err := gitx.OutputAll("for-each-ref", "--format=%(objectname) %(refname:short)", "refs/heads")
	if err != nil {
		return ""
	}
	for _, line := range lines {
		obj, name, ok := strings.Cut(line, " ")
		if ok && obj == sha {
			return name
		}
	}
	return ""
}
