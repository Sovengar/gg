# gg stf

Draw the status tree with changed paths under every commit, five at most, and a
`+N more` tail for the ones the cap hides: it is `gg status -f` under its own
name. Reach for it when a commit's files matter but the full list would flood
the screen; `gg sta` is the uncapped form.

The body of the view is `gg status`: the uncommitted lane, one lane per branch
worktree, the common base and the two hints.

## Usage

    gg stf [count]
    gg st -f [count]
    gg status -f [count]

## Arguments

    count   how many commits each lane lists, 6 by default. Any other argument
            is ignored.

## Flags

    -a, --all    lift the cap: every changed path, the same as gg sta
    -h, --help   this page, even when other arguments are present

## What it does

1. Draws exactly what `gg status` draws, lanes and base included.
2. Under every commit it lists up to five paths from
   `git show --name-status --no-renames --format= <sha>`, with the same status
   letter colours as the uncommitted lane.
3. When a commit touches more than five paths it closes the list with `+N more`
   in dim grey, where N is the number of paths the cap hides. A commit with five
   paths or fewer gets no tail.

The view only reads: no oplog entry, no footer, nothing written.

## Examples

    gg stf              five paths per commit, then +N more
    gg stf 20           the same, twenty commits per lane
    gg st -f            the flag spelling
    gg stf -a           every path, the same as gg sta

A commit that lands seven files ends like this:

    ┊● k3 41d0c11 3h feat: split the parser [Sovengar]
    ┊   A internal/parse/token.go
    ┊   A internal/parse/scan.go
    ┊   M internal/parse/parse.go
    ┊   D internal/parse/old.go
    ┊   A internal/parse/token_test.go
    ┊   +2 more

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
    gg sta      every changed path per commit, no cap
    gg dash     the same tree over the newest branches
    gg diff     the uncommitted changes, box by box
