# gg discard

Drop whatever a changeId names: the edits of a path, one hunk, a commit, a
branch and the worktree that holds it, or every uncommitted change at once. It
is the verb that removes work, and every kind it drops is recorded in the oplog,
so `gg undo` brings it back.

## Usage

    gg discard [flags] [changeId...]
    gg d [flags] [changeId...]
    gg del [flags] [changeId...]
    gg discard [flags] -- <path...>

## Aliases

    d, del    the same verb, shorter

## Arguments

    changeId...   a dirty path, a commit, a branch, or a hunk handle. The
                  resolver types each one before the verb runs.
    zz            every uncommitted edit, and the new files too
    (none)        every edit is dropped. A new file git never saw stays.

A plain name is sorted by what it is: an existing path first, then a branch,
then a commit. Only a sha counts as a commit here, so a typo never drops one,
and HEAD or a tag is never a changeId. A hunk handle is the file changeId, a
colon, then the hunk id, exactly as `gg diff --ids` prints it: `kdm:6a`. The
prefixes `path:`, `commit:` and `branch:` force one kind, `--` stops the
resolution, and `@` is accepted and dropped.

## Flags

    -f, --force    drop what the tree or the history would refuse: unmerged
                   commits, a dirty worktree
    -y, --yes      answer every question, and never wait on stdin
    -r, --remote   also delete the upstream copy of a deleted branch
    -h, --help     this page
    -p, --print    print the resolved command, run nothing

`-a`/`--all` is refused on purpose: a bare discard already drops every edit.

## What it does

1. A path restores from HEAD, worktree and index together, and the rest of the
   tree is left alone. A new path has no change to drop, so a discard that names
   only new paths refuses and touches nothing.
2. A hunk reverses one region of one path. A handle from an older diff names
   nothing, and nothing changes.
3. A commit is dropped and its descendants are replayed on its parent. The root
   commit cannot be dropped.
4. A branch goes with its worktree when one holds it. The tip stays in
   `refs/gg/deleted/<branch>`, so its changeId still resolves. After a delete the
   five newest branches print, the default branch among them.
5. `zz` resets the tree, then deletes the new files. It asks first, listing
   them, because no commit can restore them; `-f` skips the question.
6. A bare discard is the reset alone. A new file git never saw stays; one that
   was already staged as an addition goes with the reset, without a question.

A branch that tracks an upstream asks about it, naming the remote it actually
tracks: `-r` deletes it, `-y` answers no for you, and a branch without an
upstream never asks. The current branch, the main worktree and unmerged commits
are refused, and `-f` is what overrides the last two. `gg clear` is gone: its
work is `gg discard zz`.

## Examples

    gg discard                 drop every edit; an unseen new file stays
    gg discard zz              drop every edit and delete the new files
    gg discard zz -f           the same, without the question
    gg discard kd              drop the edits of that path
    gg discard kdm:6a          drop one hunk of docs/README.md
    gg discard 4d1f0a7         drop that commit, replay its children
    gg discard side            remove the branch and the worktree holding it
    gg discard side -f -y -r   drop unmerged commits, no questions, and origin
    gg discard -- -weird-file  a path that starts with a dash

## Errors

    gg discard: no -a. A bare discard drops every edit already.
    gg discard: unknown flag -x
    gg discard: 1a2b3c4 is the root commit; nothing to replay it onto
    gg discard: kd:zz is not a hunk handle
    gg discard: no hunk zz in docs/README.md. Run gg diff for the live handles.
    gg discard: the hunk no longer applies to docs/README.md
    gg discard: 1 path(s) are new, so no change is there to drop:
        the new paths are listed under it, then the same refusal closes with
        gg discard: nothing touched. gg discard zz deletes new files.
    gg discard zz: 2 new path(s) would be deleted, and git cannot restore them:
    gg discard zz: nothing deleted. Pass -f to drop new paths.
    gg discard: side is the current branch; switch away first
    gg discard: side holds commits that are not merged. -f drops them.
    gg discard: side has 2 dirty file(s). Use -f to drop them.
    gg discard: refusing non-interactively; add -y to remove it
    gg discard: no worktree and no branch named nope

## Git equivalent

    git reset --hard --quiet                    no names
    git restore --source=HEAD --staged --worktree -- <path>    a path
    git apply -R --whitespace=nowarn -          one hunk, from its patch
    git history drop <sha>                      a commit, git >= 2.56
    git rebase --autostash --onto <sha>^ <sha>  the same without git history
    git branch -d <branch>                      merged branch, -D with -f
    git worktree remove [--force] <path>        the worktree holding it
    git push <remote> --delete <branch>         the upstream copy, with -r
    git reset --hard --quiet && git clean -f -d    zz

## See also

    gg uncommit  move a commit back into the tree instead
    gg revert    undo a commit without rewriting it
    gg diff      the live hunk handles
    gg undo      step back over a discard
    gg branch    what is left after a delete
