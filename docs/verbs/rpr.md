# gg rpr

Review a pull request without leaving your branch: fetch its head and mount it
in a worktree of its own, commits intact. It is `gg pr review <number>`.

## Usage

    gg rpr <number>
    gg pr review <number>
    gg pr review <number> -e
    gg epr <number>

## Aliases

    rpr    the short form of `pr review`

## Arguments

    <number>   the pull request number. Exactly one.

## Flags

    -e, --edit, edit   soft reset on the PR base, so the diff sits live in the tree
    -p, --print        print the call and run nothing (a gg-wide flag)
    -h, --help         this page

## What it does

1. Reads one number. No number, two numbers, a value that is not a number, or an
   unknown flag is an error, and nothing is fetched.
2. The worktree is a sibling directory of the repository, named after it:
   `/home/me/gg` gets `/home/me/gg.pr-42`. A directory that already exists is
   refused rather than rebuilt.
3. Fetches the pull request head into a gg ref of its own:
   `git fetch origin +refs/pull/42/head:refs/gg/pr/42`.
4. Runs `git worktree add --detach <path> refs/gg/pr/42`, so the review checkout
   sits on the PR head with a clean tree and no branch to disturb.
5. With `-e`, it asks `gh` for the PR base, takes the merge base of the PR head
   and `origin/<base>`, then runs `git -C <path> reset --soft <base>`. Every file
   the pull request touches then shows as a staged change on top of the base,
   ready to edit or amend.
6. Prints the path it mounted, prefixed `gg pr review: `.
7. Records an oplog entry and prints the footer. Removing the worktree afterwards
   is `gg wt rm` or `git worktree remove`.

`gh` must be installed for `-e`, which needs the base branch name; the plain
mount needs only git. `gg tools` shows the install state of both.

## Examples

    gg rpr 42                   mount PR 42, read the commits as they are
    gg pr review 42             the long form, same verb
    gg pr review 42 -e          the whole diff staged on the PR base
    gg epr 42                   the same as -e
    gg rpr 42                   again: refused, /home/me/gg.pr-42 already exists
    cd /home/me/gg.pr-42        then read, build and test there

## Errors

    gg pr review: name the pull request number
        No number was passed.
    gg pr review: name one pull request number
        Two numbers were passed.
    gg pr review: no pull request numbered "abc"
        The argument did not parse as a number.
    gg pr review: unknown flag -x
        Only -e, --edit and edit are read.
    gg pr review: not inside a work tree
        The mount needs a checkout to sit beside.
    gg pr review: /home/me/gg.pr-42 already exists
        The worktree is already mounted; use it, or remove it first.
    gg pr review: cannot read the base of PR 42
        gh could not name the base, so -e cannot place the diff.
    gg pr review: no merge base between PR 42 and origin/main
        The PR head and the remote base share no ancestor.

## Git equivalent

    git fetch origin +refs/pull/42/head:refs/gg/pr/42
    git worktree add --detach ../gg.pr-42 refs/gg/pr/42
    git -C ../gg.pr-42 reset --soft <merge-base>          with -e

## See also

    gg pr              open pull requests for the stack
    gg pr fixup        one review fix as a commit of its own
    gg wt              the worktrees, listed and removed
    gg discard         remove a worktree and the branch it holds
