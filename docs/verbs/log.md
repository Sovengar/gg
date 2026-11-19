# gg log

Print the commit graph, one line per commit, with the changeId every other verb
takes, a compact age, the pushed state and the author. It is the graph `gg dash`
draws, without the lanes and without the uncommitted section.

## Usage

    gg log [count] [flags] [git log args...]
    gg l [count] [flags]

## Aliases

    l     the short form, identical in every way

## Arguments

    count   how many commits to print, 6 by default. It must be the first
            argument after the flags; a number that is not positive, and any
            other non-flag argument, stays in the list passed to git log.

## Flags

    -l, --long   print the full 40 character sha instead of the short one
    -h, --help   this page, even when other arguments are present
    (any other)  reaches `git log` verbatim, after gg's own format

## What it does

1. Reads `-l/--long` and the count, then runs
   `git log --graph --decorate=short --format=<record> -n <count>` with the rest
   of the arguments appended.
2. Redraws every line as `<graph> <changeId> <sha> <age> <subject>
   <decorations> {<merge sides>} [<author>] (<signature>)`, so a graph of `*`,
   `|`, `/` and `\` arrives as dots and box characters.
3. Paints the ball bright green when a remote-tracking branch already reaches
   the commit, magenta when it only exists locally. The pushed set comes from
   `git rev-list --remotes --max-count=5000`.
4. Paints the age by recency: `30s`, `5m`, `2h`, `10d`, `3w`, `2mo`, bright
   green under an hour, green under a day, yellow under a week, dim past that.
5. Adds what git already knows: the changeId, a yellow decoration block, both
   sides of a merge as `{8fb1f04 62638fe}`, the author in blue and the `%G?`
   signature letter in red. git's own colours are stripped first, then every
   line is repainted, so the lanes stay one width.
6. On a terminal the output goes through `less -FRSX` when `less` is installed,
   and is streamed when it is not. A pipe always gets `--color=never`, so
   scripts parse the same columns.

The view only reads: no oplog entry, no footer, nothing written.

## Examples

    gg log                  6 commits, newest first
    gg log 20               twenty of them
    gg log -l 40            forty rows with the full sha
    gg l 3                  the alias, three rows
    gg log 5 --author=Sovengar    the extra argument reaches git log
    gg log 10 --since=1.week      so does this one
    gg -p log 3             print mode: prints `gg log 3`, runs nothing

A row reads:

    ● n6 66533a6 2m feat: session helper (HEAD -> trunk) [Sovengar] (N)

## Errors

None of its own: a failing `git log` prints git's message on stderr and exits 1.
`gg log -h` prints this page instead, because gg reads `-h` first.

## Git equivalent

    git log --graph --decorate=short -n 6

## See also

    gg dash     the graph in lanes, over the newest branches
    gg status   the tree with the uncommitted changes
    gg show     one commit in full
    gg diff     the uncommitted changes, box by box
    gg changeId resolve a changeId without running a verb
