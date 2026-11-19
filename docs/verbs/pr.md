# gg pr

Open a pull request for the current branch, and for every branch it stands on in
the stack, bottom-up. Each pull request bases on the branch under it, the same
parent `gg br -a` draws, so no metadata has to be stored anywhere.

## Usage

    gg pr [flags] [gh pr create args...]
    gg pr new [flags] [gh pr create args...]

## Arguments

    gh args...   reach `gh pr create` verbatim
    (none)       the current branch and its stack

## Flags

    -d, --draft    create the pull requests as drafts
    -1, --single   only the current branch, not the stack
    -p, --print    print the call and run nothing (a gg-wide flag)
    -h, --help     this page

## What it does

1. `pr new` is accepted and dropped; `pr review <n>` and `pr fixup <changeId>`
   are the other two faces of this verb, with pages of their own.
2. A detached HEAD is refused, because a pull request needs a branch.
3. The chain is the current branch, then its stack parents, up to the floor of
   the stack: the default branch, meaning the one `origin/HEAD` names, else a
   branch named `main`, `master`, `trunk` or `develop`, else the branch you are
   on. A `gitbutler/` branch ends the walk. `-1` keeps the chain to the current
   branch alone.
4. For each branch, base first: `git push --force-with-lease -u origin <branch>`.
   A failed push stops the run there, so a stack is never opened past the branch
   that could not be published.
5. A branch that already has a pull request is left alone and reported, so a
   second run only adds what the stack is still missing.
6. The base of each pull request is its stack parent, or the default branch when
   it has none: a sibling cut from the same commit bases on the floor, not on its
   sibling.
7. Runs `gh pr create --head <branch> --base <base> --fill`, adding `--draft`
   when asked, then your extra arguments.

`gh` must be installed: `gg tools` shows whether it is. The push itself needs
only git.

## Examples

    gg pr                       this branch and the stack under it
    gg pr -1                    this branch only
    gg pr -d                    drafts, one per stacked branch
    gg pr --reviewer alice      extra gh flags pass through
    gg pr new                   the explicit form, same verb
    gg pr                       feat/b bases on feat/a, feat/a bases on main

## Errors

    gg pr: detached HEAD, switch to a branch first
        A pull request needs a branch, not a raw commit.
    gg pr: cannot push feat/b
        The push failed; no pull request is created after it.
    gg pr: feat/a already has #42, left as it is
        The pull request exists; the run keeps going with the next branch.

## Git equivalent

None: gg does it in Go, with `git push --force-with-lease -u origin <branch>`
and `gh pr create`.

## See also

    gg pr review   mount a pull request in its own worktree
    gg pr fixup    one review fix as a commit of its own
    gg land        the same target, without a pull request
    gg br -a       the stack the chain is read from
    gg push        the push, with its own guards
