# gg commit

Save the changes you name, or every dirty path, as one commit. The index stays
an implementation detail: nothing asks you to stage first.

## Usage

    gg commit [-m <msg>] [flags] [changeId...]
    gg c [-m <msg>] [flags] [changeId...]
    gg ce [-m <msg>] [flags] <changeId...>

## Aliases

    c     the short form, identical in every way
    ce    commit -e: the names are the ones left out

## Arguments

    changeId...   which changes to save. Each one is a dirty path, a commit id,
                  a branch name or a hunk id, resolved from one namespace. A
                  path or a commit contributes the dirty paths it covers.
    (none)        every dirty path, new files included

ChangeIds resolve before the verb runs: a real ref or an existing path wins
over a changeId, and `--` stops the resolution, so `gg commit -- -weird-file`
reaches the path untouched. `path:`, `commit:` and `branch:` force one kind.
`@` is accepted as a prefix and dropped.

## Flags

    -m, --message <msg>   commit message. Repeatable, git style.
    -F, --file <file>     read the message from a file
    -e, --exclude         the changeIds are the ones left out (same as ce)
    --author <who>        override the author of this commit
    --date <when>         override the author date
    -C, --reuse-message <rev>   take that commit's message and authorship
    -c, --reedit-message <rev>  take its message, edit it
    -t, --template <file>       start the message from that file
    -p, --print           print the resolved git command, run nothing
    -n, --dry             same as -p
    -h, --help            this page, even when other arguments are present

Any other flag reaches `git commit` verbatim. `-h` is read by gg first, so it
never reaches git.

## What it does

1. Splits the arguments into flags and names. Flags with a value (`-m`, `-F`,
   `--author`, `--date`, `-C`, `-c`, `-t`) keep the next argument.
2. With `-e`, every dirty path that no name covers is committed instead. A
   directory name covers the paths under it. A name that matches no dirty path
   is an error: `gg ce: <path> is not a dirty path`. Excluding every path is
   also an error: `gg ce: that leaves no dirty path to commit`.
3. With no names it runs `git add -A` and then `git commit`. With names it
   marks untracked ones intent-to-add (`git add -N`) and commits with
   `git commit <flags> -- <paths>`, so the tree and the index stay consistent.
4. HEAD is a wip commit (`gg new`): the commit is finalized in place with
   `git commit --amend`, then the wip marker is dropped. A wip commit needs a
   message, so name no path and pass `-m`.

The commit is recorded in the oplog, so `gg undo` steps back over it. The
footer shows the new HEAD as changeId, short sha, ahead and behind.

## Examples

    gg c -m "fix the parser"            every dirty path, one commit
    gg c -m "partial" src/a.go          only that path
    gg c -m "partial" kd qz             only those two changeIds
    gg ce -m "rest" docs/               everything except docs/
    gg c -m "wip: save" --no-verify     extra git flags pass through
    gg c -p -m "preview"                print the git command, run nothing
    gg c -F .git/COMMIT_EDITMSG         message from a file

## Errors

    gg c: HEAD is a wip commit. Name no path, or run gg uncommit first.
        A wip commit holds the whole tree; naming paths cannot be split.
    gg c: a wip commit needs a message. Use gg c -m "msg", or gg uncommit ...
        Finalizing the wip needs an explicit message.
    gg ce: HEAD is a wip commit. Run gg uncommit first, then gg ce.
        Exclude mode never finalizes a wip commit.
    gg ce: that leaves no dirty path to commit
        The exclusion covered everything.

## Git equivalent

    git add -A && git commit -m "msg"                no names
    git add -N -- <path>; git commit -m "msg" -- <path>   with names
    git commit --amend -m "msg"                      in a wip commit

## See also

    gg new       save now, keep absorbing changes until you commit
    gg amend     fold changes into an existing commit
    gg uncommit  take the commit apart again
    gg discard   drop a change without committing it
    gg status    see what is dirty before you commit
