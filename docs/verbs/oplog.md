# gg oplog

List the operations `gg undo` can step back over, newest first: the short sha,
how long ago it ran and the verb that wrote it. `gg oplog clear` drops the log.

## Usage

    gg oplog
    gg oplog clear

## Arguments

    clear   drop the oplog and forget every entry. Any other argument lists.

## Flags

    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

## What it does

1. Walks the chain from `refs/gg/oplog/cur` and prints one line per entry:
   two spaces, the short sha in colour, the age and the verb name.

       1f3c9ab  2m  commit
       b72e1d0  9m  discard

   The age is compact: `12s`, `5m`, `2h`, `3d`, `2w`, `1mo`.
2. An entry is a commit under `refs/gg/oplog` holding the worktree tree, the HEAD
   it stood on, the managed refs (`refs/heads`, `refs/tags`, `refs/gg`) and
   HEAD's symbolic target. Its committer is `gg oplog <gg-oplog@localhost>`, which
   is how an entry is told apart from a real commit.
3. Entries are written before the verb runs, so the newest line names the last
   operation you asked for, and the state it points at is the one before it.
4. Only the newest 100 entries are kept. Once the chain grows past that, the
   older ones are rebuilt away and their objects are left to `git gc`.
5. With no entry at all, it prints `no operations yet` and exits 0. That is also
   what a fresh repository answers.
6. `clear` drops `refs/gg/oplog/tip`, `refs/gg/oplog/cur` and
   `refs/gg/oplog/redo`, then prints
   `gg: oplog cleared; run git gc --prune=now to drop the objects`. The entry
   objects survive until garbage collection.

## Examples

    gg oplog                    what undo can reach, newest first
    gg oplog clear              forget the log
    git gc --prune=now          and free the objects
    gg undo                     after clear: nothing to undo

## Errors

None of its own: listing and clearing both exit 0, and an empty log prints
`no operations yet` instead of failing.

## Git equivalent

None: gg keeps the log itself, in refs of its own, and writes it in Go.

## See also

    gg undo      step back to the previous state
    gg redo      step forward again after an undo
    gg discard   a mutating verb that the log records
