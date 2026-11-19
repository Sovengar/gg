# gg sta

Draw the status tree with every changed path under every commit: it is
`gg status -a` under its own name. Reach for it when the five-path cap of
`gg stf` hides the file you are looking for.

The body of the view is `gg status`: the uncommitted lane, one lane per branch
worktree, the common base and the two hints. The only difference is the file
list under each commit.

## Usage

    gg sta [count]
    gg st -a [count]
    gg status -a [count]

## Arguments

    count   how many commits each lane lists, 6 by default. Any other argument
            is ignored.

## Flags

    -h, --help   this page, even when other arguments are present

`-a` is already on, so passing it again changes nothing; `gg stf` is the capped
form of the same tree.

## What it does

1. Draws exactly what `gg status` draws, lanes and base included.
2. Under every commit it lists every path that commit touches, from
   `git show --name-status --no-renames --format= <sha>`, in the order git
   prints them, with the same status letter colours as the uncommitted lane.
3. Never prints a `+N more` tail: nothing is capped, so nothing is hidden.

The view only reads: no oplog entry, no footer, nothing written.

## Examples

    gg sta              every changed path, 6 commits per lane
    gg sta 20           the same, twenty commits per lane
    gg st -a            the flag spelling
    gg -p sta           print mode: prints `gg sta`, runs nothing

A commit that lands seven files lists all seven:

    ┊● k3 41d0c11 3h feat: split the parser [Sovengar]
    ┊   A internal/parse/token.go
    ┊   A internal/parse/scan.go
    ┊   M internal/parse/parse.go
    ┊   D internal/parse/old.go
    ┊   A internal/parse/token_test.go
    ┊   M internal/parse/parse_test.go
    ┊   M internal/parse/README.md

## Errors

    gg: no commits yet
        HEAD does not resolve, so there is no tree to draw; exits 1.

## Git equivalent

None: gg does it in Go. It composes
`git status --short --untracked-files=all`,
`git for-each-ref --sort=-committerdate refs/heads`, `git show --name-status`
and `git merge-base` itself.

## See also

    gg status   the default view, no file list
    gg stf      five changed paths per commit, with a +N more tail
    gg dash     the same tree over the newest branches
    gg diff     the uncommitted changes, box by box
