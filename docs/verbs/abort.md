# gg abort

Unstick whichever operation owns the tree: rebase, merge, cherry-pick or revert.
With `--keep`, the edits you made while a rebase was stopped survive the abort
in the working tree.

## Usage

    gg abort [--keep]

## Arguments

    (none)   gg finds the stuck operation itself

## Flags

    -k, --keep   stash the live edits, abort, then restore them (a rebase only)
    -p, --print  print the call and run nothing (a gg-wide flag)
    -h, --help   this page

## What it does

1. Looks for the operation marker, in this order, and aborts the first one found,
   saying which: `gg: aborting the rebase`.

       rebase-merge        git rebase --abort
       rebase-apply        git rebase --abort
       MERGE_HEAD          git merge --abort
       CHERRY_PICK_HEAD    git cherry-pick --abort
       REVERT_HEAD         git revert --abort

2. Without `--keep`, on a clean tree, or for a merge, cherry-pick or revert, it
   runs the plain abort and the tree goes back to where the operation started.
3. With `--keep` on a dirty rebase it stashes first
   (`git stash push -u -m "gg abort --keep"`), aborts, then pops. Tracked edits
   and new files both come back. If the pop does not apply it says so and leaves
   the work in the stash.
4. Nothing found: `gg: nothing to abort`, exit 1.
5. Records an oplog entry and prints the footer, like every mutating verb.

The rebase state lives in files git deletes, so `--keep` is the only way to hold
on to work that the abort would otherwise sweep away.

## Examples

    gg abort                    a stopped rebase: back to the branch, tree clean
    gg abort --keep             the same, edits and new files kept
    gg edit kd; echo fix >> a.go
    gg abort -k                 a.go keeps the fix, the branch is back
    gg abort                    a merge left conflict markers behind
    gg abort                    a cherry-pick stopped on a conflict

## Errors

    gg: nothing to abort
        No rebase, merge, cherry-pick or revert is in progress.
    gg: your changes are in the stash (git stash list), the pop needs a manual fix
        The abort ran, but the pop conflicted; the work waits in the stash.

## Git equivalent

    git rebase --abort            rebase
    git merge --abort             merge
    git cherry-pick --abort       cherry-pick
    git revert --abort            revert
    git stash push -u -m "gg abort --keep"; git rebase --abort; git stash pop
                                  with --keep

## See also

    gg continue  resume the stopped operation instead
    gg undo      step back over an operation that already finished
    gg conflict  resolve the conflicted paths
    gg edit      stop a rebase at a commit
