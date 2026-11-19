# gg sim

Render a git operation before running it. gg runs the `git-sim` binary with
your arguments, so the picture of what a rebase, a merge or a reset would do
comes before anything moves.

## Usage

    gg sim [args...]

## Arguments

    args...   the operation to render, passed to git-sim untouched. With none
              the binary decides what to draw.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg sim show kd` hands git-sim the sha `kd` names. A real ref or an
existing path wins over a changeId, and `--` stops the resolution.

## Flags

git-sim has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to git-sim.

## What it does

1. Finds the `git-sim` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr. The call is `git-sim <args>`, not
   `git sim <args>`: no git alias has to exist for this verb to work.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in the table, so no
   oplog entry and no footer. The render is a drawing, not a change.
4. `-p` prints the command line instead of running it, with gg's own quoting:

        git-sim rebase -i 'HEAD~3'

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there.

## Examples

    gg sim status                  draw the current state
    gg sim rebase -i HEAD~3        what those three commits would replay onto
    gg sim merge side              the merge, before it happens
    gg -p sim rebase -i HEAD~3     print the command, draw nothing
    gg sim -h                      this page, read by gg, not by git-sim

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A git-sim that refuses
an argument prints its own message and its own exit code comes back through gg.

## Git equivalent

None: the call is the external binary, which reads the repository itself.

    git-sim <args>

## See also

    gg tools     every external tool and its install state
    gg rebase    run the replay the render showed
    gg merge     the merge, once you have seen it
    gg log       the graph, without the drawing
