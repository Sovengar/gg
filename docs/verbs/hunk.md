# gg hunk

Review a diff in Hunk, the terminal diff reviewer. gg runs the `hunk` binary
with your arguments, so gg stays out of the way: no page of its own, no flags
to learn, just the reviewer.

## Usage

    gg hunk [args...]

## Arguments

    args...   what Hunk should open. A bare revision is one form, and `diff` and
              `show` are the subcommands the table names; the words are Hunk's,
              not gg's. With none the binary decides what to open.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg hunk show kd` hands Hunk the sha `kd` names, which is how a commit
read out of `gg log` reaches the reviewer. A real ref or an existing path wins
over a changeId, and `--` stops the resolution. A `file:hunk` handle is not
resolved here: only `gg diff`, `gg discard` and the path verbs read one.

## Flags

Hunk has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to Hunk.

## What it does

1. Finds the `hunk` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr, which a full-screen reviewer needs.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in the table, so no
   oplog entry, no footer and no snapshot. Whatever Hunk changes in the tree is
   Hunk's to undo, not `gg undo`'s.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        hunk diff

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there.

## Examples

    gg hunk                        open the reviewer with no argument
    gg hunk diff                   review the working tree diff
    gg hunk show kd                review one commit, resolved by gg
    gg hunk -h                     this page, read by gg, not by hunk
    gg -p hunk diff                print the command, review nothing

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A Hunk that fails
prints its own message, and its exit code comes back through gg unchanged.

## Git equivalent

None: the call is the external binary, which reads the repository itself.

    hunk <args>

## See also

    gg tools    every external tool and its install state
    gg diff     the diff with one changeId per hunk, for gg discard
    gg show     one commit, printed in full
    gg tuicr    the other reviewer, if Hunk is not your taste
