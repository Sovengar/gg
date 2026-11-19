# gg ship

Turn a branch into one commit on the target branch, local only. The commits are
squashed, the target fast-forwards onto the new commit, the branch's worktree
goes away, and nothing is pushed.

## Usage

    gg ship [branch]
    gg ship [branch] -b <target>
    gg ship [branch] -m <message> [-y]

## Arguments

    branch   the branch to ship. Default: the branch you are on. A branch
             changeId resolves to its branch name. It must be a local branch,
             and it cannot be the target.

## Flags

    -b, --onto, --base <target>   the target branch. Default: the repository's
                                  default branch
    -m, --message <msg>           the message of the squashed commit
    -y, --yes                     skip the confirmation
    -p, --print                   print the gg call, run nothing

## What it does

1. The guards come first: no wip commit at HEAD, a branch name even on a
   detached HEAD, a target (default branch or `-b`), a branch that is not the
   target, and both branches local.
2. A branch with no commit the target lacks is refused: its tip is already the
   merge base, so there is nothing to squash.
3. The message: with one commit, its subject is reused. With several, `-m` is
   required.
4. The target's worktree is checked: when another worktree holds the target and
   this one is not on it, gg refuses; when this worktree is on the target and
   dirty, it refuses too. A dirty worktree holding the shipped branch refuses as
   well.
5. Without `-y` it prints what shipping costs and asks
   `Ship <branch> onto <target>? [y/N]:`. A non-interactive run refuses.
6. It builds the squashed tree with the index-free plumbing:

        git merge-tree --write-tree <target-tip> <branch>
        git commit-tree <tree> -p <target-tip> -m <message>

   A conflicting merge tree stops it before any ref moves.
7. It moves the branch ref to that one commit with
   `git update-ref -m "gg ship" refs/heads/<branch> <new> <old>`. When you are
   standing on the branch, it also runs `git reset --hard --quiet <new>` so the
   checkout matches the rewritten commit.
8. It fast-forwards the target: `git merge --ff-only --quiet <new>` when the
   target is checked out here, `git update-ref -m "gg ship"` otherwise.
9. It removes the shipped branch's worktree with
   `git worktree remove --force <path>`, unless that worktree is the current
   directory, in which case it says to run `gg discard <branch>` elsewhere.
10. The branch ref now equals the target: one commit, same sha. No remote is
    touched, so nothing is published, and `gg undo` puts the old refs back. The
    removed directory is not restored, since undo restores refs, not another
    worktree.

## Examples

    gg ship side                       one commit on the default branch
    gg ship side -m "feat: side work"  name the message for several commits
    gg ship side -b release            onto another local target
    gg ship                            the branch you are on, one commit
    gg -p ship side                    print the gg call, run nothing

## Errors

    gg ship: -b needs a target branch
    gg ship: -m needs a message
    gg ship: unknown flag <flag>
    gg ship: name one branch to ship
    gg ship: detached HEAD, name the branch to ship
    gg ship: this repository names no default branch; pass -b <target>
    gg ship: <branch> is the target branch itself
    gg ship: no branch named <branch>
    gg ship: no local branch named <target>
    gg ship: cannot read <ref>
    gg ship: <branch> and <target> share no base
    gg ship: <branch> holds no commit <target> lacks
    gg ship: <branch> holds <n> commits; name the squashed message with -m
    gg ship: cannot read the message of <branch>; pass -m
    gg ship: <path> holds <target>; ship from there, or switch this worktree to <target>
    gg ship: this worktree holds <target> and has uncommitted changes. Commit or discard them first.
    gg ship: <path> has <n> dirty file(s). Commit or discard them first.
    gg ship: refusing non-interactively; add -y to ship
    gg ship: aborted, nothing shipped
    gg ship: <branch> conflicts with <target>. Rebase it first: gg rb <target>
    gg ship: cannot write the squashed commit
    gg ship: <path> stays; run gg discard <branch> from another worktree to remove it
    gg ship: removed <path>
    gg: HEAD is a wip commit. Run gg c -m "msg" to finish it, or gg uncommit to drop it.

## Git equivalent

    None: gg does it in Go. The plumbing it runs:

        git merge-tree --write-tree <target> <branch>
        git commit-tree <tree> -p <target-tip> -m <message>
        git update-ref -m "gg ship" refs/heads/<branch> <new> <old>
        git reset --hard --quiet <new>            when the branch is checked out
        git merge --ff-only --quiet <new>         when the target is checked out
        git worktree remove --force <path>        when the branch has one

## See also

    gg land      fast-forward the target and push, keeping the commits
    gg pr        push the branch and open a pull request
    gg wt        the worktree ship removes
    gg go        walk to the branch directory first
    gg undo      restore the refs after a ship
