# gg gitdash

Open gitdash, the workspace dashboard. gg runs the `gitdash` binary with your
arguments; with `-print` the same dashboard prints its status table instead of
drawing a full screen.

## Usage

    gg gitdash [args...]

## Arguments

    args...   gitdash's own arguments, passed through untouched. `-print` is the
              one the table names: it prints the status table. With none the
              binary opens its dashboard.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb. A real ref or an existing path wins over a changeId, and `--` stops the
resolution.

## Flags

gitdash has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to gitdash.

`-print` is a single word and is not read by gg: only `-p`, `--print`, `-n` and
`--dry` are the gg-wide ones, so `-print` reaches the binary as written.

## What it does

1. Finds the `gitdash` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in gg's table, so no gg
   oplog entry, no footer and no snapshot. Whatever gitdash changes is
   gitdash's to undo, not `gg undo`'s.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        gitdash -print

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there.

## Examples

    gg gitdash                the dashboard
    gg gitdash -print         the status table on stdout, no full screen
    gg -p gitdash -print      print the command, run nothing
    gg gitdash -h             this page, read by gg, not by gitdash

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A gitdash that refuses
an argument prints its own message, and its exit code comes back through gg.

## Git equivalent

None: the call is the external binary, which reads the repository itself.

    gitdash <args>

## See also

    gg tools     every external tool and its install state
    gg           the dashboard gg draws itself
    gg status    the tree, with the changeIds a verb takes
    gg lazygit   the other terminal UI over the same repository
