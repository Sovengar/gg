# gg jj

Run Jujutsu. gg runs the `jj` binary with your arguments, so the verb is a
doorway: every revset, subcommand and flag is Jujutsu's own, and gg keeps its
changeIds readable for the round trip.

## Usage

    gg jj [args...]

## Arguments

    args...   any jj command and its arguments, passed through untouched. With
              none the binary prints its own help.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg jj show kd` hands jj the sha `kd` names. A real ref or an existing path
wins over a changeId, and `--` stops the resolution. A revset is not a changeId,
so `@`, `::@` and `main..side` reach jj as typed.

## Flags

jj has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to jj.

## What it does

1. Finds the `jj` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr, which jj's pager and editor need.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in gg's table, so no gg
   oplog entry, no gg footer and no snapshot. jj keeps its own operation log and
   `jj undo` steps back over a jj command, not `gg undo`.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        jj status

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there. `gg init` and `gg clone` colocate a jj repo
when jj is installed, so the two views usually agree from the first commit.

## Examples

    gg jj                     jj with no argument, its own help
    gg jj status              the working copy state, as jj sees it
    gg jj log -r @            the current change, one line
    gg jj new main            a new change on top of main
    gg -p jj status           print the command, run nothing
    gg jj -h                  this page, read by gg, not by jj

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A jj that fails prints
its own message, and its exit code comes back through gg unchanged.

## Git equivalent

None: the call is the external binary, which drives the colocated git repo
itself.

    jj <args>

## See also

    gg tools     every external tool and its install state
    gg init      git init, then colocate a jj repo on top
    gg clone     the same, on a fresh clone
    gg changeId  the id the two systems share
