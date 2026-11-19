# gg uncommit

Move a commit back into the working tree: the change stays, the commit goes, and
its descendants are replayed on top. With no argument it uncommits HEAD, which
is also how a wip commit is taken back.

## Usage

    gg uncommit [changeId...]
    gg uc [changeId...]

## Aliases

    uc    the short form, identical in every way

## Arguments

    changeId...   the commits to take apart, handled in the order given. A
                  commit id, a branch name or a sha, resolved before the verb
                  runs.
    (none)        HEAD

A merge commit is refused: uncommit one side instead. A commit outside the
history of HEAD is refused too, and so is a commit that carries no change.

## Flags

gg uncommit has no flag of its own; `-h` prints this page, `-p`/`-n` print the
call instead of running it.

## What it does

1. HEAD: `git reset --quiet HEAD~1`. The change lands in the tree, unstaged:
   gg has no staging step, so nothing arrives in the index. A new file comes
   back untracked.
2. Any other commit: the history is replayed without it, its patch is applied
   back with a three-way merge, and the index is cleared again:
   `git rebase --onto <sha>^ <sha>`, `git apply --3way <patch>`,
   `git reset --quiet`. The rest of the tree must be clean before this runs.
3. A rebase that stops on a conflict keeps the dropped change in
   `.git/gg-uncommit.patch` and says where, so the work survives the resolution.
4. Inside a wip commit the edits are folded into the wip first, so `gg uncommit`
   takes the whole tree back and drops the marker.

Every uncommit is recorded in the oplog, so `gg undo` puts the commit back. The
footer shows the new HEAD as changeId, short sha, ahead and behind.

## Examples

    gg uncommit                take HEAD apart, the change back in the tree
    gg uncommit 4d1f0a7        that commit, children replayed on its parent
    gg uc kd                   the same, shorter
    gg uncommit kd qz          two commits, in that order
    gg uncommit                inside a wip commit: the tree comes back, marker gone

## Errors

    gg uncommit: HEAD has no parent to uncommit into
        HEAD is the root commit, so there is no parent to move back onto.
    gg uncommit: qz is not a commit any more; rerun with a fresh changeId
    gg uncommit: side is a merge, uncommit one side instead
    gg uncommit: 4d1f0a7 is not in the history of HEAD
    gg uncommit: the tree is dirty, commit or discard first
    gg uncommit: 4d1f0a7 carries no change to move back
    gg uncommit: the change of 4d1f0a7 did not apply cleanly; fix it by hand
    gg uncommit: the change is kept in .git/gg-uncommit.patch; apply it after the rebase

## Git equivalent

    git reset --quiet HEAD~1                      no argument, HEAD
    git rebase --onto <sha>^ <sha>                any other commit, then
    git apply --3way <patch> && git reset --quiet  the patch back in the tree

## See also

    gg new       park the tree in a wip commit instead
    gg commit    save the changes as a new commit
    gg amend     fold changes into the other direction
    gg discard   drop the change instead of keeping it
    gg undo      step back over the uncommit
