# gg ce

Save every dirty path except the ones you name, as one commit. The excluded
paths stay dirty in the tree. It is `gg commit -e`, with the flag written as its
own verb.

## Usage

    gg ce [-m <msg>] [flags] <changeId...>
    gg c -e [-m <msg>] [flags] <changeId...>
    gg commit -e [-m <msg>] [flags] <changeId...>

## Aliases

    -e, --exclude   the commit flag ce wraps: the names are the ones left out
    ce              its own row in the registry, printed as "commit -e ce"

## Arguments

    changeId...   the dirty paths to leave out. A directory name covers every
                  path under it. Resolved before the verb runs, so a path
                  changeId, a plain path and a `path:` prefix all work.
    (none)        nothing is excluded, so every dirty path is committed

Each name must cover at least one dirty path, and at least one dirty path must
survive the exclusion.

## Flags

    -m, --message <msg>   commit message. Repeatable, git style.
    -F, --file <file>     read the message from a file
    -e, --exclude         accepted here too: it is what gg ce already does
    --author <who>        override the author of this commit
    --date <when>         override the author date
    -C, --reuse-message <rev>   take that commit's message and authorship
    -c, --reedit-message <rev>  take its message, edit it
    -t, --template <file>       start the message from that file
    -p, --print           print the resolved git command, run nothing
    -n, --dry             same as -p
    -h, --help            this page, even when other arguments are present

## What it does

1. Reads the dirty paths from `git status --porcelain --untracked-files=all`, so
   a new file counts like an edited one.
2. Drops every path a name covers. A name that matches no dirty path is an
   error, and so is an exclusion that leaves nothing to commit.
3. Commits what is left the way `gg commit` does with names: untracked paths are
   marked intent-to-add (`git add -N`) and the commit names them explicitly
   (`git commit <flags> -- <paths>`), so the excluded paths stay untouched in the
   tree and the index stays consistent.
4. Refuses outright when HEAD is a wip commit: exclude mode never finalizes a
   wip, because a wip holds the whole tree and cannot be split.

The commit is recorded in the oplog, so `gg undo` steps back over it. The footer
shows the new HEAD as changeId, short sha, ahead and behind.

## Examples

    gg ce -m "all but b" b.txt      every dirty path except b.txt
    gg ce -m "docs later" docs/     everything except the docs tree
    gg ce -m "rest" kd qz           leave out those two changeIds
    gg c -e -m "rest" b.txt         the same verb through the commit row
    gg ce -m "rest" --no-verify     extra git flags pass through
    gg ce -p -m "preview" b.txt     print the git command, run nothing

## Errors

    gg ce: HEAD is a wip commit. Run gg uncommit first, then gg ce.
        Exclude mode never finalizes a wip commit.
    gg ce: zzzz is not a dirty path
        A name covered nothing, so the exclusion is not what you meant.
    gg ce: that leaves no dirty path to commit
        The exclusion covered every dirty path.
    gg ce: cannot read the dirty paths
        git status failed, so gg cannot tell what to leave out.

## Git equivalent

    git add -N -- <kept paths>; git commit -m "msg" -- <kept paths>

There is no git command that commits the complement of a path list, so gg builds
the kept list first and then runs the two calls above.

## See also

    gg commit    the row this verb belongs to
    gg new       park everything in a wip commit instead
    gg discard   drop the excluded paths instead of keeping them dirty
    gg status    see what is dirty before you exclude
