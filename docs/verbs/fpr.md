# gg fpr

Add a review fixup commit on the PR branch instead of rewriting published
history: the pull request then shows one new commit, and the fold happens at
merge time. Reach for it when a reviewer asked for a change on a commit that is
already pushed.

## Usage

    gg fpr <changeId>
    gg pr fixup <changeId>

## Aliases

    fpr    the shorthand for `pr fixup`; both reach this verb

## Arguments

    changeId   the commit the fix belongs to. Resolved before the verb runs, so
               a commit id, a branch name or a plain sha all work.

The commit must be one of the branch's own: inside `git rev-list <base>..HEAD`,
where the base is the pull request base gh reports, or the stack parent gg
derives. It must also already be on `origin/<branch>`; an unpublished commit is
amended instead.

## Flags

gg fpr takes exactly one argument and no flag of its own. `-h` prints this page,
`-p`/`-n` print the call instead of running it.

## What it does

1. Refuses a wip HEAD: finish it with a message first.
2. Refuses a detached HEAD, since there is no PR branch to add to.
3. Reads the base of the pull request with
   `gh pr view <branch> --json baseRefName`, and falls back to the stack parent
   gg derives when gh cannot answer. `origin/<base>` is used when it exists, so a
   stack that already moved still tells which PR owns the commit.
4. Refuses a commit that belongs to a lower branch, and names `gg go <branch>`
   so the fix lands on the pull request that owns it.
5. Marks the whole tree for commit (`git add -A`) and records
   `git commit --fixup=<sha>`.

Nothing is pushed and nothing is rewritten, so the branch keeps its history and
the PR keeps its review. `gg push` publishes the fixup, and a squash merge or
`gg rebase -i --autosquash` folds it into the commit it names. The new commit is
recorded in the oplog and the footer shows the new HEAD.

The `gh` binary must be installed for the base lookup; without it gg falls back
to the derived stack parent. `gg tools` lists only the `tool:` verbs, so `gh`
is not among them.

## Examples

    gg fpr 4d1f0a7             a fixup commit for that commit
    gg pr fixup kd             the same verb, spelled out
    gg fpr 4d1f0a7 -p          print what would run, no rewrite
    gg push                    publish the fixup on the PR branch
    gg rebase -i --autosquash  fold it locally instead of a squash merge

## Errors

    gg pr fixup <commit>
        The verb was called with no argument, or with more than one.
    gg pr fixup: no commit named "qz"
        The changeId resolves to nothing.
    gg pr fixup: HEAD is a wip commit. Run gg c -m <msg> first, then fixup.
    gg pr fixup: detached HEAD, switch to the PR branch first
    gg pr fixup: 4d1f0a7 does not belong to side (the commits above origin/main)
    gg pr fixup: run gg go <branch> for the pull request that owns it
    gg pr fixup: 4d1f0a7 is not published on origin/side
    gg pr fixup: fix an unpublished commit with gg amend 4d1f0a7

## Git equivalent

    git add -A && git commit --fixup=<sha>

## See also

    gg pr        open the pull requests for this branch and its stack
    gg amend     the unpublished path: rewrite the commit itself
    gg push      publish the fixup commit
    gg rebase    -i --autosquash folds fixups into their target
    gg go        walk to the branch that owns a commit
