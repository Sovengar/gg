# gg brlm

List the local branches git already considers merged into a target, HEAD by
default. Reach for it when a stack has landed and you want to know which
branches are safe to delete.

## Usage

    gg brlm [target]
    gg br -lm [target]

## Arguments

    target   the branch or commit to test against, HEAD by default. Only a first
             argument without a leading dash is read; flags are not passed on.

## What it does

1. Runs `git branch --merged <target> --format=%(refname:short)` to learn which
   branches the target already holds.
2. Keeps those rows from gg's own branch list, so every line carries the branch
   changeId, the name, the upstream drift (`↑n ↓m`, or `no upstream`) and a star
   on the current branch.
3. Prints a dim `merged into <target>:` line and one row per branch, newest
   commit first.

The view only reads: no oplog entry, no footer.

## Examples

    gg brlm               everything merged into HEAD
    gg br -lm             the same, spelled as the flag
    gg brlm main          everything main already holds
    gg br -lm origin/main against the remote-tracking branch
    gg -p brlm            print mode: prints `gg brlm`, runs nothing

A row reads like the plain branch list, so a merged branch is easy to hand to
`gg discard`:

    merged into HEAD:
      n6 main    ↑0 ↓0  default
      k3 side    ↑0 ↓0

## Errors

    gg br -lm: <target> is not a commit
        `git branch --merged` refused the target; exits 1.
    gg br -lm: no branch is merged into <target>
        The target holds none of the local branches. The line is printed as an
        error, but the exit code is 0.

## Git equivalent

    git branch --merged <target>

## See also

    gg branch   the full list and the other modes
    gg bra      the stacks the branches form
    gg brlr     remote branches only
    gg brla     local and remote in one list
    gg discard  delete a branch; its tip stays in refs/gg/deleted
