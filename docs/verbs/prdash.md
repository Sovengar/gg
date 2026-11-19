# gg prdash

Open prdash, the dashboard over the open pull requests. gg adds nothing: the
command is the binary, so every flag and every view is prdash's own.

## Usage

    gg prdash [args...]

## Arguments

    args...   any prdash argument, passed through untouched. With none the
              binary decides what to show.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg prdash kd` hands prdash the sha `kd` names. A real ref or an existing
path wins over a changeId, and `--` stops the resolution. prdash reads GitHub
itself, so nothing else is resolved for it.

## Flags

prdash has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to prdash.

## What it does

1. Finds the `prdash` binary on PATH and runs it with your arguments, inheriting
   stdin, stdout and stderr.
2. Waits for it and returns its exit code unchanged, so a failing prdash fails
   gg the same way.
3. Prints nothing of its own: the verb is not mutating in the table, so it
   records no oplog entry and prints no footer.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        prdash <args>

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH. `gg tools` prints its install state,
and MISSING when it is not there.

## Examples

    gg prdash                  the dashboard, no arguments
    gg prdash -h               this page, read by gg, not by prdash
    gg -p prdash               print the command, run nothing
    GG_ECHO=1 gg prdash        trace the exec on stderr

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there.

## Git equivalent

None: the call is the external binary, which reads the pull requests itself.

    prdash <args>

## See also

    gg tools     every external tool and its install state
    gg pr        open pull requests for the stack
    gg rpr       mount one pull request to review it
    gg epr       mount it again with the diff live
