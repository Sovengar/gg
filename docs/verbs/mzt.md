# gg mzt

Run git-machete, the branch layout tool, with your arguments. gg adds nothing:
the command is the binary, so its subcommands and flags are git-machete's own.

## Usage

    gg mzt [args...]

## Arguments

    args...   any git-machete argument, passed through untouched. With none the
              binary decides what to print. git-machete must be installed
              separately; `gg tools` prints whether it is on PATH.

## What it does

1. It runs the external binary `git-machete` with your arguments, with the
   inherited stdin, stdout and stderr.
2. The exit code is the binary's own, so a failing layout command fails gg the
   same way.
3. Nothing is recorded: `mzt` is not a mutating verb, so there is no oplog entry
   and no footer. Whatever git-machete does to your branches is git-machete's
   to undo.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        git-machete <args>

## Examples

    gg mzt                 the binary, with no arguments
    gg mzt status          the branch layout as git-machete draws it
    gg mzt traverse        walk the layout, rebasing as it goes
    gg -p mzt status       print the command, run nothing

## Errors

    None of its own. With the binary missing the call exits 1 and prints
    nothing; `gg tools` names the missing tool before you get there.

## Git equivalent

    None: the call is the external binary.

        git-machete <args>

## See also

    gg tools    every external tool and its install state
    gg br -a    the stacks gg draws without git-machete
    gg help     the compact verb table
