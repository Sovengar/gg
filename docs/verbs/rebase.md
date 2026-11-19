# gg rebase

Replay the current branch onto another base, handing the call to `git rebase`.
Every flag reaches git, so `-i`, `--onto` and `--autosquash` keep working.

## Usage

    gg rebase <base>
    gg rebase -i <base>
    gg rebase --onto <newbase> <upstream>

## Aliases

    rb    the same verb, two letters

## Arguments

    base   anything git rebase accepts as an upstream: a branch, a ref, a sha.
           gg resolves changeIds first: a branch changeId names its branch, a
           commit changeId names its sha.

## What it does

1. A wip commit is folded first, because the rebase rewrites HEAD and the wip
   commit is the branch's tip.
2. It runs `git rebase` with your arguments appended verbatim, nothing added
   and nothing filtered.
3. The call is recorded in the oplog and the footer prints the new HEAD.
4. A conflict stops the rebase; `gg continue` resumes it, `gg abort` drops it,
   and `gg abort -k` keeps the live worktree edits.

## Examples

    gg rb main                   replay this branch on main
    gg rebase main               same, spelled out
    gg rb -i HEAD~3              your own interactive rebase
    gg rb --onto main side       git's own --onto form
    gg rb --autosquash main      fixup commits folded as they replay
    gg -p rb main                print the gg call, run nothing

## Errors

    None of its own: git rebase owns the plan and prints its errors. A stopped
    rebase names the file to fix and points at `git rebase --continue`.

## Git equivalent

    git rebase <base>

## See also

    gg move      replay one commit onto another base
    gg edit      stop a rebase on a commit
    gg continue  resume a stopped rebase
    gg abort     abort the operation in progress
    gg update    fetch and rebase on the upstream
    gg sync      rebase on a branch, then push
