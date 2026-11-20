package ops

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRNewArgs(t *testing.T) {
	cases := []struct {
		args          []string
		draft, single bool
		rest          []string
	}{
		{nil, false, false, nil},
		{[]string{"-d"}, true, false, nil},
		{[]string{"--draft"}, true, false, nil},
		{[]string{"-1"}, false, true, nil},
		{[]string{"--single", "-d"}, true, true, nil},
		{[]string{"--title", "two words", "-d", "--body", "why"}, true, false, []string{"--title", "two words", "--body", "why"}},
		{[]string{"--reviewer", "ana", "--label", "bug"}, false, false, []string{"--reviewer", "ana", "--label", "bug"}},
	}
	for _, c := range cases {
		draft, single, rest := prNewArgs(c.args)
		if draft != c.draft || single != c.single || strings.Join(rest, "\x00") != strings.Join(c.rest, "\x00") {
			t.Errorf("prNewArgs(%q) = %v, %v, %q; want %v, %v, %q",
				c.args, draft, single, rest, c.draft, c.single, c.rest)
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

// prStub replaces the push and the gh calls with recorders. ghOutput reports
// that no pull request exists, unless existing names one.
func prStub(t *testing.T, existing string) (pushed, created *[]string) {
	t.Helper()
	oldRun, oldGH, oldOut := prRun, ghRun, ghOutput
	t.Cleanup(func() { prRun, ghRun, ghOutput = oldRun, oldGH, oldOut })
	push, make := &[]string{}, &[]string{}
	prRun = func(args ...string) int { *push = append(*push, strings.Join(args, " ")); return 0 }
	ghOutput = func(args ...string) (string, error) {
		if existing == "" {
			return "", errors.New("no pull requests found")
		}
		return existing, nil
	}
	ghRun = func(args ...string) int { *make = append(*make, strings.Join(args, " ")); return 0 }
	return push, make
}

// prStackRepo is a repo whose HEAD, feat/b, stands on feat/a, which stands on
// main. feat/c is a sibling of feat/a, cut from the same main commit.
func prStackRepo(t *testing.T) string {
	t.Helper()
	dir := prRepo(t)
	prGit(t, dir, "switch", "-q", "-c", "feat/a")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "a1")
	prGit(t, dir, "switch", "-q", "-c", "feat/b")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "b1")
	prGit(t, dir, "switch", "-q", "main")
	prGit(t, dir, "switch", "-q", "-c", "feat/c")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "c1")
	prGit(t, dir, "switch", "-q", "feat/b")
	return dir
}

func TestPRChainNamesTheBaseFirst(t *testing.T) {
	prInDir(t, prStackRepo(t))
	cases := []struct {
		branch string
		single bool
		want   string
	}{
		{"feat/b", false, "feat/a feat/b"},
		{"feat/a", false, "feat/a"},
		{"feat/c", false, "feat/c"},
		{"feat/b", true, "feat/b"},
	}
	for _, c := range cases {
		got := strings.Join(prChain(c.branch, c.single), " ")
		if got != c.want {
			t.Errorf("prChain(%q, %v) = %q, want %q", c.branch, c.single, got, c.want)
		}
	}
}

func TestPRNewPublishesTheStack(t *testing.T) {
	prInDir(t, prStackRepo(t))
	pushed, created := prStub(t, "")

	if rc := PRNew(nil); rc != 0 {
		t.Fatalf("PRNew = %d, want 0", rc)
	}
	wantPush := "push --force-with-lease -u origin feat/a push --force-with-lease -u origin feat/b"
	if got := strings.Join(*pushed, " "); got != wantPush {
		t.Errorf("pushed %q, want %q", got, wantPush)
	}
	wantCreate := "pr create --head feat/a --base main --fill pr create --head feat/b --base feat/a --fill"
	if got := strings.Join(*created, " "); got != wantCreate {
		t.Errorf("gh %q, want %q", got, wantCreate)
	}
}

func TestPRNewSinglePublishesHeadOnly(t *testing.T) {
	prInDir(t, prStackRepo(t))
	pushed, created := prStub(t, "")

	if rc := PRNew([]string{"-1", "-d"}); rc != 0 {
		t.Fatalf("PRNew(-1) = %d, want 0", rc)
	}
	if got := strings.Join(*pushed, " "); got != "push --force-with-lease -u origin feat/b" {
		t.Errorf("pushed %q, want only feat/b", got)
	}
	if got := strings.Join(*created, " "); got != "pr create --head feat/b --base feat/a --fill --draft" {
		t.Errorf("gh %q, want the feat/b draft", got)
	}
}

func TestPRNewLeavesAnExistingPR(t *testing.T) {
	prInDir(t, prStackRepo(t))
	pushed, created := prStub(t, "42")

	if rc := PRNew(nil); rc != 0 {
		t.Fatalf("PRNew = %d, want 0", rc)
	}
	if len(*pushed) != 2 {
		t.Errorf("pushed %q, want both branches pushed before their check", *pushed)
	}
	if len(*created) != 0 {
		t.Errorf("gh %q, want no pull request created", *created)
	}
}

func TestPRNewStopsWhenAPushFails(t *testing.T) {
	prInDir(t, prStackRepo(t))
	prStub(t, "")
	prRun = func(args ...string) int { return 1 }
	created := &[]string{}
	ghRun = func(args ...string) int { *created = append(*created, args[0]); return 0 }

	if rc := PRNew(nil); rc != 1 {
		t.Fatalf("PRNew with a failed push = %d, want 1", rc)
	}
	if len(*created) != 0 {
		t.Errorf("gh %q, want no pull request created", *created)
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

// prFixupRepo builds a repository with a bare origin, main pushed, and a
// two-branch stack feat/a -> feat/b pushed, on feat/a.
func prFixupRepo(t *testing.T) (dir, a, b string) {
	t.Helper()
	dir = t.TempDir()
	prGit(t, dir, "init", "-q", "-b", "main", ".")
	prGit(t, dir, "config", "user.email", "test@example.com")
	prGit(t, dir, "config", "user.name", "Test")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	remote := filepath.Join(t.TempDir(), "origin.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	prGit(t, remote, "init", "-q", "--bare", ".")
	prGit(t, dir, "remote", "add", "origin", remote)
	prGit(t, dir, "push", "-qu", "origin", "main")

	prGit(t, dir, "switch", "-q", "-c", "feat/a")
	prWrite(t, dir, "a.txt")
	prGit(t, dir, "add", "-A")
	prGit(t, dir, "commit", "-qm", "a1")
	a = prGit(t, dir, "rev-parse", "HEAD")
	prGit(t, dir, "push", "-qu", "origin", "feat/a")

	prGit(t, dir, "switch", "-q", "-c", "feat/b")
	prWrite(t, dir, "b.txt")
	prGit(t, dir, "add", "-A")
	prGit(t, dir, "commit", "-qm", "b1")
	b = prGit(t, dir, "rev-parse", "HEAD")
	prGit(t, dir, "push", "-qu", "origin", "feat/b")
	prInDir(t, dir)
	return dir, a, b
}

// prFixupBases stubs the PR base lookup, so the guard reads the base the forge
// reports without calling gh.
func prFixupBases(t *testing.T, bases map[string]string) {
	t.Helper()
	old := ghOutput
	t.Cleanup(func() { ghOutput = old })
	ghOutput = func(args ...string) (string, error) {
		if len(args) >= 3 {
			if base, ok := bases[args[2]]; ok {
				return base, nil
			}
		}
		return "", errors.New("no pull request")
	}
}

func TestPRFixupCreatesAVisibleCommit(t *testing.T) {
	dir, _, _ := prFixupRepo(t)
	prFixupBases(t, map[string]string{"feat/a": "main"})
	prGit(t, dir, "switch", "-q", "feat/a")
	head := prGit(t, dir, "rev-parse", "HEAD")
	prWrite(t, dir, "review.txt")

	if rc := PRFixup([]string{"@HEAD"}); rc != 0 {
		t.Fatalf("PRFixup = %d, want 0", rc)
	}
	if got := prGit(t, dir, "log", "-1", "--format=%s"); got != "fixup! a1" {
		t.Errorf("subject = %q, want fixup! a1", got)
	}
	if got := prGit(t, dir, "rev-parse", "HEAD^"); got != head {
		t.Errorf("fixup parent = %q, want %q", got, head)
	}
	if got := prGit(t, dir, "log", "--format=%s", "origin/feat/a..HEAD"); got != "fixup! a1" {
		t.Errorf("origin/feat/a..HEAD = %q, want one fixup commit", got)
	}
}

func TestPRFixupRefusesAForeignCommit(t *testing.T) {
	dir, a, _ := prFixupRepo(t)
	prFixupBases(t, map[string]string{"feat/a": "main", "feat/b": "feat/a"})
	prWrite(t, dir, "review.txt")
	// HEAD is feat/b; a1 belongs to feat/a, so the fix would land on the wrong PR.
	if rc := PRFixup([]string{a}); rc != 1 {
		t.Fatalf("PRFixup(foreign) = %d, want 1", rc)
	}
	if got := prGit(t, dir, "log", "-1", "--format=%s"); got != "b1" {
		t.Errorf("HEAD moved to %q, want b1 untouched", got)
	}
}

func TestPRFixupRefusesAForeignCommitWhenTheStackMoved(t *testing.T) {
	dir, a, _ := prFixupRepo(t)
	prFixupBases(t, map[string]string{"feat/a": "main", "feat/b": "feat/a"})
	// A fixup on feat/a moves its tip, so the derived stack can no longer see
	// feat/a under feat/b; the forge-reported base still names the real owner.
	prGit(t, dir, "switch", "-q", "feat/a")
	prWrite(t, dir, "review-a.txt")
	if rc := PRFixup([]string{"HEAD"}); rc != 0 {
		t.Fatalf("PRFixup on feat/a = %d, want 0", rc)
	}
	prGit(t, dir, "switch", "-q", "feat/b")
	prWrite(t, dir, "review-b.txt")
	if rc := PRFixup([]string{a}); rc != 1 {
		t.Fatalf("PRFixup(foreign after the stack moved) = %d, want 1", rc)
	}
}

func TestPRFixupRefusesAnUnpublishedCommit(t *testing.T) {
	dir := prRepo(t)
	prFixupBases(t, map[string]string{})
	prGit(t, dir, "switch", "-q", "-c", "feat/x")
	prGit(t, dir, "commit", "-q", "--allow-empty", "-m", "x1")
	prWrite(t, dir, "review.txt")
	prInDir(t, dir)

	if rc := PRFixup([]string{"HEAD"}); rc != 1 {
		t.Fatalf("PRFixup(unpublished) = %d, want 1", rc)
	}
	if got := prGit(t, dir, "log", "-1", "--format=%s"); got != "x1" {
		t.Errorf("HEAD moved to %q, want x1 untouched", got)
	}
}

func TestPRFixupNeedsExactlyOneArgument(t *testing.T) {
	if rc := PRFixup(nil); rc != 2 {
		t.Errorf("PRFixup(nil) = %d, want 2", rc)
	}
	if rc := PRFixup([]string{"a", "b"}); rc != 2 {
		t.Errorf("PRFixup(two) = %d, want 2", rc)
	}
}
