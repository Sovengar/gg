# gg copy

Replay one commit onto a branch, cherry-pick style. With no branch it copies
onto the branch you are on; with a branch it switches there, copies, and comes
back, so you end on the branch you started from.

## Usage

    gg copy <changeId> [branch]
    gg cp <changeId> [branch]

## Aliases

    cp    the short form, identical in every way

## Arguments

    changeId   the commit to copy. Resolved before the verb runs, so a commit
               id, a branch name or a sha all work.
    branch     where to copy it. Must be a local branch. Omitted, the commit is
               copied onto the branch you are on.

## Flags

gg copy has no flag of its own. `-h` prints this page, `-p`/`-n` print the call
instead of running it.

## What it does

1. Refuses with no changeId, or when the first argument is a flag.
2. One argument: `git cherry-pick <sha>` on the current branch.
3. Two arguments: refuses a branch that does not exist and refuses a dirty tree,
   because the switch would carry the edits along. Then it prints
   `gg: copying <sha> onto <branch>`, runs `git switch <branch>`,
   `git cherry-pick <sha>`, and finally `git switch <current>` when the copy
   went through.
4. A cherry-pick that stops on a conflict leaves you on the target branch with
   the conflict, and says which branch it stopped on. Resolve it there, or run
   `gg abort`.

The copy is recorded in the oplog and the footer shows the new HEAD.

## Examples

    gg copy kd             copy that commit onto this branch
    gg copy kd main        copy it onto main, back on your branch after
    gg cp 4d1f0a7 side     the same, shorter, onto side
    gg -p copy kd side     print the plan, copy nothing

## Errors

    gg copy <changeId>: name the commit to copy
    gg copy: no branch named nope
    gg copy: the tree is dirty, commit or discard first
    gg: cherry-pick stopped on main; resolve it, or run gg abort

## Git equivalent

    git cherry-pick <sha>                                    no branch
    git switch <branch>; git cherry-pick <sha>; git switch <previous>   with one

## See also

    gg discard   drop the commit instead of copying it
    gg rebase    replay a whole branch onto another base
    gg merge     bring the branch itself in
    gg abort     drop a cherry-pick that stopped
    gg status    the tree after the copy
