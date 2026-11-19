# gg reflog

Where HEAD has been: git's reflog, read straight from the repository. Reach for
it when a rewrite or a delete moved a branch and you want the sha it had a
minute ago.

## Usage

    gg reflog
    gg reflog <ref>

## Arguments

    ref   which reflog to read. A branch name shows that branch's log, and
          `HEAD` is the default. Any argument git reflog accepts is passed on.

## What it does

1. It runs `git reflog` with your arguments appended.

        git reflog <args>

2. Nothing is recorded and no footer prints: the verb only reads, so it gets no
   oplog entry.
3. The output is git's own, one line per move: the sha, the action that moved
   the ref, and the message. Copy a sha from there and any gg verb takes it,
   because a plain sha always resolves.

## Examples

    gg reflog              where HEAD has been
    gg reflog side         the same for one branch
    gg reflog show side    the long form of that call
    gg -p reflog           print the git command, run nothing

## Errors

    None of its own: a ref that does not resolve fails in git's words.

## Git equivalent

    git reflog <args>

## See also

    gg undo     step back over an operation, from gg's own oplog
    gg redo     step forward again
    gg oplog    the operations undo can reach
    gg log      the commit graph instead of the trail
