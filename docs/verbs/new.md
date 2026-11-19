# gg new

Save the whole working tree as a wip commit on the current branch, then keep
folding later edits into it until you finish it with `gg c -m "msg"` or take it
back with `gg uncommit`. It is the stash gg has: a commit you can see, move
around, and undo.

## Usage

    gg new [-m <msg>] [--force]
    gg new -h

## Arguments

None. `gg new` commits the whole tree, so an argument it does not understand is
an error rather than something to ignore, and a typo never becomes a commit.

## Flags

    -m, --message <msg>   the wip message, "wip" by default. --message=<msg>
                          reads the same way.
    -f, --force           allow folding a wip commit that is already on the
                          upstream
    -p, --print           print the call, run nothing
    -h, --help            this page

## What it does

1. Refuses while git owns the tree: a rebase, a merge, a cherry-pick or a revert
   in progress would be swallowed by a commit.
2. Refuses on an unmerged path, because `git add -A` would commit the conflict
   markers.
3. Refuses when this wip commit is already on the upstream: folding into it
   rewrites a published commit, and `--force` takes that risk on purpose.
4. When HEAD is already the wip commit, folds the uncommitted edits into it
   first, then records the new wip on top, with
   `git add -A` then `git commit --allow-empty --no-verify -m <msg>`.
5. Points a per-worktree marker (`.git/gg/wip`, one per checkout, so two
   worktrees keep two wips) at HEAD.

While HEAD is the wip commit, `gg dash` prints a wip badge and the verbs that
rewind history fold your edits into it first: `gg go`, `gg update`, `gg sync`,
`gg amend`, `gg absorb`, `gg uncommit`, `gg copy` and the editing verbs.
`gg push` refuses until the wip is finished or taken back, and `gg revert`
refuses too. `gg c` with a message finalizes it in place; `gg uncommit` hands
the tree back and drops the marker.

## Examples

    gg new                          park the tree as "wip"
    gg new -m "wip: parser"         give it a message
    gg c -m "feat: parser"          finish it: the wip becomes this commit
    gg uncommit                     take it back, edits in the tree
    gg new --force                  rewrite a wip already on the upstream
    gg -p new -m "wip"              print the call, run nothing

## Errors

    gg new: unknown argument "-x": gg new takes -m <msg> and --force
    gg new: -m needs a message
    gg new: git is in the middle of a rebase. Finish or abort it first.
    gg new: the tree has conflicts. Resolve them first; gg new would commit the markers.
    gg new: this wip commit is already on origin/side. Take it back with gg uncommit first, or pass --force.

## Git equivalent

    git add -A && git commit --allow-empty --no-verify -m "wip"

The wip marker is gg's own bookkeeping under `.git`, with no git equivalent.

## See also

    gg commit     finish the wip with a real message
    gg uncommit   take the wip back into the tree
    gg wt         a parallel branch in its own worktree, each with its own wip
    gg status     the wip badge and the dirty lane
