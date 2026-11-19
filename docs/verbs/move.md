# gg move

Replay one commit and everything after it onto another base. It is the rebase
for a single commit: the rest of the branch order stays as it was.

## Usage

    gg move <changeId> <onto>

## Aliases

    mv    the same verb, two letters

## Arguments

    changeId   the commit to lift, and with it every commit above it.
    onto       the commit or branch to replay onto.

Both are required and both are resolved first, so `@kd`, `kd` and a sha all
work. Arguments past the second are ignored.

## Flags

    None of its own. The rebase itself runs with --autostash, so uncommitted
    edits survive the rewrite.

## What it does

1. A wip commit is folded first, because the replay rewrites HEAD.
2. Fewer than two arguments stop it: `gg move <commit> <onto>`.
3. It runs exactly one git call:

        git rebase --autostash --onto <onto> <changeId>^

   The `^` keeps the commit itself in the range, so it lands on the new base
   with its children replayed above it.
4. The call is recorded in the oplog and the footer prints the new HEAD. A
   conflict stops the rebase there; `gg continue` resumes it and `gg abort`
   drops it.

## Examples

    gg move @kd main          put that commit, and the ones above it, on main
    gg mv kd @qz              onto another commit, by changeId
    gg move kd origin/main    onto the fetched upstream
    gg -p mv kd main          print the gg call, run nothing

## Errors

    gg move <commit> <onto>
        One argument or none: nothing was run. git's own errors follow a bad
        base or a commit that does not resolve.

## Git equivalent

    git rebase --autostash --onto <onto> <commit>^

## See also

    gg rebase    replay the branch onto a base, with every flag
    gg edit      stop on a commit to amend it
    gg continue  resume a rebase that stopped on a conflict
    gg abort     drop the rebase and go back
    gg squash    fold the commit into its parent instead
