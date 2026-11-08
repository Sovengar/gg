package ops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRNewArgs(t *testing.T) {
	cases := []struct {
		args  []string
		draft bool
		rest  []string
	}{
		{nil, false, nil},
		{[]string{"-d"}, true, nil},
		{[]string{"--draft"}, true, nil},
		{[]string{"--title", "two words", "-d", "--body", "why"}, true, []string{"--title", "two words", "--body", "why"}},
		{[]string{"--reviewer", "ana", "--label", "bug"}, false, []string{"--reviewer", "ana", "--label", "bug"}},
	}
	for _, c := range cases {
		draft, rest := prNewArgs(c.args)
		if draft != c.draft || strings.Join(rest, "\x00") != strings.Join(c.rest, "\x00") {
			t.Errorf("prNewArgs(%q) = %v, %q; want %v, %q", c.args, draft, rest, c.draft, c.rest)
		}
	}
}

func TestPRReviewArgs(t *testing.T) {
	cases := []struct {
		args []string
		num  int
		edit bool
		bad  bool
	}{
		{[]string{"42"}, 42, false, false},
		{[]string{"42", "edit"}, 42, true, false},
		{[]string{"42", "-e"}, 42, true, false},
		{[]string{"-e", "42"}, 42, true, false},
		{[]string{"--edit", "7"}, 7, true, false},
		{nil, 0, false, true},
		{[]string{"edit"}, 0, false, true},
		{[]string{"abc"}, 0, false, true},
		{[]string{"0"}, 0, false, true},
		{[]string{"42", "43"}, 0, false, true},
		{[]string{"--stray", "42"}, 0, false, true},
	}
	for _, c := range cases {
		num, edit, err := prReviewArgs(c.args)
		if c.bad {
			if err == nil {
				t.Errorf("prReviewArgs(%q) = %d, %v; want an error", c.args, num, edit)
			}
			continue
		}
		if err != nil {
			t.Errorf("prReviewArgs(%q) failed: %v", c.args, err)
			continue
		}
		if num != c.num || edit != c.edit {
			t.Errorf("prReviewArgs(%q) = %d, %v; want %d, %v", c.args, num, edit, c.num, c.edit)
		}
	}
}

// prGit runs git in dir and fails the test on error.
func prGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func prInDir(t *testing.T, dir string) {
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

func prRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prGit(t, dir, "init", "-q", "-b", "main", ".")
	prGit(t, dir, "config", "user.email", "test@example.com")
	prGit(t, dir, "config", "user.name", "Test")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	return dir
}

func prWrite(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPRBaseSelectsStackParent(t *testing.T) {
	dir := prRepo(t)
	prGit(t, dir, "switch", "-q", "-c", "feat/a")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "a1")
	prGit(t, dir, "switch", "-q", "-c", "feat/b")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "b1")
	// feat/c is a sibling of feat/a, cut from the same main commit
	prGit(t, dir, "switch", "-q", "main")
	prGit(t, dir, "switch", "-q", "-c", "feat/c")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "c1")
	prInDir(t, dir)

	cases := []struct{ branch, want string }{
		{"feat/a", "main"},
		{"feat/b", "feat/a"},
		{"feat/c", "main"}, // the sibling bases on the default, not on feat/a
		{"main", "main"},
	}
	for _, c := range cases {
		if got := prBase(c.branch); got != c.want {
			t.Errorf("prBase(%q) = %q, want %q", c.branch, got, c.want)
		}
	}
}

// prUpstreamRepo is a repo on feat/a whose upstream is main.
func prUpstreamRepo(t *testing.T) string {
	t.Helper()
	dir := prRepo(t)
	prGit(t, dir, "switch", "-q", "-c", "feat/a")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "a1")
	prGit(t, dir, "update-ref", "refs/remotes/origin/main", "main")
	prGit(t, dir, "config", "remote.origin.url", dir)
	prGit(t, dir, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	prGit(t, dir, "config", "branch.feat/a.remote", "origin")
	prGit(t, dir, "config", "branch.feat/a.merge", "refs/heads/main")
	return dir
}

func TestPRNewCallsGH(t *testing.T) {
	dir := prUpstreamRepo(t)
	prInDir(t, dir)

	old := ghRun
	t.Cleanup(func() { ghRun = old })
	var got []string
	ghRun = func(args ...string) int {
		got = append([]string(nil), args...)
		return 0
	}

	if rc := PRNew(nil); rc != 0 {
		t.Fatalf("PRNew = %d, want 0", rc)
	}
	want := "pr create --head feat/a --base main --fill"
	if strings.Join(got, " ") != want {
		t.Errorf("gh %q, want %q", strings.Join(got, " "), want)
	}

	got = nil
	if rc := PRNew([]string{"new", "--label", "bug"}); rc != 0 {
		t.Fatalf("PRNew(new ...) = %d, want 0", rc)
	}
	want = "pr create --head feat/a --base main --fill --label bug"
	if strings.Join(got, " ") != want {
		t.Errorf("gh %q, want %q", strings.Join(got, " "), want)
	}

	got = nil
	if rc := PRNew([]string{"-d", "--title", "two words", "--reviewer", "ana"}); rc != 0 {
		t.Fatalf("PRNew(-d ...) = %d, want 0", rc)
	}
	want = "pr create --head feat/a --base main --fill --draft --title two words --reviewer ana"
	if strings.Join(got, " ") != want {
		t.Errorf("gh %q, want %q", strings.Join(got, " "), want)
	}
}

func TestPRNewNoUpstreamRefuses(t *testing.T) {
	dir := prRepo(t)
	prGit(t, dir, "switch", "-q", "-c", "feat/a")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "a1")
	prInDir(t, dir)

	old := ghRun
	t.Cleanup(func() { ghRun = old })
	called := false
	ghRun = func(args ...string) int {
		called = true
		return 0
	}

	if rc := PRNew(nil); rc != 1 {
		t.Fatalf("PRNew with no upstream = %d, want 1", rc)
	}
	if called {
		t.Error("gh was invoked for a branch with no upstream")
	}
}

// prFixture builds origin with PR 42 on top of main, and a clone to review it.
type prFixture struct {
	work string
	head string
	base string
}

func prReviewFixture(t *testing.T) prFixture {
	t.Helper()
	tmp := t.TempDir()
	origin := filepath.Join(tmp, "origin.git")
	prGit(t, tmp, "init", "--bare", "-q", "-b", "main", origin)

	seed := filepath.Join(tmp, "seed")
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	prGit(t, seed, "init", "-q", "-b", "main", ".")
	prGit(t, seed, "config", "user.email", "test@example.com")
	prGit(t, seed, "config", "user.name", "Test")
	prWrite(t, seed, "base.txt")
	prGit(t, seed, "add", "base.txt")
	prGit(t, seed, "commit", "-q", "-m", "base")
	base := prGit(t, seed, "rev-parse", "HEAD")
	prGit(t, seed, "switch", "-q", "-c", "feature")
	prWrite(t, seed, "pr.txt")
	prGit(t, seed, "add", "pr.txt")
	prGit(t, seed, "commit", "-q", "-m", "pr")
	head := prGit(t, seed, "rev-parse", "HEAD")
	prGit(t, seed, "remote", "add", "origin", origin)
	prGit(t, seed, "push", "-q", "origin", "main")
	prGit(t, seed, "push", "-q", "origin", "feature:refs/pull/42/head")

	work := filepath.Join(tmp, "work")
	prGit(t, tmp, "clone", "-q", origin, work)
	prGit(t, work, "config", "user.email", "test@example.com")
	prGit(t, work, "config", "user.name", "Test")
	return prFixture{work: work, head: head, base: base}
}

func TestPRReviewMountsWorktree(t *testing.T) {
	f := prReviewFixture(t)
	prInDir(t, f.work)

	path := worktreePath(f.work, "pr-42")
	if rc := PRReview([]string{"42"}); rc != 0 {
		t.Fatalf("PRReview(42) = %d, want 0", rc)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("worktree %s missing: %v", path, err)
	}
	if got := prGit(t, path, "rev-parse", "HEAD"); got != f.head {
		t.Errorf("worktree HEAD = %s, want the PR head %s", got, f.head)
	}
	if status := prGit(t, path, "status", "--porcelain"); status != "" {
		t.Errorf("worktree is not clean: %q", status)
	}
	if got := prGit(t, path, "rev-parse", "--abbrev-ref", "HEAD"); got != "HEAD" {
		t.Errorf("worktree is on %q, want detached HEAD", got)
	}

	// the same path is refused instead of being rebuilt
	if rc := PRReview([]string{"42"}); rc != 1 {
		t.Errorf("PRReview(42) over an existing worktree = %d, want 1", rc)
	}
}

func TestPRReviewEditLeavesDiffLive(t *testing.T) {
	f := prReviewFixture(t)
	prInDir(t, f.work)

	old := ghOutput
	t.Cleanup(func() { ghOutput = old })
	ghOutput = func(args ...string) (string, error) {
		want := "pr view 42 --json baseRefName -q .baseRefName"
		if strings.Join(args, " ") != want {
			t.Errorf("gh %q, want %q", strings.Join(args, " "), want)
		}
		return "main", nil
	}

	path := worktreePath(f.work, "pr-42")
	if rc := PRReviewEdit([]string{"42"}); rc != 0 {
		t.Fatalf("PRReviewEdit(42) = %d, want 0", rc)
	}
	if got := prGit(t, path, "rev-parse", "HEAD"); got != f.base {
		t.Errorf("worktree HEAD = %s, want the PR base %s", got, f.base)
	}
	staged := prGit(t, path, "diff", "--cached", "--name-only")
	if !strings.Contains(staged, "pr.txt") {
		t.Errorf("staged diff = %q, want it to hold the PR change", staged)
	}
}
