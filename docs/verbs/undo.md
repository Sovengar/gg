# gg undo

Step the worktree and HEAD back to the state before an operation. Every
mutating verb snapshots the tree first, so one `gg undo` rolls back a commit, a
discard, a branch creation or a rebase alike.

## Usage

    gg undo [count] [--force]

## Arguments

    count   how many operations to step back. A positive number, 1 by default.

## Flags

    -f, --force   undo even when it removes commits the upstream already has
    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

## What it does

1. Reads the count and `--force`. Any other token that is not a positive number
   fails with `gg undo: the count must be a positive number`.
2. Walks the oplog chain back one entry per operation. The chain starts at
   `refs/gg/oplog/cur`, so the count is measured in operations, not in commits:
   one rebase is one step, however many commits it replayed.
3. When the worktree differs from the newest entry, that live state is recorded
   as an entry of its own before stepping back. The edits are not lost: `gg redo`
   brings them back.
4. Restores what the entry kept: HEAD, the branch that is checked out, every
   `refs/heads`, `refs/tags` and `refs/gg` ref, then the snapshot tree. The tree
   arrives unstaged, so a restored edit shows as ` M tracked.txt` and a restored
   new file as `?? new.txt`. gg never stages for you.
5. Refuses when the restored state would not contain the upstream tip, unless
   `--force` is passed: the undo would remove commits that are already published.
6. Prints no footer. The only entry it writes is the live state kept by step 3;
   undo takes no step of its own in the log.

## Examples

    gg undo                     step one operation back
    gg undo 3                   three operations back, in one restore
    gg c -m "oops"              committed the wrong thing
    gg undo                     the commit is gone, its paths are dirty again
    gg discard side             deleted a branch
    gg undo                     the branch and its ref are back
    gg undo --force             step over a commit the upstream already holds

## Errors

    gg undo: the count must be a positive number
        The count was not a number, or lower than 1.
    gg: nothing to undo
        The oplog holds no entry yet: no mutating verb has run here.
    gg: cannot undo 4
        The count reaches past the oldest entry in the chain.
    gg: undo would remove commits published on main; pass --force to undo anyway
        The state to restore predates the upstream tip.

## Git equivalent

None: gg does it in Go. The restore itself shells out to `git read-tree`,
`git reset --hard` and `git update-ref`, but no single git command undoes a
rebase, a discard and a branch deletion the same way.

## See also

    gg redo      step forward again after an undo
    gg oplog     the operations undo can reach
    gg uncommit  take one commit apart without restoring the whole state
    gg discard   drop a change, undoable with gg undo
