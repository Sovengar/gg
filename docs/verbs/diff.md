# gg diff

Show the uncommitted changes. By default gg hands the work to plain `git diff`,
so the pager you configured (delta, less) keeps its own layout; `--ids` draws
gg's boxes instead, one per hunk, each carrying the changeId that `gg discard`
takes.

## Usage

    gg diff [flags] [revision...]
    gg diff -w
    gg diff --ids
    gg diff <changeId|path>

## Arguments

    revision...   read by git diff as usual. Any non-flag argument counts as a
                  revision, a path included, and a revision reads committed
                  history, so its boxes carry no hunk changeIds.
    changeId      resolved before the verb runs: a file changeId becomes the
                  path it names, so `gg diff kd` is `git diff <path>`

## Flags

    --ids        draw gg's boxes, one changeId per hunk. gg removes the flag
                 before git ever sees it. A pipe ignores it: not a terminal
                 always gets plain git diff.
    -h, --help   this page, even when other arguments are present
    (any other)  reaches `git diff` verbatim: -w, --stat, -U10, -- path

## What it does

1. Takes `--ids` out of the argument list.
2. Without `--ids`, or when stdout is not a terminal, runs plain
   `git diff <args>` and returns its exit code: no parsing, no boxes, your
   pager.
3. With `--ids` on a terminal it runs `git diff --no-color --no-ext-diff
   <args>`, and prints nothing when there is no diff.
4. Draws one box per hunk: a border sized to the title, the title is
   `<file changeId>:<hunk id> <path>`, then a gutter with the old and new line
   numbers, removals red and additions green. A binary file gets one box that
   says `Binary file`.
5. Reads the hunk ids from the working tree in 3-line context regions, which is
   the same context the boxes print, so the title names the region `gg discard`
   will drop. Flags such as `-w` still describe the tree, so the hunk ids stay.
6. Prints the file changeId alone when a non-flag argument is present, because
   a committed region, or a single path, is not the hunk set any verb can act
   on. `gg diff --ids` with no path is the form that names hunks.

## Examples

    gg diff                     plain git diff, your pager draws it
    gg diff -w                  whitespace ignored
    gg diff --stat              git's own summary
    gg diff --ids               boxes, each title a file:hunk changeId
    gg -p diff -w               print mode: prints `gg diff -w`
    gg discard kdm:6a           drop the hunk a box named, keep the rest

With `--ids`, a title is the handle to reuse:

    ─────────────────────╮
    kdm:6a docs/README.md│
    ─────────────────────╯
       12 12│ the line as it was
       12   │-the line it replaces
         12│+the line that replaces it

## Errors

None of its own. Without `--ids` the exit code is git diff's; with `--ids` gg
returns 0 after drawing, and 0 as well when there is nothing to draw.

## Git equivalent

    git diff
    git diff --no-color --no-ext-diff <args>    the read behind --ids

## See also

    gg status   every dirty path with its changeId
    gg discard  drop a path, a hunk, a commit or a branch
    gg commit   save the changes you name
    gg show     one commit in full
