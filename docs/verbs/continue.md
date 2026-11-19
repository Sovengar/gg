# gg continue

Resume the rebase git stopped, and fold the working tree into the commit it
stopped on. It is what you run after resolving a conflict, and after `gg edit`
touched the commit the rebase halted at.

## Usage

    gg continue

## Arguments

    (none)   the rebase plan decides where to go next

Any argument you pass is ignored: the verb always runs `git rebase --continue`.

## Flags

    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

## What it does

1. When the rebase stopped on an edit point, the dirty tree is folded into the
   commit being edited (`git add -A`, then `git commit --amend --no-edit`). An
   edit point is the last command in `.git/rebase-merge/done`: an
   `edit <commit>` line, or a `break` after a merge commit. The fold is skipped
   when the index has unmerged paths or the tree is clean, so a resolved
   conflict is never swallowed by the wrong commit.
2. The wip fold other rewinding verbs do is a no-op here: git owns the tree
   while the rebase is stopped, so nothing is amended behind its back.
3. Runs `git rebase --continue`.
4. Records an oplog entry before it starts and prints the footer after it: the
   new HEAD as changeId, short sha, ahead and behind.

While the rebase is stopped HEAD is detached; the branch comes back when the
rebase finishes, or when `gg abort` gives up on it.

## Examples

    gg edit kd                  the rebase stops on that commit
    (edit the files)
    gg continue                 the edit folds into kd, the rebase finishes
    git merge side              a conflict stops the merge
    gg conflict                 resolve it with the configured tool
    gg continue                 after a cherry-pick or a rebase, same verb
    gg -p continue              print "gg continue" and run nothing

## Errors

gg prints no message of its own here. A rebase that is not in progress, an
unresolved conflict or an empty commit reaches git, and git's text and exit code
are what you see.

## Git equivalent

    git rebase --continue

## See also

    gg edit      stop a rebase at a commit
    gg abort     give up on the stopped operation
    gg conflict  resolve the conflicted paths first
    gg uncommit  a rebase that `gg continue` finishes
