# gg sync

Rebase the current branch on a branch taken from origin, then push it. It is the
one step that brings a feature branch up to date with its base and publishes the
result.

## Usage

    gg sync <branch>
    gg s <branch>

## Aliases

    s    the short form, identical in every way

## Arguments

    <branch>   the branch to rebase on, read from origin. Required.

Only the first argument is read; the rest are ignored.

## Flags

    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

## What it does

1. Refuses a wip commit: the same guard `gg push` uses, since the run ends in a
   push.
2. Without a branch it fails with
   `gg sync <branch>: name the branch to rebase against`.
3. When the tree is dirty and HEAD is a wip commit, the edits are folded into it
   first (`git add -A`, then `git commit --amend --no-edit`), unless a stopped
   operation already owns the tree.
4. Runs `git fetch --prune --all`.
5. Runs `git pull --rebase --autostash origin <branch>`. A conflict stops here,
   with the rebase in progress: `gg continue` or `gg abort`.
6. Then calls `gg push` with no arguments, so every push guard applies: the
   lease push for a rewrite that is yours alone, the question for a foreign
   commit, the refusal when nobody can answer, and `-u origin HEAD` when the
   branch has no upstream.
7. Records an oplog entry and prints the footer.

## Examples

    gg sync main                rebase on origin/main, then push
    gg s develop                the same, other base
    gg sync                     refused: the branch is required
    gg sync main                a conflicting rebase: gg continue when resolved
    gg -p sync                  print "gg sync" and run nothing, even with no branch

## Errors

    gg sync <branch>: name the branch to rebase against
        No branch was passed.
    gg: HEAD is a wip commit. Run gg c -m "msg" to finish it, or gg uncommit ...
        The push guard, before anything is fetched.

The push half adds its own messages: no upstream, a moved upstream, a dropped
foreign commit, and `gg: refusing to push non-interactively; run 'gg update'
first`.

## Git equivalent

    git fetch --prune --all && git pull --rebase --autostash origin <branch> && git push

## See also

    gg update  rebase on the upstream, without the push
    gg push    the push half and its guards
    gg abort   give up on a conflicting pull --rebase
    gg pr      publish the branch as a pull request instead
