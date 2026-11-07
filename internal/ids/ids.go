// Package ids gives commits a short changeId that survives history rewrites.
//
// The changeId comes from the commit's Change-Id trailer when present, so an
// amend, a rebase or a squash keeps the same changeId. Commits without that
// trailer fall back to a changeId derived from their sha, which does change when
// the commit is rewritten.
//
// changeIds are the shortest unique prefix, never shorter than two characters.
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

// MinLen and MaxLen bound a changeId.
const (
	MinLen = 2
	MaxLen = 4
)

// Index maps changeIDs to commits for the current repository.
type Index struct {
	changeIDs map[string]string   // sha -> changeId
	byToken   map[string]string   // token -> sha, only unique prefixes
	ambig     map[string][]string // token -> shas, when not unique
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
		raws[sha] = RawChangeID(key)
	}

	// count how many commits share each prefix, so changeIDs stay unique
	owners := prefixOwners(raws)

	ix := &Index{
		changeIDs: make(map[string]string, len(raws)),
		byToken:   make(map[string]string, len(owners)),
		ambig:     make(map[string][]string),
	}
	for sha, raw := range raws {
		ix.changeIDs[sha] = shortestChangeID(owners, raw)
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
func RawChangeID(key string) string {
	sum := sha256.Sum256([]byte(key))
	var b strings.Builder
	for i := 0; i < MaxLen; i++ {
		b.WriteByte(alphabet[int(sum[i])%len(alphabet)])
	}
	return b.String()
}

// ChangeID returns the changeId for a sha, or a sha-derived one when the commit is
// not in the index.
func (ix *Index) ChangeID(sha string) string {
	if h, ok := ix.changeIDs[sha]; ok {
		return h
	}
	return RawChangeID("sha:" + strings.ToLower(sha))
}

// Short returns the changeId and the abbreviated sha, for display.
func (ix *Index) Short(sha string) string {
	if len(sha) < 7 {
		return ix.ChangeID(sha) + " " + sha
	}
	return ix.ChangeID(sha) + " " + sha[:7]
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
		var changeIDs []string
		for _, sha := range shas {
			changeIDs = append(changeIDs, ix.ChangeID(sha))
		}
		ui.Errf("gg: %s is ambiguous: %s", token, strings.Join(changeIDs, " "))
		return Ref{}, Ambiguous
	}
	return Ref{}, Unknown
}

// Shaped reports whether a token could be a changeId.
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
