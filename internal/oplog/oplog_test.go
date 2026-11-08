package oplog

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main", ".")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	return dir
}

func inDir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func list(t *testing.T) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := List()
	_ = w.Close()
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("List() = %d", code)
	}
	return string(out)
}

func TestRecordListsEntriesNewestFirst(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	write(t, "f.txt", "one\n")
	Record("commit")
	write(t, "f.txt", "two\n")
	Record("discard")

	tip, ok := getRef(oplogRef)
	if !ok {
		t.Fatal("tip ref was not set")
	}
	if cur, ok := getRef(curRef); !ok || cur != tip {
		t.Fatalf("cur = %q, %v; want tip %q", cur, ok, tip)
	}

	lines := strings.Split(strings.TrimSpace(list(t)), "\n")
	if len(lines) != 2 {
		t.Fatalf("List() returned %d lines, want 2:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], "discard") {
		t.Errorf("newest entry = %q, want discard", lines[0])
	}
	if !strings.Contains(lines[1], "commit") {
		t.Errorf("older entry = %q, want commit", lines[1])
	}
}

func TestUndoRestoresClearedWorktreeAndRedoReturnsIt(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	write(t, "tracked.txt", "keep\n")
	git(t, dir, "add", "tracked.txt")
	git(t, dir, "commit", "-q", "-m", "tracked")

	write(t, "tracked.txt", "keep\nedit\n")
	write(t, "untracked.txt", "fresh\n")
	Record("clear")

	// Simulate the clear itself.
	git(t, dir, "reset", "--hard", "HEAD")
	if err := os.Remove("untracked.txt"); err != nil {
		t.Fatal(err)
	}

	if rc := Undo(1); rc != 0 {
		t.Fatalf("Undo(1) = %d", rc)
	}
	if got := read(t, "tracked.txt"); got != "keep\nedit\n" {
		t.Errorf("tracked.txt = %q after undo", got)
	}
	if got := read(t, "untracked.txt"); got != "fresh\n" {
		t.Errorf("untracked.txt = %q after undo", got)
	}

	if rc := Redo(); rc != 0 {
		t.Fatalf("Redo() = %d", rc)
	}
	if got := read(t, "tracked.txt"); got != "keep\n" {
		t.Errorf("tracked.txt = %q after redo", got)
	}
	if _, err := os.Stat("untracked.txt"); !os.IsNotExist(err) {
		t.Errorf("untracked.txt still exists after redo: %v", err)
	}
	if rc := Redo(); rc == 0 {
		t.Error("Redo() succeeded with nothing to redo")
	}
}

func TestUndoTwoGoesBackTwoEntries(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	write(t, "a.txt", "a\n")
	Record("commit")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "c1")

	write(t, "b.txt", "b\n")
	Record("commit")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "c2")

	if rc := Undo(2); rc != 0 {
		t.Fatalf("Undo(2) = %d", rc)
	}
	if got := git(t, dir, "rev-list", "--count", "HEAD"); got != "1" {
		t.Errorf("HEAD count after undo 2 = %s, want 1", got)
	}
	if got := read(t, "a.txt"); got != "a\n" {
		t.Errorf("a.txt = %q after undo 2", got)
	}
	if _, err := os.Stat("b.txt"); !os.IsNotExist(err) {
		t.Errorf("b.txt still exists after undo 2: %v", err)
	}

	if rc := Redo(); rc != 0 {
		t.Fatalf("Redo() = %d", rc)
	}
	if got := git(t, dir, "rev-list", "--count", "HEAD"); got != "3" {
		t.Errorf("HEAD count after redo = %s, want 3", got)
	}
	if got := read(t, "b.txt"); got != "b\n" {
		t.Errorf("b.txt = %q after redo", got)
	}
}

func TestNewSnapshotDropsRedo(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	write(t, "a.txt", "a\n")
	Record("commit")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "c1")

	write(t, "b.txt", "b\n")
	Record("commit")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "c2")

	if rc := Undo(2); rc != 0 {
		t.Fatalf("Undo(2) = %d", rc)
	}
	Record("commit")

	if rc := Redo(); rc == 0 {
		t.Error("Redo() succeeded after a new snapshot")
	}
	if _, ok := getRef(redoRef); ok {
		t.Error("redo ref was not cleared by the new snapshot")
	}
}

func TestRetentionKeepsNewestHundred(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	for i := 0; i < 105; i++ {
		write(t, "f.txt", fmt.Sprintf("%d\n", i))
		Record("commit")
		git(t, dir, "add", "-A")
		git(t, dir, "commit", "-q", "-m", fmt.Sprintf("c%d", i))
	}

	lines := strings.Split(strings.TrimSpace(list(t)), "\n")
	if len(lines) != keep {
		t.Fatalf("List() returned %d entries, want %d", len(lines), keep)
	}
	if !strings.Contains(lines[0], "commit") {
		t.Errorf("newest entry = %q", lines[0])
	}
}

func TestTrimClearsRedoOutsideTheNewChain(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	tree, err := emptyTree()
	if err != nil {
		t.Fatal(err)
	}
	prev := ""
	for i := 0; i <= keep; i++ {
		prev, err = commitEntry(tree, prev, "", "commit", "")
		if err != nil {
			t.Fatal(err)
		}
	}
	newest := prev
	redo, err := commitEntry(tree, "", "", "redo", "")
	if err != nil {
		t.Fatal(err)
	}

	setRef(oplogRef, newest)
	setRef(curRef, newest)
	setRef(redoRef, redo)
	trim()

	if _, ok := getRef(redoRef); ok {
		t.Error("redo ref survived trim outside the rebuilt chain")
	}
	if got := len(chain(mustRef(t, curRef))); got != keep {
		t.Errorf("chain length after trim = %d, want %d", got, keep)
	}
}

func mustRef(t *testing.T, ref string) string {
	t.Helper()
	sha, ok := getRef(ref)
	if !ok {
		t.Fatalf("%s is not set", ref)
	}
	return sha
}

func TestRecordWritesRefsTrailer(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	Record("commit")
	cur := mustRef(t, curRef)
	message, err := gitxOutput("log", "-1", "--format=%B", cur)
	if err != nil {
		t.Fatal(err)
	}
	subject, body, found := strings.Cut(message, "\n\n")
	if !found || subject != "commit" {
		t.Fatalf("message = %q, want subject commit and a trailer body", message)
	}
	blob, ok := strings.CutPrefix(strings.TrimSpace(body), "gg-refs: ")
	if !ok || strings.Contains(blob, "\n") {
		t.Fatalf("trailer body = %q, want one gg-refs line", body)
	}
	text, err := gitxOutput("cat-file", "blob", strings.TrimSpace(blob))
	if err != nil {
		t.Fatal(err)
	}
	head := git(t, dir, "rev-parse", "HEAD")
	if !strings.Contains(text, "refs/heads/main "+head) {
		t.Errorf("refs blob = %q, want refs/heads/main %s", text, head)
	}
	if strings.Contains(text, "refs/gg/oplog/") {
		t.Errorf("refs blob contains the oplog refs: %q", text)
	}
}

func TestRestoreRecreatesDeletedBranch(t *testing.T) {
	dir := newRepo(t)
	inDir(t, dir)

	git(t, dir, "branch", "side")
	sha := git(t, dir, "rev-parse", "refs/heads/side")
	Record("dbr")
	git(t, dir, "branch", "-D", "side")

	if rc := Undo(1); rc != 0 {
		t.Fatalf("Undo(1) = %d", rc)
	}
	if got := git(t, dir, "rev-parse", "refs/heads/side"); got != sha {
		t.Errorf("refs/heads/side = %s after undo, want %s", got, sha)
	}
}

func gitxOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
