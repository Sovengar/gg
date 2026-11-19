# gg land

Land a branch on the target by fast-forwarding the remote target to it and
pushing. No pull request, no review, no CI: it is the merge you make on purpose
when the branch is already reviewed and ready.

## Usage

    gg land [branch] [flags]
    gg land --stack <branch> [flags]

## Arguments

    branch   the branch to land, HEAD's branch by default
    (none)   HEAD's branch

## Flags

    -b, --onto, --base <target>  land on that branch instead of the default one
    -y, --yes    accept the no-review warning instead of answering it
    -s, --stack  land the whole chain above the branch in one push
    -p, --print  print the call and run nothing (a gg-wide flag)
    -h, --help   this page

## What it does

1. Reads one branch, one target, `-y` and `-s`. An unknown flag, a second branch
   or a `-b` with no value is an error.
2. Refuses a wip commit, and refuses when there is no `origin`: the target lives
   on a remote.
3. The target is `-b`, else the repository default: `origin/HEAD` when it is
   set, else a branch named `main`, `master`, `trunk` or `develop`. When the
   repository names none, it stops and asks for `-b`.
4. Refuses a branch that is the target, or does not exist.
5. With `--stack`, walks the single-child chain up from the branch and lands the
   top of it, because a child's history already holds its parent. A fork is
   refused: each line then lands on its own.
6. Runs `git fetch --quiet --prune origin`.
7. Reports a branch already landed and exits 0, including the case where a child
   landed first and carried the base into the target.
8. Refuses a branch that is not strictly ahead of the target: a real merge there
   would land without the review and CI a pull request runs.
9. Refuses a base that still has unlanded children on top: the stack lands leaf
   to root, so the base publishes last.
10. Prints the price of the shortcut, then asks
    `Land <branch> onto origin/<target>? [y/N]: `. `-y` skips the question; a tree
    with nobody to answer refuses rather than assume.
11. Runs `git push origin <sha>:refs/heads/<target>`, a fast-forward on the
    remote, never a merge commit.
12. Moves the local target with it, when that is a fast-forward and no worktree
    has the target checked out (`git update-ref -m "gg land"`), and says which of
    the two happened.
13. Records an oplog entry and prints the footer. The push itself is not
    undoable: `gg undo` restores local refs, not the remote.

## Examples

    gg land                     land HEAD's branch on origin/main
    gg land topic -y            no question, no review, no CI
    gg land -b release topic    land on release instead of main
    gg land --stack base -y     base and the child above it, one push
    git switch -qc base; git switch -qc child
    gg land base -y             refused: child is still stacked on it
    gg land base --stack -y     lands child, which publishes base with it
    gg land main -y             refused: main is the target itself

## Errors

    gg land: -b needs a target branch
    gg land: name one branch to land
    gg land: unknown flag --fast
    gg land: no origin remote to land on
    gg land: detached HEAD, name the branch to land
    gg land: this repository names no default branch; pass -b <target>
    gg land: main is the target branch itself
    gg land: no branch named feat/missing
    gg land: origin has no branch release
    gg land: base forks into api, ui; land each top branch on its own
    gg land: base is already landed on origin/main
    gg land: base is not ahead of origin/main, so it cannot fast-forward.
    gg land: gg pr opens a pull request instead, with review and CI.
    gg land: base has stacked branches on top: child
    gg land: land the children first, leaf to root, so the base publishes last.
    gg land: to drop one instead, gg discard branch:child
    gg land: refusing to land non-interactively; add -y to accept the risk
    gg land: aborted, nothing landed

## Git equivalent

    git push origin <branch>:refs/heads/<target>
    git update-ref -m "gg land" refs/heads/<target> <sha>     the local target

## See also

    gg pr        open a pull request: review and CI before the target moves
    gg ship      squash a branch into one commit on a local target
    gg push      publish the branch itself
    gg br -a     the stack land reads the children from
    gg revert    undo a landed commit with a new one
