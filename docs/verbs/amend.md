# gg amend

Fold the working tree into HEAD, or into a commit of your choice, without
staging anything first. The message stays as it is.

## Usage

    gg amend
    gg amend <changeId> [args...]

## Arguments

    (none)     fold the tracked changes into HEAD
    changeId   fold the changes into that commit instead, and replay its
               descendants on top. Resolved before the verb runs.

With no argument the message is untouched. With a changeId the commit keeps its
message too, and the fold rewrites that commit and everything above it.

## Flags

    -h, --help   this page, read by gg first
    -p, --print  print the call, run nothing
    -n, --dry    same as -p

Nothing else is gg's: an argument after the changeId reaches
`git history fixup` verbatim.

## What it does

1. No argument: `git commit -a --amend --no-edit`. Tracked changes go in and the
   message stays. A new, untracked file is not added, because `git commit -a`
   only sees paths git already tracks. Use `gg c` when a new file belongs in the
   commit.
2. One argument: needs git 2.56 or newer, then `git history fixup <sha>
   [args...]`. The change lands in that commit and its descendants are replayed.
3. Inside a wip commit the uncommitted edits are folded into the wip first, the
   way every history-moving verb does.

The amend is recorded in the oplog, so `gg undo` restores the old HEAD, and the
footer shows the new one. An amend keeps the `Change-Id` trailer, so the commit
keeps its changeId while the sha moves.

## Examples

    gg amend                       fold the tracked changes into HEAD
    gg amend kd                    fold into that commit
    gg amend 4d1f0a7 --no-verify   extra git history flags pass through
    gg amend                       inside a wip: the wip is the commit amended
    gg -p amend                    print the plan, amend nothing

## Errors

    gg: amending another commit needs git >= 2.56 (git history fixup)
        An argument was given but this git has no history command.

git reports the rest: a clean tree, a stopped rebase, or a commit that is not in
the current history.

## Git equivalent

    git commit -a --amend --no-edit             no argument
    git history fixup <commit>                  with a changeId, git >= 2.56

## See also

    gg commit     save the changes as a new commit
    gg describe   change the message instead of the content
    gg absorb     fold changes into the commits that introduced them
    gg uncommit   move the commit back into the tree
    gg squash     fold one commit into its parent
