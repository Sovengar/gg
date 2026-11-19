# gg story

Replay the history of the repository as an animation in the terminal. Reach for
it to explain how a branch grew, or to show a review where the diff on screen
matters more than the log lines.

## Usage

    gg story [args...]

## Arguments

    args...   every argument is handed to gitlogue. A token shaped like a
              changeId is resolved before the verb runs, like every verb; the
              rest reaches the binary unchanged.

gitlogue reads the repository itself, so gg resolves nothing else for it and
passes no argument of its own.

## What it does

1. gg finds the gitlogue binary on PATH and runs it with your arguments,
   inheriting stdin, stdout and stderr.
2. gitlogue replays the history it reads: it starts at HEAD unless an argument
   names another commit.
3. gg waits for the exit code and prints nothing else. The verb is not mutating,
   so it records no oplog entry and prints no footer.

The binary must be installed and on PATH. `gg tools` prints its install state,
and MISSING when it is not there. gg prints nothing of its own for a missing
binary: the exec fails and the exit code is 1.

## Examples

    gg story                    play the history of this repository
    gg story --commit HEAD      start from that commit
    gg -p story --commit HEAD   print the command, run nothing
    GG_ECHO=1 gg story          trace the exec on stderr

## Errors

None of its own. A missing or failing gitlogue exits with the exec code, and
gitlogue reports what it cannot replay:

    gg story            exits 1 with no message when gitlogue is not installed
    gg tools            names the binary and prints MISSING instead

## Git equivalent

None: gg does it in Go, by running gitlogue, which reads the repository itself.

## See also

    gg log     the graph, without the animation
    gg tools   external tools and their install state
