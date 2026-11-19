# gg redo

Step forward again to the state an undo left behind. `gg undo` keeps the entry
it stepped off in `refs/gg/oplog/redo`, and this verb restores it, tree and refs
alike.

## Usage

    gg redo

## Arguments

    (none)        the redo ref holds one state

Anything passed is ignored: the verb always restores the one redo entry.

## Flags

    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

## What it does

1. Reads `refs/gg/oplog/redo`. When there is none, it fails with
   `gg: nothing to redo`.
2. When the worktree changed since the undo, that live state is recorded first
   and the redo target is rebuilt on top of it. A following `gg undo` therefore
   returns your edits, not the old target.
3. Restores HEAD, the checked-out branch, every `refs/heads`, `refs/tags` and
   `refs/gg` ref, then the snapshot tree, unstaged, exactly like `gg undo`.
4. Clears the redo ref: one redo per undo. Any new mutating verb clears it too,
   so work made after an undo drops it and `gg redo` answers
   `gg: nothing to redo`.
5. Prints no footer. When step 2 wrote an entry, that entry holds the restored
   state, so a following `gg undo` lands on the edits the redo kept.

## Examples

    gg undo
    gg redo                     back where you started
    gg undo 3
    gg redo                     the newest of the three steps only
    gg undo
    gg c -m "other work"        a new operation
    gg redo                     nothing to redo: the new work dropped it
    gg undo
    echo fix >> src/a.go        edit the restored tree
    gg redo                     the redo target comes back, the edit is kept
    gg undo                     and the edit returns

## Errors

    gg: nothing to redo
        No undo to step forward over, or a later operation cleared the ref.

## Git equivalent

None: gg does it in Go. The restore uses `git read-tree`, `git reset --hard`
and `git update-ref`, but git keeps no forward pointer an undo could follow.

## See also

    gg undo    step back to the previous state
    gg oplog   the operations undo can reach
