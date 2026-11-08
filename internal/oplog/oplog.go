// Package oplog records the state before every mutating verb so that undo and
// redo can move the worktree through the operations.
package oplog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"gg/internal/gitx"
	"gg/internal/ui"
)

const (
	oplogRef = "refs/gg/oplog/tip"
	curRef   = "refs/gg/oplog/cur"
	redoRef  = "refs/gg/oplog/redo"
	keep     = 100

	// Entries are marked with a committer identity so the two parents of an
	// entry can be told apart from the ordinary commit in P2.
	markerName  = "gg oplog"
	markerEmail = "gg-oplog@localhost"
)

// Record snapshots HEAD and the full worktree as the state before verb runs.
func Record(verb string) {
	if !gitx.InRepo() {
		return
	}
	tree, err := snapshotTree()
	if err != nil {
		ui.Errf("gg: oplog: %v", err)
		return
	}
	message, err := currentMessage(verb)
	if err != nil {
		ui.Errf("gg: oplog: %v", err)
		return
	}
	cur, _ := getRef(curRef)
	head, _ := gitx.Head()
	sha, err := commitEntry(tree, cur, head, message, "")
	if err != nil {
		ui.Errf("gg: oplog: %v", err)
		return
	}
	setRef(oplogRef, sha)
	setRef(curRef, sha)
	clearRef(redoRef)
	trim()
}

// List prints the entries reachable by undo, newest first.
func List() int {
	cur, ok := getRef(curRef)
	if !ok {
		fmt.Println(ui.Dim("no operations yet"))
		return 0
	}
	for _, sha := range chain(cur) {
		short, err := gitx.Output("rev-parse", "--short", sha)
		if err != nil || short == "" {
			short = sha
		}
		verb, _ := gitx.Output("log", "-1", "--format=%s", sha)
		epoch, _ := gitx.Output("log", "-1", "--format=%ct", sha)
		fmt.Printf("  %s  %s  %s\n", ui.Cyan(short), ui.Dim(age(epoch)), verb)
	}
	return 0
}

// Clear drops the oplog refs. The objects stay in the repository until git gc.
func Clear() int {
	clearRef(oplogRef)
	clearRef(curRef)
	clearRef(redoRef)
	fmt.Println(ui.Dim("gg: oplog cleared; run git gc --prune=now to drop the objects"))
	return 0
}

// Undo restores the state before the last n operations. A leading force
// argument bypasses the published-commit guard.
func Undo(n int, force ...bool) int {
	forced := len(force) > 0 && force[0]
	if n < 1 {
		n = 1
	}
	cur, ok := getRef(curRef)
	if !ok {
		ui.Errf("gg: nothing to undo")
		return 1
	}

	tree, err := snapshotTree()
	if err != nil {
		ui.Errf("gg: oplog: %v", err)
		return 1
	}
	head, hasHead := gitx.Head()
	curTree, _ := entryTree(cur)
	curHead, hasCurHead := p2(cur)
	same := tree == curTree && hasHead == hasCurHead && (!hasHead || head == curHead) && refsMatch(cur)

	steps := len(chain(cur))
	if same {
		steps--
	}
	if n > steps {
		ui.Errf("gg: cannot undo %d", n)
		return 1
	}

	if !same {
		message, err := currentMessage("undo")
		if err != nil {
			ui.Errf("gg: oplog: %v", err)
			return 1
		}
		sha, err := commitEntry(tree, cur, head, message, "")
		if err != nil {
			ui.Errf("gg: oplog: %v", err)
			return 1
		}
		setRef(oplogRef, sha)
		setRef(curRef, sha)
		cur = sha
	}

	target := cur
	for i := 0; i < n; i++ {
		prev, ok := p1(target)
		if !ok {
			ui.Errf("gg: cannot undo %d", n)
			return 1
		}
		target = prev
	}
	if !forced {
		if targetHead, ok := p2(target); ok {
			if upstream, ok := gitx.Upstream(); ok {
				if !isAncestor(upstream, targetHead) {
					branch, _ := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
					ui.Errf("gg: undo would remove commits published on %s; pass --force to undo anyway", branch)
					return 1
				}
			}
		}
	}
	if rc := restore(target, cur); rc != 0 {
		return rc
	}
	setRef(curRef, target)
	if tip, ok := getRef(oplogRef); ok {
		setRef(redoRef, tip)
	} else {
		clearRef(redoRef)
	}
	return 0
}

// Redo restores the state an undo left behind. It records the current state
// first, so a following undo returns edits made after the undo.
func Redo() int {
	target, ok := getRef(redoRef)
	if !ok {
		ui.Errf("gg: nothing to redo")
		return 1
	}

	tree, err := snapshotTree()
	if err != nil {
		ui.Errf("gg: oplog: %v", err)
		return 1
	}
	head, hasHead := gitx.Head()
	targetTree, _ := entryTree(target)
	targetHead, hasTargetHead := p2(target)
	same := tree == targetTree && hasHead == hasTargetHead && (!hasHead || head == targetHead) && refsMatch(target)

	restoreSha := target
	from := ""
	if !same {
		// Insert the current state before the target, then rebuild the target
		// state on top of it. The next undo lands on the recorded edit.
		prev, _ := getRef(curRef)
		message, err := currentMessage("redo")
		if err != nil {
			ui.Errf("gg: oplog: %v", err)
			return 1
		}
		edit, err := commitEntry(tree, prev, head, message, "")
		if err != nil {
			ui.Errf("gg: oplog: %v", err)
			return 1
		}
		message, _ = gitx.Output("log", "-1", "--format=%B", target)
		date, _ := gitx.Output("log", "-1", "--format=%ct", target)
		restoreSha, err = commitEntry(targetTree, edit, targetHead, message, date)
		if err != nil {
			ui.Errf("gg: oplog: %v", err)
			return 1
		}
		from = edit
		setRef(oplogRef, restoreSha)
	}

	if rc := restore(restoreSha, from); rc != 0 {
		return rc
	}
	setRef(curRef, restoreSha)
	clearRef(redoRef)
	return 0
}

// restore resets HEAD and the worktree to an entry. from is the state being
// left behind: it is loaded into the index first so files that are absent from
// the target are removed even when they are currently untracked.
func restore(sha, from string) int {
	if from != "" {
		if rc := gitx.Run("read-tree", "-u", "--reset", from); rc != 0 {
			return rc
		}
	}
	if head, ok := p2(sha); ok {
		if rc := gitx.Run("reset", "--hard", head); rc != 0 {
			return rc
		}
	} else {
		_, _ = gitx.Output("update-ref", "-d", "HEAD")
	}
	if rc := gitx.Run("read-tree", "-u", "--reset", sha); rc != 0 {
		return rc
	}
	if rc := applyRefs(sha); rc != 0 {
		return rc
	}
	if _, ok := gitx.Head(); ok {
		if rc := gitx.Run("reset", "--quiet"); rc != 0 {
			return rc
		}
	} else if rc := gitx.Run("read-tree", "--empty"); rc != 0 {
		return rc
	}
	setRef(curRef, sha)
	return 0
}

// currentMessage captures the refs and builds the entry message for verb.
func currentMessage(verb string) (string, error) {
	blob, err := snapshotRefs()
	if err != nil {
		return "", err
	}
	return verb + "\n\ngg-refs: " + blob, nil
}

// snapshotRefText renders the managed refs and HEAD's symbolic target.
func snapshotRefText() (string, error) {
	lines, err := refLines()
	if err != nil {
		return "", err
	}
	if head, err := gitx.Output("symbolic-ref", "--quiet", "HEAD"); err == nil && head != "" {
		lines = append(lines, "@head "+head)
	} else if head, ok := gitx.Head(); ok {
		lines = append(lines, "@detached "+head)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

// snapshotRefs writes the managed refs and HEAD's symbolic target to a blob.
func snapshotRefs() (string, error) {
	text, err := snapshotRefText()
	if err != nil {
		return "", err
	}
	return hashObject(text)
}

// refsMatch reports whether the current refs equal the entry's snapshot. Old
// entries without a trailer are treated as matching.
func refsMatch(entry string) bool {
	blob, ok := refsTrailer(entry)
	if !ok {
		return true
	}
	want, err := gitx.Output("cat-file", "blob", blob)
	if err != nil {
		return false
	}
	have, err := snapshotRefText()
	if err != nil {
		return false
	}
	return want == have
}

// refLines lists the managed refs, excluding the oplog's own refs.
func refLines() ([]string, error) {
	lines, err := gitx.OutputAll("for-each-ref",
		"--format=%(refname) %(objectname)",
		"refs/heads", "refs/tags", "refs/gg")
	if err != nil {
		return nil, err
	}
	out := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(line, "refs/gg/oplog/") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

func hashObject(text string) (string, error) {
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// refsTrailer returns the refs blob sha stored in an entry, when present.
func refsTrailer(sha string) (string, bool) {
	message, err := gitx.Output("log", "-1", "--format=%B", sha)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(message, "\n") {
		if blob, ok := strings.CutPrefix(strings.TrimSpace(line), "gg-refs: "); ok {
			return strings.TrimSpace(blob), true
		}
	}
	return "", false
}

// applyRefs restores the refs captured by an entry. A missing trailer is an
// old entry, so refs are left alone.
func applyRefs(entry string) int {
	blob, ok := refsTrailer(entry)
	if !ok {
		return 0
	}
	text, err := gitx.Output("cat-file", "blob", blob)
	if err != nil {
		ui.Errf("gg: oplog: refs snapshot: %v", err)
		return 1
	}
	wanted := make(map[string]string)
	head, detached := "", ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "@head "):
			head = strings.TrimSpace(strings.TrimPrefix(line, "@head "))
			continue
		case strings.HasPrefix(line, "@detached "):
			detached = strings.TrimSpace(strings.TrimPrefix(line, "@detached "))
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !managedRef(fields[0]) {
			continue
		}
		wanted[fields[0]] = fields[1]
	}

	current := make(map[string]string)
	lines, err := refLines()
	if err != nil {
		ui.Errf("gg: oplog: refs snapshot: %v", err)
		return 1
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && managedRef(fields[0]) {
			current[fields[0]] = fields[1]
		}
	}
	for ref, sha := range wanted {
		if cur, ok := current[ref]; !ok || cur != sha {
			if _, err := gitx.Output("update-ref", ref, sha); err != nil {
				ui.Errf("gg: oplog: restore %s: %v", ref, err)
				return 1
			}
		}
	}
	for ref := range current {
		if _, ok := wanted[ref]; ok {
			continue
		}
		if _, err := gitx.Output("update-ref", "-d", ref); err != nil {
			ui.Errf("gg: oplog: delete %s: %v", ref, err)
			return 1
		}
	}
	if detached != "" {
		if _, err := gitx.Output("update-ref", "--no-deref", "HEAD", detached); err != nil {
			ui.Errf("gg: oplog: restore HEAD: %v", err)
			return 1
		}
	} else if head != "" {
		if _, err := gitx.Output("symbolic-ref", "HEAD", head); err != nil {
			ui.Errf("gg: oplog: restore HEAD: %v", err)
			return 1
		}
	}
	return 0
}

// managedRef reports whether undo owns the ref namespace.
func managedRef(ref string) bool {
	if strings.HasPrefix(ref, "refs/gg/oplog/") {
		return false
	}
	return strings.HasPrefix(ref, "refs/heads/") ||
		strings.HasPrefix(ref, "refs/tags/") ||
		strings.HasPrefix(ref, "refs/gg/")
}

// snapshotTree writes the tracked and untracked worktree to a tree object
// without touching the real index.
func snapshotTree() (string, error) {
	gitDir, err := gitx.Output("rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(gitDir) {
		if gitDir, err = filepath.Abs(gitDir); err != nil {
			return "", err
		}
	}
	index := filepath.Join(gitDir, fmt.Sprintf("gg-tmp-index-%d-%d", os.Getpid(), tempSeq.Add(1)))
	_ = os.Remove(index)
	defer os.Remove(index)

	env := []string{"GIT_INDEX_FILE=" + index}
	base, ok := gitx.Head()
	if !ok {
		if base, err = emptyTree(); err != nil {
			return "", err
		}
	}
	if _, err := outputEnv(env, "read-tree", base); err != nil {
		return "", err
	}
	if _, err := outputEnv(env, "add", "-A"); err != nil {
		return "", err
	}
	return outputEnv(env, "write-tree")
}

var tempSeq atomic.Uint64

func emptyTree() (string, error) {
	cmd := exec.Command("git", "mktree")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func commitEntry(tree, prev, head, message, date string) (string, error) {
	args := []string{"commit-tree", tree}
	if prev != "" {
		args = append(args, "-p", prev)
	}
	if head != "" && head != prev {
		args = append(args, "-p", head)
	}
	args = append(args, "-m", message)
	env := []string{
		"GIT_COMMITTER_NAME=" + markerName,
		"GIT_COMMITTER_EMAIL=" + markerEmail,
	}
	if date != "" {
		env = append(env, "GIT_COMMITTER_DATE=@"+date+" +0000")
	}
	return outputEnv(env, args...)
}

func outputEnv(env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getRef(ref string) (string, bool) {
	sha, err := gitx.Output("rev-parse", "--verify", "--quiet", ref)
	if err != nil || sha == "" {
		return "", false
	}
	return sha, true
}

func setRef(ref, sha string) error {
	_, err := gitx.Output("update-ref", ref, sha)
	return err
}

func clearRef(ref string) {
	_, _ = gitx.Output("update-ref", "-d", ref)
}

func parents(sha string) []string {
	out, err := gitx.Output("rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return nil
	}
	fields := strings.Fields(out)
	if len(fields) < 2 {
		return nil
	}
	return fields[1:]
}

// isEntry reports whether sha is an oplog entry rather than an ordinary commit.
func isEntry(sha string) bool {
	if sha == "" {
		return false
	}
	email, err := gitx.Output("log", "-1", "--format=%ce", sha)
	return err == nil && strings.TrimSpace(email) == markerEmail
}

// p1 returns the previous entry in the chain, if this entry has one.
func p1(sha string) (string, bool) {
	ps := parents(sha)
	if len(ps) == 0 || !isEntry(ps[0]) {
		return "", false
	}
	return ps[0], true
}

// p2 returns HEAD at snapshot time, if the entry has one.
func p2(sha string) (string, bool) {
	ps := parents(sha)
	if len(ps) >= 2 {
		return ps[1], true
	}
	if len(ps) == 1 && !isEntry(ps[0]) {
		return ps[0], true
	}
	return "", false
}

func entryTree(sha string) (string, error) {
	tree, err := gitx.Output("rev-parse", sha+"^{tree}")
	if err != nil {
		return "", err
	}
	return tree, nil
}

func chain(cur string) []string {
	var out []string
	for cur != "" {
		out = append(out, cur)
		prev, ok := p1(cur)
		if !ok {
			break
		}
		cur = prev
	}
	return out
}

// trim rebuilds the newest keep entries when the chain has grown past it.
func trim() {
	cur, ok := getRef(curRef)
	if !ok {
		return
	}
	entries := chain(cur)
	if len(entries) <= keep {
		return
	}
	entries = entries[:keep]

	prev := ""
	newest := ""
	for i := len(entries) - 1; i >= 0; i-- {
		old := entries[i]
		tree, err := entryTree(old)
		if err != nil {
			return
		}
		head, _ := p2(old)
		message, _ := gitx.Output("log", "-1", "--format=%B", old)
		date, _ := gitx.Output("log", "-1", "--format=%ct", old)
		sha, err := commitEntry(tree, prev, head, message, date)
		if err != nil {
			return
		}
		prev = sha
		if i == 0 {
			newest = sha
		}
	}
	setRef(oplogRef, newest)
	setRef(curRef, newest)
	if redo, ok := getRef(redoRef); ok && !isAncestor(redo, newest) {
		clearRef(redoRef)
	}
}

// isAncestor reports whether a is an ancestor of b.
func isAncestor(a, b string) bool {
	_, err := gitx.Output("merge-base", "--is-ancestor", a, b)
	return err == nil
}

// age renders a compact relative time: 5m, 2h, 10d, 3w.
func age(epoch string) string {
	secs, err := strconv.ParseInt(strings.TrimSpace(epoch), 10, 64)
	if err != nil {
		return ""
	}
	d := time.Since(time.Unix(secs, 0))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/24/7))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	}
}
