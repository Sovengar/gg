# gg tuicr

Open tuicr, the code review TUI, on the revset you name. gg runs the `tuicr`
binary with your arguments: the review pane, the keys and the flags are tuicr's
own.

## Usage

    gg tuicr [args...]

## Arguments

    args...   what to review: a revset, with `-w` for the working tree. Both are
              tuicr's, not gg's; `gg hh -v` repeats the note the registry
              carries. With none the binary decides what to open.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg tuicr kd` hands tuicr the sha `kd` names. A real ref or an existing
path wins over a changeId, and `--` stops the resolution.

## Flags

tuicr has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other flag, `-w` included, belongs to tuicr.

## What it does

1. Finds the `tuicr` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr, which a full-screen TUI needs.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in the table, so no
   oplog entry, no footer and no snapshot. Whatever tuicr changes in the tree is
   tuicr's to undo, not `gg undo`'s.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        tuicr -w

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there.

## Examples

    gg tuicr                       review what the default revset names
    gg tuicr -w                    review the working tree
    gg tuicr main..side            review that range
    gg -p tuicr -w                 print the command, review nothing
    gg tuicr -h                    this page, read by gg, not by tuicr

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A tuicr that refuses a
revset prints its own message, and its exit code comes back through gg.

## Git equivalent

None: the call is the external binary, which reads the repository itself.

    tuicr <args>

## See also

    gg tools    every external tool and its install state
    gg hunk     the diff reviewer, if tuicr is not your taste
    gg diff     the diff with one changeId per hunk
    gg log      the graph that names the revsets
