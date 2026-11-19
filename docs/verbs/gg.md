# gg gg

The dashboard: a bare `gg` draws the tree over the five newest branches, one
lane each, with the uncommitted paths on top. This row is the help table's name
for that default view; `gg dash` spells the same call out.

## Usage

    gg

## Arguments

    (none)   the view takes no argument. Anything after the verb is read as a
             verb of its own, so a count or `-f`/`-a` needs the spelled-out
             call.

The name is a help row, not a dispatch of its own: `gg gg -h` prints this page,
and `gg gg` on its own reaches git unchanged (`git gg`), which git refuses. The
dashboard is the empty command line, nothing after `gg`.

## Flags

    -p, --print   print the substituted call, `gg dash`, and draw nothing
    -n, --dry     the same
    -h, --help    this page

There is no flag for the view itself here. To choose a count or a file list, say
`gg dash` and pass it there.

## What it does

1. With no verb at all, gg substitutes `dash` before anything else runs, so the
   empty command line and `gg dash` are one call. `-p` prints that substitution:

        gg dash

2. Draws the shape `gg status` draws: the uncommitted lane with its changeIds,
   then one lane per branch with six commits each, the commit the lanes part
   from, and two hints.
3. Picks the lanes differently from `gg status`: the current branch first, then
   the most recently committed branches, with the last slot kept for the default
   branch, the one `origin/HEAD` points at, else `main`, `master`, `trunk` or
   `develop`. Five lanes at most.
4. Groups the lanes into stacks, one box per stack, opened with the branch on top
   and closed with its own corner, the way `gg bra` reads them.
5. Badges the uncommitted header with `wip` while HEAD is a wip commit, and `in
   <dir>` when you run it in a subdirectory.
6. Writes nothing: no oplog entry, no footer, no git command of its own. The
   underlying reads are listed under Git equivalent.

## Examples

    gg                 the five newest branches, 6 commits each
    gg -p              print `gg dash`, draw nothing
    gg dash 10         ten commits per lane, spelled out
    gg dash -f         five changed paths under every commit
    gg dash -a         every changed path under every commit
    gg gg -h           this page

## Errors

    gg: no commits yet
        HEAD does not resolve: an empty repository has no tree to draw. Exit 1.

## Git equivalent

None: gg does it in Go. The view composes, among others:

    git status --short --untracked-files=all
    git for-each-ref --sort=-committerdate refs/heads
    git log <base>..<tip>
    git merge-base

## See also

    gg dash      the same view, with the count and the file flags
    gg status    the tree over the worktree lanes
    gg bra       the stacks drawn as stacks
    gg log       the graph without the lanes
    gg help      the table this row is read in
