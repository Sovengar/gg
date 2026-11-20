# Fixup after review on stacked PRs

Status: reviewed and implemented on 2026-11-18 against git 2.56.1, gh 2.104.0 and gg at
`257748d`. Reproduction lives in a throwaway repository; no production remote
was touched.

Question: how does gg let a reviewer-driven fix reach only the pull request it
belongs to when several PRs are stacked on each other?

## What gg already does

- `gg amend` (no argument) runs `git commit -a --amend --no-edit`: the whole
  worktree folds into HEAD. It only fixes the top commit of the current branch.
- `gg amend <changeId>` runs `git history fixup <changeId>` (git >= 2.56):
  staged changes fold into that commit and every descendant is replayed.
  `git history` updates all descendant branches by default
  (`--update-refs=branches`), so a stack `main -> a -> b -> c` ends with
  a, b and c all pointing at rewritten commits.
- `gg pr` walks the stack bottom-up, pushes each branch with
  `--force-with-lease -u origin <branch>` and calls `gh pr create --head <b>
  --base <parent>`. A branch that already has a PR is pushed and then left
  alone, so a second run after a fix only refreshes the heads.
- `gg pr review <n> [-e]` mounts `refs/pull/<n>/head` in a detached worktree
  (`-e` soft-resets on the base to expose the PR diff). It is read/inspect
  only: the mounted worktree has no branch and no push path back.

## Reproduction

Local bare remote plus a stub `gh`, so no GitHub account is needed.

```sh
bin=/tmp/ggfix/bin
mkdir -p "$bin" && cd /tmp/ggfix
git init -q --bare remote.git
git init -q -b main repo && cd repo
git config user.email t@t && git config user.name T
echo base > f.txt && git add . && git commit -qm base
git remote add origin ../remote.git && git push -qu origin main

git switch -qc a && echo A > a.txt && git add . && git commit -qm A
git switch -qc b && echo B > b.txt && git add . && git commit -qm B
git switch -qc c && echo C > c.txt && git add . && git commit -qm C
git push -qu origin a b c

# stub: first run creates PRs, later runs report the existing number
cat > "$bin/gh" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
  "pr view") [ -f /tmp/ggfix/prs ] && echo 1 || exit 1 ;;
  "pr create") touch /tmp/ggfix/prs; echo https://example/pr/1 ;;
esac
EOF
chmod +x "$bin/gh"

PATH="$bin:$PATH" gg pr          # creates 3 PRs: main<-a<-b<-c
PATH="$bin:$PATH" gg pr          # second run: pushes only, PRs untouched
```

Review asks for a change on the lowest PR (`a`), while HEAD is still `c`:

```sh
echo "A v2" > a.txt && git add a.txt
gg st                            # read the changeId of commit A
gg amend <changeId-of-A>         # git history fixup: rewrites a, b, c
gg pr                            # pushes a, b, c with a lease; PRs left as is
```

Observed:

```
$ git for-each-ref --format='%(refname:short) %(objectname:short)' refs/heads refs/remotes
a a80dfda        origin/a a80dfda
b 4ab2d52        origin/b 4ab2d52
c 7d3cc08        origin/c 7d3cc08
main e2e73fc     origin/main e2e73fc

$ git diff --stat origin/main origin/a   # PR 1
 a.txt | 1 +
$ git diff --stat origin/a origin/b      # PR 2
 b.txt | 1 +
$ git diff --stat origin/b origin/c      # PR 3
 c.txt | 1 +
```

The fix reached only PR 1's diff. PRs 2 and 3 keep their own diff because
their commits were replayed unchanged on the rewritten parent, so the review
comments survive (the commits are marked outdated by GitHub, not deleted).
`gg pr` refused to create a second PR for any branch.

Without a `Change-Id` trailer, the rewritten commit gets a new sha and therefore
a new derived changeId. Install the hook first so ids stay stable:

```sh
gg hooks                          # commit-msg hook adds Change-Id
gg st                             # id before: nzm
gg amend                          # id after:  nzm
```

## Limitations found

1. `gg amend` never checks whether the target commit is published. Rewriting a
   commit that already backs a PR immediately produces a force-push need; only
   the leased push in `gg pr` prevents damage, and nothing warns the user.
   `gg new` already refuses a pushed wip commit (`--force` overrides); `amend`
   has no equivalent guard.
2. There is no way to add a review fixup as its own commit (`git commit
   --fixup=<sha>`) and autosquash it later. `gg amend <id>` rewrites history
   immediately, so the reviewer loses the incremental "what I changed after
   your review" view.
3. Every branch above the amended one is force-pushed. The content of their PRs
   is unchanged, but GitHub shows a force-push event on each PR. That is
   inherent to stacked branches; a single-branch run (`gg pr -1`) avoids
   touching siblings but not the ancestors.
4. `gg pr review <n> -e` gives no way to push a fix back to the PR head: the
   worktree is detached and has no branch. The fix must be made in the branch
   checkout, then `gg pr` publishes it.
5. `gg amend <id>` needs git >= 2.56; older git exits with the version hint.
   `gg amend` (HEAD) works everywhere through `git commit --amend`.
6. `gg pr` hardcodes `origin` in both the push and the PR lookup. `gg remote`
   only lists or edits git remotes (a bare `gg remote` is `git remote -v`), so
   a repository whose review remote has another name must rename it or add an
   `origin` alias.

## Implemented

`gg pr fixup <changeId>` (alias `gg fpr`) now stages the whole tree and runs
`git commit --fixup=<sha>`. It refuses a target outside `prBase..HEAD` (the fix
would land on another PR) and a target not reachable from `origin/<branch>`
(not published, so `gg amend` is the tool). The pull request shows the fixup as
one new commit. A squash merge folds it; a rebase or merge-commit strategy
needs `gg rebase -i --autosquash <base>` first.

## Remaining suggestions

1. `gg amend --dry-run [<id>]`: pass through `git history fixup --dry-run` and
   print the ref updates, matching the existing `-p`/`--dry` convention.
2. Guard published commits: before `gg amend <id>`, check
   `git for-each-ref refs/remotes` for a tip that contains the target; warn with
   the branch and PR, and require `-f` when the commit is published. Reuse the
   check `gg new` already performs on the wip commit.
3. `gg pr --only <branch>`: publish just the named branch (and its push), for
   the case where only the lowest PR needs a refresh and the user does not want
   every PR above it force-pushed. `-1` today means "current branch only", not
   "this branch only".
4. Optional: after `gg pr review <n> -e`, print the exact follow-up commands
   (`gg go <branch>`, apply, `gg amend <id>`, `gg pr`) so the detached worktree
   does not look like a dead end.
