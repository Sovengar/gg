# gg lazygit

Open lazygit, the terminal git UI. gg runs the `lazygit` binary with your
arguments: the panes, the keys and the flags are lazygit's own, and gg adds
nothing to them.

## Usage

    gg lazygit [args...]

## Arguments

    args...   git arguments for lazygit, as the table notes: it takes the words
              a git command would, and picks the pane they address. With none
              it opens on the status panel.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg lazygit kd` hands lazygit the sha `kd` names. A real ref or an existing
path wins over a changeId, and `--` stops the resolution.

## Flags

lazygit has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to lazygit.

## What it does

1. Finds the `lazygit` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr, and with the terminal it needs to draw.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in gg's table, so no gg
   oplog entry, no footer and no snapshot. Whatever lazygit changes in the tree
   or in the history is lazygit's to undo, not `gg undo`'s.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        lazygit log

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there.

## Examples

    gg lazygit                the status panel
    gg lazygit log            the log panel, straight away
    gg lazygit branch         the branch panel
    gg -p lazygit log         print the command, open nothing
    gg lazygit -h             this page, read by gg, not by lazygit

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A lazygit that fails
prints its own message, and its exit code comes back through gg unchanged.

## Git equivalent

None: the call is the external binary, which drives git itself.

    lazygit <args>

## See also

    gg tools     every external tool and its install state
    gg status    the tree lazygit's status panel mirrors
    gg log       the graph, without leaving the terminal
    gg gitdash   the other dashboard over the workspace
