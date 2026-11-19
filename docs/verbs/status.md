# gg status

Draw the workspace as one tree: every dirty path with the changeId a verb
takes, a lane per branch worktree with its commits, the commit the branches
part from, and two hints. Reach for it before a commit or a rollback, when you
want the whole picture on one screen.

## Usage

    gg status [count] [flags]
    gg st [count] [flags]
    gg status -f [count]
    gg status -a [count]

## Aliases

    st    the short form, identical in every way

## Arguments

    count         how many commits each lane lists, 6 by default. Any other
                  argument is ignored.

## Flags

    -f, --files   list changed paths under every commit, five at most, and close
                  the list with a `+N more` tail for the rest
    -a, --all     list every changed path under every commit
    -h, --help    this page, even when other arguments are present

## What it does

1. Runs `git status --short --untracked-files=all` and prints one row per dirty
   path: its file changeId, one status letter and the path. `A` (untracked or
   added) is green, `D` is red, everything else cyan. The section opens with
   `╭┄ zz [uncommitted]`, gains a `wip` badge while HEAD is a wip
   commit, and `in <dir>` when you run it in a subdirectory; paths outside that
   directory are then left out, and the lane says `(no changes here)` when
   nothing is dirty in scope.
2. Picks the lanes: the current branch first, then every branch a sibling
   worktree holds, newest commit first, four at most. gitbutler's own refs are
   never lanes.
3. Fills each lane from `git log <base>..<tip>` with the same row the log
   prints: ball, commit changeId, short sha, age, subject, decorations, author.
   A lane with nothing above the base is dropped here; the dashboard keeps it.
4. Prints the common base under the lanes as
   `┴ <short> (common base) <date> <subject>`. The base is the merge base with
   `gitbutler/target`, else the default branch, else the upstream, else the root
   commit, so the tree always closes.
5. Closes with two hint lines: the first says the first token on each row is the
   changeId, the second points at `gg diff` and `gg c -m "message"`.

When the repository has more than one worktree and Worktrunk's `wt` binary is on
PATH, a lane header also carries `↑n ↓m` against the main worktree and `⚑` when
that other worktree is dirty. Without `wt` the headers stay bare.

The view only reads: no oplog entry, no footer, nothing written.

## Examples

    gg st                  the tree, 6 commits per lane
    gg st 10               ten commits per lane
    gg st -f               five changed paths under every commit
    gg st -a               every changed path under every commit
    gg stf                 the same as gg st -f
    gg sta                 the same as gg st -a

A row looks like this, and the token in front is what other verbs take:

    ╭┄ zz [uncommitted]
    ┊   kd M docs/README.md
    ┊   px A notes.md
    ┊
    ┊╭┄ n6 [trunk]
    ┊● n6 66533a6 2m feat: session helper (HEAD -> trunk) [Sovengar] (N)
    ├╯
    ┊
    ┴ 0a1b2c3 (common base) 2026-11-01 Initial commit

## Errors

    gg: no commits yet
        HEAD does not resolve, so there is no tree to draw; exits 1.

## Git equivalent

None: gg does it in Go. It composes
`git status --short --untracked-files=all`,
`git for-each-ref --sort=-committerdate refs/heads`, `git log <base>..<tip>` and
`git merge-base` itself.

## See also

    gg stf      the same tree with five changed paths per commit
    gg sta      the same tree with every changed path per commit
    gg dash     the same tree over the newest branches, the default view
    gg log      the graph without the lanes
    gg diff     the uncommitted changes, box by box
    gg commit   save the change a changeId names
