package ids

import (
	"sort"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// BranchIndex gives every local branch its own changeId, derived from the branch
// name instead of the tip commit. Two branches on the same commit share a
// commit changeId, but never a branch changeId, so a listing always names the
// branch you mean.
type BranchIndex struct {
	changeIDs map[string]string   // branch -> changeId
	byToken   map[string]string   // token -> branch, only unique prefixes
	ambig     map[string][]string // token -> branches, when not unique
}

// LoadBranches reads the local branches once and gives each a changeId.
func LoadBranches() (*BranchIndex, error) {
	names, err := gitx.OutputAll("for-each-ref", "refs/heads", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	raws := make(map[string]string, len(names)) // branch -> raw changeId
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		raws[name] = RawChangeID("branch:" + name)
	}

	owners := prefixOwners(raws)

	ix := &BranchIndex{
		changeIDs: make(map[string]string, len(raws)),
		byToken:   make(map[string]string, len(owners)),
		ambig:     make(map[string][]string),
	}
	for name, raw := range raws {
		ix.changeIDs[name] = shortestChangeID(owners, raw)
	}
	for token, names := range owners {
		if len(names) == 1 {
			for name := range names {
				ix.byToken[token] = name
			}
			continue
		}
		var sorted []string
		for name := range names {
			sorted = append(sorted, name)
		}
		sort.Strings(sorted)
		ix.ambig[token] = sorted
	}
	return ix, nil
}

// ChangeID returns the branch changeId, or a name-derived one for a branch that is
// not in the index.
func (ix *BranchIndex) ChangeID(name string) string {
	if h, ok := ix.changeIDs[name]; ok {
		return h
	}
	return RawChangeID("branch:" + name)
}

// Ambiguous lists the branches a token could name, when there are several.
func (ix *BranchIndex) Ambiguous(token string) []string {
	return ix.ambig[token]
}

// Resolve turns a two to four character token into a branch name.
func (ix *BranchIndex) Resolve(token string) (string, Status) {
	if !Shaped(token) {
		return "", Unknown
	}
	if name, ok := ix.byToken[token]; ok {
		return name, Resolved
	}
	if names, ok := ix.ambig[token]; ok {
		ui.Errf("gg: %s names %d branches: %s", token, len(names), strings.Join(names, " "))
		return "", Ambiguous
	}
	return "", Unknown
}

// prefixOwners counts the branches that own each raw prefix, so changeIDs stay
// unique inside the set.
func prefixOwners(raws map[string]string) map[string]map[string]bool {
	owners := make(map[string]map[string]bool)
	for name, raw := range raws {
		for n := MintLen(); n <= MaxLen; n++ {
			p := raw[:n]
			if owners[p] == nil {
				owners[p] = make(map[string]bool)
			}
			owners[p][name] = true
		}
	}
	return owners
}

// shortestChangeID is the shortest prefix of raw that names one owner only.
func shortestChangeID(owners map[string]map[string]bool, raw string) string {
	for n := MintLen(); n <= MaxLen; n++ {
		if len(owners[raw[:n]]) == 1 {
			return raw[:n]
		}
	}
	return raw
}
