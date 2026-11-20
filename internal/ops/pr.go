package ops

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/ui"
)

// ghRun runs gh with inherited stdio. A package var so tests can stub it.
var ghRun = func(args ...string) int { return gitx.RunBin("gh", args...) }

// ghOutput runs gh and returns trimmed stdout.
var ghOutput = func(args ...string) (string, error) {
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// PR opens a pull request for the current branch, or mounts one to review.
// A stacked branch is published with the branches it stands on, bottom-up, the
// way `but pr new <top>` publishes a stack: each pull request bases on the one
// under it, exactly the branch that `gg br -a` draws it under, so no metadata
// is stored. -1 keeps the run to the current branch alone.
func PRNew(args []string) int {
	if len(args) > 0 && args[0] == "review" {
		return PRReview(args[1:])
	}
	if len(args) > 0 && args[0] == "fixup" {
		return PRFixup(args[1:])
	}
	if len(args) > 0 && args[0] == "new" {
		args = args[1:]
	}
	draft, single, rest := prNewArgs(args)

	branch, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "" || branch == "HEAD" {
		ui.Errf("gg pr: detached HEAD, switch to a branch first")
		return 1
	}
	for _, b := range prChain(branch, single) {
		if rc := prPublish(b, draft, rest); rc != 0 {
			return rc
		}
	}
	return 0
}

// PRFixup adds a review fixup commit instead of rewriting published history:
// the pull request then shows one new commit, and the fold happens at merge
// time with a squash merge or `gg rebase -i --autosquash`. The target must be
// an unpublished-to-this-branch commit: amending a lower PR from the top of the
// stack would attach the fix to the wrong pull request.
func PRFixup(args []string) int {
	if len(args) != 1 {
		ui.Errf("gg pr fixup <commit>")
		return 2
	}
	sha, ok := prFixupSha(args[0])
	if !ok {
		ui.Errf("gg pr fixup: no commit named %q", args[0])
		return 1
	}
	if gitx.InWip() {
		ui.Errf("gg pr fixup: HEAD is a wip commit. Run gg c -m <msg> first, then fixup.")
		return 1
	}
	branch, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "" || branch == "HEAD" {
		ui.Errf("gg pr fixup: detached HEAD, switch to the PR branch first")
		return 1
	}
	base := prFixupBase(branch)
	own, err := gitx.Output("rev-list", base+"..HEAD")
	if err != nil || !hasLine(own, sha) {
		ui.Errf("gg pr fixup: %s does not belong to %s (the commits above %s)", abbrev(sha), branch, base)
		ui.Errf("gg pr fixup: run gg go <branch> for the pull request that owns it")
		return 1
	}
	if _, err := gitx.Output("merge-base", "--is-ancestor", sha, "origin/"+branch); err != nil {
		ui.Errf("gg pr fixup: %s is not published on origin/%s", abbrev(sha), branch)
		ui.Errf("gg pr fixup: fix an unpublished commit with gg amend %s", args[0])
		return 1
	}
	if rc := gitx.Run("add", "-A"); rc != 0 {
		return rc
	}
	return gitx.Run("commit", "--fixup="+sha)
}

// prFixupBase names the ref a branch's own commits start above: the pull
// request base when gh can name it, so a stack that has already moved still
// tells which PR owns a commit, and the derived stack parent otherwise.
func prFixupBase(branch string) string {
	base, err := ghOutput("pr", "view", branch, "--json", "baseRefName", "-q", ".baseRefName")
	if err != nil || base == "" {
		base = prBase(branch)
	}
	if _, err := gitx.Output("rev-parse", "--verify", "--quiet", "origin/"+base); err == nil {
		return "origin/" + base
	}
	return base
}

// prFixupSha resolves a changeId, or any ref git already knows.
func prFixupSha(token string) (string, bool) {
	token = strings.TrimPrefix(token, "@")
	if ix, err := ids.Load(); err == nil {
		if ref, status := ix.Resolve(token); status == ids.Resolved {
			return ref.SHA, true
		}
	}
	sha, err := gitx.Output("rev-parse", "--verify", "--quiet", token+"^{commit}")
	return sha, err == nil && sha != ""
}

// hasLine reports whether text holds line as one of its lines.
func hasLine(text, line string) bool {
	for _, l := range strings.Split(text, "\n") {
		if l == line {
			return true
		}
	}
	return false
}

// prPublish pushes one branch and opens its pull request. A branch that
// already has one is left alone, so a second run after a change only adds the
// reviews the stack is missing.
func prPublish(branch string, draft bool, rest []string) int {
	if rc := prPush(branch); rc != 0 {
		return rc
	}
	base := prBase(branch)
	if num, ok := prExisting(branch); ok {
		ui.Errf("gg pr: %s already has #%d, left as it is", branch, num)
		return 0
	}
	ui.Errf("gg pr: %s bases on %s", branch, base)
	present := []string{"pr", "create", "--head", branch, "--base", base, "--fill"}
	if draft {
		present = append(present, "--draft")
	}
	return ghRun(append(present, rest...)...)
}

// prRun pushes one branch. A package var so tests can stub the push.
var prRun = func(args ...string) int { return gitx.Run(args...) }

// prPush publishes a branch the way the review path needs it: with a lease, so
// a rewritten branch lands only when the remote is still what this repository
// saw, and never a commit someone else pushed.
func prPush(branch string) int {
	if rc := prRun("push", "--force-with-lease", "-u", "origin", branch); rc != 0 {
		ui.Errf("gg pr: cannot push %s", branch)
		return rc
	}
	return 0
}

// prExisting names the pull request a branch already has, if any.
func prExisting(branch string) (int, bool) {
	out, err := ghOutput("pr", "view", branch, "--json", "number", "-q", ".number")
	if err != nil || out == "" {
		return 0, false
	}
	n, convErr := strconv.Atoi(out)
	if convErr != nil {
		return 0, false
	}
	return n, true
}

// prChain names the branches to publish, base first: every branch under the
// current one up to the floor of its stack, then the branch itself. A floor is
// never published, and -1 keeps the chain to the branch alone.
func prChain(branch string, single bool) []string {
	if single {
		return []string{branch}
	}
	rows := branchRows()
	floor := DefaultBranch()
	nodes := stackNodes(rows, floor)
	chain := []string{branch}
	seen := map[string]bool{branch: true}
	for p := prParent(nodes, branch); p != "" && !seen[p]; p = prParent(nodes, p) {
		if p == floor || strings.HasPrefix(p, "gitbutler/") {
			break
		}
		chain = append(chain, p)
		seen[p] = true
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

// prParent names the branch a branch is stacked on, or "" when it stands on
// the floor of its stack.
func prParent(nodes map[string]*stackNode, name string) string {
	if node, ok := nodes[name]; ok {
		return node.parent
	}
	return ""
}

// prNewArgs splits the draft flag and the single-branch flag from the
// arguments gh should see verbatim.
func prNewArgs(args []string) (draft, single bool, rest []string) {
	for _, a := range args {
		switch a {
		case "-d", "--draft":
			draft = true
		case "-1", "--single":
			single = true
		default:
			rest = append(rest, a)
		}
	}
	return draft, single, rest
}

// prBase names the base of a PR: the stack parent, or the default branch when
// the branch has no parent. A sibling cut from the same commit has no parent
// in the stack, so it bases on the default branch.
func prBase(branch string) string {
	def := DefaultBranch()
	if node, ok := stackNodes(branchRows(), def)[branch]; ok && node.parent != "" {
		return node.parent
	}
	return def
}

// prReview mounts a PR in its own worktree, or with its diff live when asked
// to edit: the worktree then sits on the PR base with the whole diff staged.
func PRReview(args []string) int {
	num, edit, err := prReviewArgs(args)
	if err != nil {
		ui.Errf("gg pr review: %v", err)
		return 2
	}
	return prReviewMount(num, edit)
}

// prReviewEdit is `gg pr review <n> -e`: the same mount, diff live.
func PRReviewEdit(args []string) int {
	return PRReview(append([]string{"-e"}, args...))
}

// prReviewArgs reads the PR number and the edit flag.
func prReviewArgs(args []string) (num int, edit bool, err error) {
	for _, a := range args {
		switch a {
		case "edit", "-e", "--edit":
			edit = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			return 0, false, fmt.Errorf("unknown flag %s", a)
		}
		if num != 0 {
			return 0, false, fmt.Errorf("name one pull request number")
		}
		n, convErr := strconv.Atoi(a)
		if convErr != nil || n < 1 {
			return 0, false, fmt.Errorf("no pull request numbered %q", a)
		}
		num = n
	}
	if num == 0 {
		return 0, false, fmt.Errorf("name the pull request number")
	}
	return num, edit, nil
}

// prReviewMount fetches the PR head and adds a detached worktree for it.
func prReviewMount(num int, edit bool) int {
	root, ok := repoRoot()
	if !ok {
		ui.Errf("gg pr review: not inside a work tree")
		return 1
	}
	path := worktreePath(root, fmt.Sprintf("pr-%d", num))
	if _, err := os.Stat(path); err == nil {
		ui.Errf("gg pr review: %s already exists", path)
		return 1
	}
	ref := fmt.Sprintf("refs/gg/pr/%d", num)
	if rc := gitx.Run("fetch", "origin", fmt.Sprintf("+refs/pull/%d/head:%s", num, ref)); rc != 0 {
		return rc
	}
	if rc := gitx.Run("worktree", "add", "--detach", path, ref); rc != 0 {
		return rc
	}
	if edit {
		base, err := prBaseCommit(num, ref)
		if err != nil {
			ui.Errf("gg pr review: %v", err)
			return 1
		}
		if rc := gitx.Run("-C", path, "reset", "--soft", base); rc != 0 {
			return rc
		}
	}
	ui.Errf("gg pr review: %s", path)
	return 0
}

// prBaseCommit is the commit the PR branched from: the merge base of the PR
// head and the remote base branch. A soft reset there leaves the whole PR diff
// in the worktree/index, ready to tweak.
func prBaseCommit(num int, ref string) (string, error) {
	base, err := ghOutput("pr", "view", strconv.Itoa(num), "--json", "baseRefName", "-q", ".baseRefName")
	if err != nil || base == "" {
		return "", fmt.Errorf("cannot read the base of PR %d", num)
	}
	sha, err := gitx.Output("merge-base", ref, "origin/"+base)
	if err != nil || sha == "" {
		return "", fmt.Errorf("no merge base between PR %d and origin/%s", num, base)
	}
	return sha, nil
}
