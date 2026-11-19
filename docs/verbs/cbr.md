# gg cbr

Create a branch and switch to it in one call. The arguments go to
`git switch -c`, so a start point works the same way.

## Usage

    gg cbr <branch>
    gg cbr <branch> <start-point>

## Aliases

    branch create   the long form: `gg branch create <branch> [start]`

## Arguments

    branch        the new branch. A branch changeId resolves to its branch name
                  first, and `--` stops that resolution.
    start-point   where the branch starts. Omitted, git uses HEAD.

## Flags

    None of its own. Every argument is appended verbatim after `switch -c`, so
    any flag `git switch` accepts passes through.
    -p, --print   print the resolved git command, run nothing

## What it does

1. A wip commit is folded first, so the new branch does not start below the
   edits you have not committed yet.
2. It runs `git switch -c` with your arguments appended verbatim.

        git switch -c <branch> [start-point]

3. The call is recorded in the oplog and the footer prints the new HEAD.
4. `gg branch` itself never creates a branch; a missing name there points here:

        gg branch: no branch or commit named <name>. gg cbr <name> creates it.

## Examples

    gg cbr side                     side from HEAD, then switch to it
    gg cbr fix/parser main          a branch from main
    gg cbr @kd                      a branch from that commit
    gg branch create side           the long form, same call
    gg -p cbr side                  print the git command, run nothing

## Errors

    fatal: a branch named 'side' already exists
        git's own refusal; switch with `gg go side` instead.
    fatal: invalid reference: nope
        The start point resolves to nothing.

## Git equivalent

    git switch -c <branch> [start-point]

## See also

    gg branch   list branches, or switch to one
    gg go       go where a branch lives, worktree included
    gg fork     put the branch in its own directory
    gg discard  remove the branch and its worktree
