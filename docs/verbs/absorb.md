# gg absorb

Fold the changed lines into the commits that introduced them, using git-absorb.
It is `git absorb` with the wip fold and the oplog entry that every mutating gg
verb gets.

## Usage

    gg absorb [args...]

## Aliases

    ab    the short form, identical in every way

## Arguments

    args...   every argument is handed to git absorb

git-absorb decides what to fold from the staged hunks it finds; gg adds no
argument of its own. A token shaped like a changeId is resolved before the verb
runs, the way it is for every verb.

## Flags

gg absorb has no flag of its own. git-absorb's flags pass through untouched, so
whatever the installed version accepts works here. `-h` prints this page and
never reaches the binary; `-p`/`-n` print the call instead of running it.

## What it does

1. Folds uncommitted edits into an active wip commit first, the way every
   history-moving verb does, so absorb never sees a wip as its starting point.
2. Runs `git absorb` with your arguments.
3. Records an oplog entry and prints the footer, so `gg undo` can step back over
   the rewrite.

Git finds `git absorb` only when git-absorb is installed and on PATH; it is a
git subcommand, so the binary is `git-absorb`. `gg tools` lists the `tool:`
verbs only, so absorb is not among them: check the install with
`git absorb --help`.

## Examples

    gg absorb          fold the staged hunks into the right commits
    gg ab              the same, shorter
    gg -p absorb       print the git call, run nothing
    GG_ECHO=1 gg absorb   trace the call on stderr

## Errors

None of its own. git reports a missing subcommand when git-absorb is not
installed, and git-absorb reports a hunk it cannot place or a tree it cannot
rewrite. `-h` is the one flag gg answers itself.

## Git equivalent

    git absorb

## See also

    gg amend    fold everything into one commit instead
    gg new      park the tree before absorbing
    gg tools    the external tools gg knows about
    gg undo     step back over the rewrite
