# gg revert

Undo one commit with a new commit on top. It never rewrites history, so it stays
safe on a published branch, where `gg discard` and `gg amend` would move commits
that other people already have.

## Usage

    gg revert [flags] <changeId>
    gg rv [flags] <changeId>

## Aliases

    rv    the short form, identical in every way

## Arguments

    changeId   the commit to undo. A commit id, a branch name or a sha, resolved
               before the verb runs.
    args...    everything else is handed to git revert, so -m N picks the
               mainline of a merge.

## Flags

    -m <n>    the mainline of a merge commit, read by git revert itself
    -h        this page, read by gg first
    -p, -n    print the resolved command, run nothing

There is no flag of gg's own here: everything else reaches `git revert`.

## What it does

1. Refuses a wip HEAD before touching anything. The wip holds the whole tree, so
   a revert on top of it would undo the wrong commit.
2. Refuses while git is mid-operation: a rebase, a merge, a cherry-pick or a
   revert has to finish or abort first.
3. Runs `git revert --no-edit <args...>`. A conflict stops there; resolve it and
   run `gg continue`, or run `gg abort` to go back.

The new commit is recorded in the oplog, so `gg undo` steps back over the revert
and restores the content it removed. The footer shows the new HEAD.

## Examples

    gg revert 4d1f0a7          undo that commit, with a new commit
    gg rv kd                   the same, shorter
    gg revert HEAD             undo the newest commit on this branch
    gg revert -m 1 4d1f0a7     keep the first parent of a merge
    gg -p revert HEAD          print the git command, run nothing

## Errors

    gg revert: name a commit.
        The arguments were empty, so there is nothing to undo.
    gg: HEAD is a wip commit. Run gg c -m "msg" to finish it, or gg uncommit to drop it.
    gg revert: git is mid-operation. Run gg continue or gg abort first.

## Git equivalent

    git revert --no-edit <commit>

## See also

    gg discard   drop a commit and replay the rest instead
    gg undo      step back over your own last operation
    gg continue  resume a revert that stopped on a conflict
    gg abort     throw the stuck operation away
    gg amend     the opposite: fold changes into a commit
