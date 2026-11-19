# gg brlr

List the remote-tracking branches straight from git. It is `gg br -lr` under
its own name: git's columns, git's flags, and no changeIds or drift columns that
gg would add.

## Usage

    gg brlr [flags]
    gg br -lr [flags]

## Arguments

    extra...   every argument reaches `git branch -r` as it is: flags, patterns
               and revisions included. Branch names and shas are refs, so gg
               resolves none of them as a changeId.

## What it does

1. Runs `git branch -r` with every argument appended, so any flag git branch
   knows reaches it unchanged.
2. Prints git's own output and returns git's own exit code. Nothing is redrawn,
   nothing is sorted by gg, and no oplog entry or footer is written.

The list is what the last fetch left behind: a branch that only exists on the
server shows up after `gg update` (or `git fetch`) has brought the remote ref
down.

## Examples

    gg brlr                  every remote-tracking branch
    gg br -lr                the same, spelled as the flag
    gg brlr --no-merged main remote branches not merged into main
    gg brlr --contains HEAD  remote branches that already hold HEAD
    gg -p brlr               print mode: prints `git branch -r`

## Errors

None of its own: a bad flag or a missing ref prints git branch's message and
returns git's exit code.

## Git equivalent

    git branch -r

## See also

    gg branch   the local list, with changeIds and drift
    gg brla     local and remote branches in one list
    gg brlm     branches already merged
    gg bra      the local branches as stacks
