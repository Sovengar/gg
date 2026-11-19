# gg push

Push the current branch, but never lose work silently: a diverged upstream asks
first, and a rewrite gg can prove is yours alone goes out under a lease.

## Usage

    gg push [git push args...]
    gg p [git push args...]

## Aliases

    p    the short form, identical in every way

## Arguments

    git args...   reach `git push` verbatim
    (none)        plain push to the upstream

## Flags

    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

gg reads no push flag of its own; everything else is git's.

## What it does

1. Refuses to publish a wip commit: `gg new` parks the tree in a commit, and a
   push would put that draft on the remote. Finish it with `gg c -m` or take it
   back with `gg uncommit`.
2. With no upstream it prints `gg: no upstream, pushing with -u origin HEAD` and
   runs `git push -u origin HEAD` with your arguments.
3. With an upstream it fetches that remote quietly, then counts both sides:
   `HEAD..upstream` (behind) and `upstream..HEAD` (ahead).
4. Behind zero: `git push` with your arguments. If that fails, it fetches again;
   when the upstream moved meanwhile it says so and asks.
5. Behind more than zero, it first asks whether the force would drop anything but
   your own work: every dropped commit must be authored by `user.email`, and no
   other `refs/remotes` branch may contain it. When both hold, it prints
   `gg: origin/main holds 2 commit(s) HEAD rewrote; retrying with lease.` and
   runs `git push --force-with-lease --force-if-includes`.
6. Otherwise it names the reason, lists the incoming commits, and asks what to
   do: `gg: [u] update and push  [f] force-with-lease  [a] abort > `. With no
   terminal to answer on, it refuses instead of forcing.
7. Records an oplog entry and prints the footer: changeId, short sha, ahead and
   behind after the push.

## Examples

    gg p                        push to @{upstream}
    gg p --tags                 extra git flags pass through
    gg -p push                  print "gg push" and run nothing
    gg p                        after gg amend on a pushed commit: lease push, no question
    gg p                        a foreign commit upstream: asks, or refuses with no tty
    gg p                        a diverged upstream: [u] rebases, [f] forces, [a] stops

## Errors

    gg: HEAD is a wip commit. Run gg c -m "msg" to finish it, or gg uncommit to drop it.
        The wip commit is unpublished work; publish it on purpose first.
    gg: no upstream, pushing with -u origin HEAD
        Informational: the push still runs, and sets the upstream.
    gg: not forcing: it would drop 1a2b3c4 by other@x
        The rewrite would drop someone else's commit.
    gg: not forcing: refs/remotes/origin/side also holds 1a2b3c4
        Another remote branch holds the commit the force would drop.
    gg: origin/main has 2 commit(s) you don't have (you are 1 ahead):
        The upstream moved; up to ten incoming commits follow.
    gg: push rejected, origin/main moved while you were working:
        The push failed and the refetch found new upstream commits.
    gg: refusing to push non-interactively; run 'gg update' first
        Nobody was there to answer the question.
    gg: aborted, nothing pushed

## Git equivalent

    git push
    git push --force-with-lease --force-if-includes    your own rewrite only
    git push -u origin HEAD                            no upstream

## See also

    gg update  fetch and rebase on the upstream
    gg sync    rebase on a named branch, then push
    gg land    fast-forward the remote target, no PR
    gg pr      open pull requests instead of pushing the target
    gg new     why a wip commit blocks the push
