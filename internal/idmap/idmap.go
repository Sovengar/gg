// Package idmap keeps one changeId namespace for commits, branches and dirty
// files, the way but does. A token therefore names one thing: no verb has to
// guess which kind a changeId belongs to.
package idmap

import (
	"strings"

	"gg/internal/files"
	"gg/internal/ids"
	"gg/internal/ui"
)

// MinDisplay is how many id characters a line draws, the way but does: at least
// three, the unique prefix painted and the padding grey.
const MinDisplay = 3

// Map hands out the short id of each entity and resolves a token back.
type Map struct {
	commits  map[string]string // sha -> short id
	branches map[string]string // branch -> short id
	files    map[string]string // path -> short id

	commitBy map[string]string // token -> sha
	branchBy map[string]string // token -> branch
	fileBy   map[string]string // token -> path

	raws map[string]string // kind:key -> full id
}

// reserved tokens are never minted: zz names the uncommitted area.
var reserved = map[string]bool{"zz": true}

// Load reads commits, branches and dirty paths, then mints one namespace over
// all of them.
func Load() *Map {
	commits := map[string]string{}
	if ix, err := ids.Load(); err == nil {
		for sha, raw := range ix.Raws() {
			commits[sha] = raw
		}
	}
	branches := map[string]string{}
	if ix, err := ids.LoadBranches(); err == nil {
		for name, raw := range ix.Raws() {
			branches[name] = raw
		}
	}
	paths := files.Load().Raws()

	raws := make(map[string]string, len(commits)+len(branches)+len(paths))
	for sha, raw := range commits {
		raws["commit:"+sha] = raw
	}
	for name, raw := range branches {
		raws["branch:"+name] = raw
	}
	for path, raw := range paths {
		raws["file:"+path] = raw
	}

	short := ids.AssignShort(raws, reserved)

	m := &Map{
		commits:  make(map[string]string, len(commits)),
		branches: make(map[string]string, len(branches)),
		files:    make(map[string]string, len(paths)),
		commitBy: make(map[string]string, len(commits)),
		branchBy: make(map[string]string, len(branches)),
		fileBy:   make(map[string]string, len(paths)),
		raws:     raws,
	}
	for key, token := range short {
		kind, name, _ := strings.Cut(key, ":")
		switch kind {
		case "commit":
			m.commits[name] = token
			m.commitBy[token] = name
		case "branch":
			m.branches[name] = token
			m.branchBy[token] = name
		case "file":
			m.files[name] = token
			m.fileBy[token] = name
		}
	}
	return m
}

var shared *Map

// Shared is the map one process uses everywhere, so the tree, the log and the
// diff print the same token for the same thing.
func Shared() *Map {
	if shared == nil {
		shared = Load()
	}
	return shared
}

// draw returns the characters a line shows: the unique prefix, padded to
// MinDisplay so the column keeps a steady width.
func draw(short, full string) string {
	end := len(short)
	if end < MinDisplay {
		end = MinDisplay
	}
	if end > len(full) {
		end = len(full)
	}
	return full[:end]
}

// paint draws those characters: the unique prefix red, the padding grey.
func paint(short, full string) string {
	shown := draw(short, full)
	if len(shown) <= len(short) {
		return ui.Red(shown)
	}
	return ui.Red(short) + ui.Dim(shown[len(short):])
}

// resolve finds the single entry whose full id starts with token. A reserved
// token never resolves, and a prefix shared by two entries is ambiguous.
func resolve(full map[string]string, token string) (string, bool) {
	if reserved[token] || token == "" {
		return "", false
	}
	found := ""
	hits := 0
	for key, raw := range full {
		if !strings.HasPrefix(raw, token) {
			continue
		}
		found = key
		hits++
		if hits > 1 {
			return "", false
		}
	}
	return found, hits == 1
}

// plainCommit is the id a commit line draws, without colour.
func (m *Map) plainCommit(sha string) (string, bool) {
	short, ok := m.commits[sha]
	if !ok {
		return "", false
	}
	return draw(short, m.raws["commit:"+sha]), true
}

// CommitDisplay is the painted id for a commit: the unique prefix red, the
// padding grey.
func (m *Map) CommitDisplay(sha string) string {
	plain, ok := m.plainCommit(sha)
	if !ok {
		return ui.Red(m.Commit(sha))
	}
	return paint(m.commits[sha], plain)
}

// CommitPlain is the drawn id without colour, for width maths.
func (m *Map) CommitPlain(sha string) string {
	if plain, ok := m.plainCommit(sha); ok {
		return plain
	}
	return m.Commit(sha)
}

// BranchDisplay is the painted id for a branch.
func (m *Map) BranchDisplay(name string) string {
	short, ok := m.branches[name]
	if !ok {
		return ui.Red(m.Branch(name))
	}
	return paint(short, m.raws["branch:"+name])
}

// BranchPlain is the drawn id without colour.
func (m *Map) BranchPlain(name string) string {
	if short, ok := m.branches[name]; ok {
		return draw(short, m.raws["branch:"+name])
	}
	return m.Branch(name)
}

// FileDisplay is the painted id for a dirty path.
func (m *Map) FileDisplay(path string) string {
	short, ok := m.files[path]
	if !ok {
		return ""
	}
	return paint(short, m.raws["file:"+path])
}

// FilePlain is the drawn id without colour.
func (m *Map) FilePlain(path string) string {
	short, ok := m.files[path]
	if !ok {
		return ""
	}
	return draw(short, m.raws["file:"+path])
}

// Commit returns the short id of a commit, minting one when the sha is not in
// the map.
func (m *Map) Commit(sha string) string {
	if token, ok := m.commits[sha]; ok {
		return token
	}
	return ids.RawChangeID("sha:" + strings.ToLower(sha))[:ids.MintLen()]
}

// Branch returns the short id of a branch, or a name-derived one.
func (m *Map) Branch(name string) string {
	if token, ok := m.branches[name]; ok {
		return token
	}
	return ids.RawChangeID("branch:" + name)[:ids.MintLen()]
}

// File returns the short id of a dirty path, or "" when it is clean.
func (m *Map) File(path string) string { return m.files[path] }

// CommitBy resolves a commit token.
func (m *Map) CommitBy(token string) (string, bool) {
	if sha, ok := m.commitBy[token]; ok {
		return sha, true
	}
	return resolve(m.rawsBy("commit:"), token)
}

// rawsBy selects the full ids of one kind.
func (m *Map) rawsBy(prefix string) map[string]string {
	out := make(map[string]string, len(m.raws))
	for key, raw := range m.raws {
		if strings.HasPrefix(key, prefix) {
			out[strings.TrimPrefix(key, prefix)] = raw
		}
	}
	return out
}

// BranchBy resolves a branch token.
func (m *Map) BranchBy(token string) (string, bool) {
	if name, ok := m.branchBy[token]; ok {
		return name, true
	}
	return resolve(m.rawsBy("branch:"), token)
}

// FileBy resolves a file token.
func (m *Map) FileBy(token string) (string, bool) {
	if path, ok := m.fileBy[token]; ok {
		return path, true
	}
	return resolve(m.rawsBy("file:"), token)
}
