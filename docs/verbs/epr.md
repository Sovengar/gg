# gg epr

Review a pull request with its diff live: mount it in a worktree of its own,
then soft reset onto the PR base so the whole change sits in the index as one
diff. Reach for it when you have to change the pull request rather than read it
commit by commit.

## Usage

    gg epr <number>
    gg pr review <number> -e
    gg pr review -e <number>

## Aliases

    epr    the short form of `pr review -e`; all three forms are one call

## Arguments

    <number>   the pull request number. Exactly one.

The number is read as typed: `epr` resolves no changeIds, so nothing in the
argument list is replaced before the verb runs.

## Flags

    -e, --edit, edit   implied by `epr`; the long form carries it
    -p, --print        print the call and run nothing (a gg-wide flag)
    -h, --help         this page

Any other flag is refused, so a typo fails before anything is fetched. `-e` and
the number may come in either order.

## What it does

1. Adds `-e` to the arguments, then reads them the way `gg pr review` does: one
   number, the edit flag, nothing else.
2. The worktree is a sibling of the repository, named after it:
   `/home/me/gg` gets `/home/me/gg.pr-42`. A directory that already exists is
   refused rather than rebuilt.
3. Fetches the pull request head into a gg ref:
   `git fetch origin +refs/pull/42/head:refs/gg/pr/42`.
4. Runs `git worktree add --detach <path> refs/gg/pr/42`, so the review checkout
   starts on the PR head with a clean tree.
5. Asks `gh` for the base branch
   (`gh pr view 42 --json baseRefName -q .baseRefName`) and takes the merge base
   of the PR head and `origin/<base>`.
6. Runs `git -C <path> reset --soft <merge-base>`. HEAD moves to the base; the
   index and the worktree stay at the PR head, so `git diff --staged` shows the
   whole pull request as staged changes and `git diff` shows nothing.
7. Prints the mounted path on stderr, prefixed `gg pr review: `, records an
   oplog entry and prints the footer. The footer still describes the checkout
   you ran the verb in, which the mount did not move.

`gh` must be installed for the base lookup, because `-e` needs the base branch
name; the plain mount needs only git. `gg tools` lists only the `tool:` verbs,
so `gh` is not among them. Removing the worktree afterwards is `gg wt rm` or
`git worktree remove`.

## Examples

    gg epr 42                       mount PR 42 with its diff staged
    gg pr review 42 -e              the long form, same call
    git -C ../gg.pr-42 diff --staged   read the whole PR as one diff
    gg epr 42                       again: refused, /home/me/gg.pr-42 exists
    gg epr 42 -p                    print the gg call, mount nothing

## Errors

The prefix is `gg pr review: ` however the verb was spelled, `epr` included.

    gg pr review: name the pull request number
        No number was passed.
    gg pr review: name one pull request number
        Two numbers were passed.
    gg pr review: no pull request numbered "abc"
        The argument did not parse as a number, or was lower than 1.
    gg pr review: unknown flag -x
        Only -e, --edit and edit are read.
    gg pr review: not inside a work tree
        The mount needs a checkout to sit beside.
    gg pr review: /home/me/gg.pr-42 already exists
        The worktree is already mounted; use it, or remove it first.
    gg pr review: cannot read the base of PR 42
        gh could not name the base, so the diff cannot be placed.
    gg pr review: no merge base between PR 42 and origin/main
        The PR head and the remote base share no ancestor.

## Git equivalent

    git fetch origin +refs/pull/42/head:refs/gg/pr/42
    git worktree add --detach ../gg.pr-42 refs/gg/pr/42
    git -C ../gg.pr-42 reset --soft <merge-base of the PR head and origin/main>

## See also

    gg rpr       the same mount, commits intact and no reset
    gg pr        open pull requests for the stack
    gg fpr       the fix on a published commit, without a worktree
    gg wt        the worktrees, listed and removed
    gg go        walk to where a branch lives
