# gg dash

The default view: bare `gg` runs it. It draws the tree `gg status` draws, but
the lanes are the five newest branches with the default branch always among
them, instead of the branches a sibling worktree holds.

## Usage

    gg
    gg dash [count] [flags]

## Arguments

    count   how many commits each lane lists, 6 by default. Any other argument
            is ignored.

## Flags

    -f, --files   five changed paths under every commit, with a `+N more` tail
    -a, --all     every changed path under every commit
    -h, --help    this page, even when other arguments are present

## What it does

1. Draws the shape of `gg status`: the uncommitted lane with its changeIds, one
   lane per branch, the common base and the two hints.
2. Picks the lanes differently: the current branch first, then the most recently
   committed branches, with the last slot kept for the default branch. That
   branch is the one `origin/HEAD` points at, else `main`, `master`, `trunk` or
   `develop`. Five lanes at most, the default branch included.
3. Keeps a lane with nothing above the common base and prints
   `(at the common base)` under it, so the default branch always has its row
   here, even when it has no commit of its own.
4. Groups the lanes into stacks the way `gg bra` reads them: one box per stack,
   opened with the branch on top and closed with a `├╯` corner.
5. Adds `wip` to the uncommitted header while HEAD is a wip commit, and `in
   <dir>` when you run it in a subdirectory.

With no arguments at all, gg runs dash, so `gg -p` prints `gg dash` and runs
nothing. The verb is hidden from `gg help`: the table lists the verbs you type,
and this one is what you get by typing nothing.

## Examples

    gg              the five newest branches, 6 commits each
    gg dash         the same, spelled out
    gg dash 10      ten commits per lane
    gg dash -f      five changed paths under every commit
    gg dash -a      every changed path under every commit

## Errors

    gg: no commits yet
        HEAD does not resolve: an empty repository has no tree to draw;
        exits 1.

## Git equivalent

None: gg does it in Go. It composes
`git status --short --untracked-files=all`,
`git for-each-ref --sort=-committerdate refs/heads`, `git log <base>..<tip>` and
`git merge-base` itself.

## See also

    gg status   the same tree over the worktree lanes
    gg stf      five changed paths per commit
    gg sta      every changed path per commit
    gg bra      the stacks drawn as stacks
    gg log      the graph without the lanes
