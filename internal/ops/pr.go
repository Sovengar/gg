package ops

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gg/internal/gitx"
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
// The base of a stacked branch is its stack parent, exactly the branch that
// `gg br -a` draws it under, so no metadata is stored.
func PRNew(args []string) int {
	if len(args) > 0 && args[0] == "review" {
		return PRReview(args[1:])
	}
	if len(args) > 0 && args[0] == "new" {
		args = args[1:]
	}
	draft, rest := prNewArgs(args)

	branch, err := gitx.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "" || branch == "HEAD" {
		ui.Errf("gg pr: detached HEAD, switch to a branch first")
		return 1
	}
	if _, ok := gitx.Upstream(); !ok {
		ui.Errf("gg pr: %s has no upstream, run gg push first", branch)
		return 1
	}
	argv := []string{"pr", "create", "--head", branch, "--base", prBase(branch), "--fill"}
	if draft {
		argv = append(argv, "--draft")
	}
	argv = append(argv, rest...)
	return ghRun(argv...)
}

// prNewArgs splits the draft flag from the arguments gh should see verbatim.
func prNewArgs(args []string) (draft bool, rest []string) {
	for _, a := range args {
		switch a {
		case "-d", "--draft":
			draft = true
		default:
			rest = append(rest, a)
		}
	}
	return draft, rest
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
