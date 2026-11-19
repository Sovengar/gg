# gg empty

Record a commit that changes nothing. Reach for it to retrigger CI, to mark a
point in history, or to start a branch before the first real change exists.

## Usage

    gg empty [-m <msg>] [flags]

## Arguments

None of its own: every argument is a git commit argument, and it is handed to
`git commit` as it stands.

## Flags

    -m, --message <msg>   commit message, read by git commit
    -h, --help            this page, read by gg first
    -p, --print           print the git command, run nothing
    -n, --dry             same as -p

Every other git commit flag passes through as well. Without a message git opens
its editor.

## What it does

Runs `git commit --allow-empty` with your arguments, and nothing else. The index
is not touched on the way in, so a dirty tree stays dirty and the empty commit
records no content. The commit is recorded in the oplog, so `gg undo` steps back
over it, and the footer shows the new HEAD.

`gg empty -m "msg"` is the usual shape; `-a`, `--no-verify`, `--author` and the
rest behave exactly as they do under git commit.

## Examples

    gg empty -m "ci: retrigger"          an empty commit with that message
    gg empty -m "start the branch"       a first commit with no content
    gg empty -m "wip" --no-verify        the hook does not run
    gg -p empty -m "preview"             print the git command, run nothing

## Errors

None of its own. git commit reports what it refuses: a missing message, or a
hook that exits non-zero.

## Git equivalent

    git commit --allow-empty

## See also

    gg commit   the ordinary commit, with the changes
    gg new      a visible wip commit that holds the whole tree
    gg tag      mark the recorded point
    gg undo     step back over the commit
