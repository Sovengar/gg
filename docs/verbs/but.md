# gg but

Run the GitButler CLI. gg runs the `but` binary with your arguments, so every
subcommand, flag and prompt is GitButler's own; the verb exists so a but habit
and a gg habit share one terminal.

## Usage

    gg but [args...]

## Arguments

    args...   any but subcommand and its arguments, passed through untouched.
              With none the binary prints its own help.

A token shaped like a changeId is resolved before the verb runs, like every gg
verb: `gg but show kd` hands but the sha `kd` names. A real ref or an existing
path wins over a changeId, and `--` stops the resolution.

## Flags

but has no flag of its own here: `-p`/`--print` and `-n`/`--dry` are gg-wide
(they print the call and run nothing), `-h`/`--help` shows this page wherever it
sits, and every other argument belongs to but.

## What it does

1. Finds the `but` binary on PATH and runs it with your arguments, with the
   inherited stdin, stdout and stderr, which but's prompts need.
2. Waits for the exit code and returns it unchanged.
3. Prints nothing of its own: the verb is not mutating in gg's table, so no gg
   oplog entry, no gg footer and no snapshot. but keeps its own operation log;
   `gg undo` does not step back over a but command.
4. `-p` prints the command line instead of running it, without a `git ` prefix:

        but status

5. `GG_ECHO=1` traces the same command line on stderr.

The binary must be installed and on PATH; `gg tools` prints its install state
and MISSING when it is not there.

## Examples

    gg but                    but with no argument, its own help
    gg but status             what the workspace holds
    gg but pr new side        publish the stack the way but does
    gg -p but status          print the command, run nothing
    gg but -h                 this page, read by gg, not by but

## Errors

None of its own. With the binary missing the call exits 1 and prints nothing;
`gg tools` names the missing tool before you get there. A but that fails prints
its own message, and its exit code comes back through gg unchanged.

## Git equivalent

None: the call is the external binary, which drives git itself.

    but <args>

## See also

    gg tools     every external tool and its install state
    gg pr        the pull requests, without leaving gg
    gg bra       the stacks, drawn the way but draws them
    gg help      the compact verb table
