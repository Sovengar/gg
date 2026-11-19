# gg brla

List local and remote branches in one list, straight from git. It is
`gg br -la` under its own name: git's columns and git's flags, with none of the
changeIds or drift gg adds to its own branch list.

## Usage

    gg brla [flags]
    gg br -la [flags]

## Arguments

    extra...   every argument reaches `git branch -a` as it is: flags, patterns
               and revisions included. Branch names and shas are refs, so gg
               resolves none of them as a changeId.

## What it does

1. Runs `git branch -a` with every argument appended, so any flag git branch
   knows reaches it unchanged.
2. Prints git's own output and returns git's own exit code. Nothing is redrawn
   and no oplog entry or footer is written.

Local branches come first, then the remote-tracking ones, the way git orders
them. For the local half with changeIds and `↑n ↓m` drift, use `gg branch`.

## Examples

    gg brla                  local and remote branches
    gg br -la                the same, spelled as the flag
    gg brla --merged main    both kinds already merged into main
    gg brla -v               git's own verbose columns
    gg -p brla               print mode: prints `git branch -a`

## Errors

None of its own: a bad flag or a missing ref prints git branch's message and
returns git's exit code.

## Git equivalent

    git branch -a

## See also

    gg branch   the local list, with changeIds and drift
    gg brlr     remote branches only
    gg brlm     branches already merged
    gg bra      the local branches as stacks
