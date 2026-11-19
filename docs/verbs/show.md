# gg show

Print one commit in full: git show, with the changeIds of `gg log` and `gg dash`
resolved before git runs. Reach for it when the log row is not enough and you
want the message, the patch and the diffstat git prints by default.

## Usage

    gg show [changeId|rev] [git show args...]
    gg sh [changeId|rev]

## Aliases

    sh    the short form, identical in every way

## Arguments

    changeId   a commit changeId from the tree or the log. It is replaced by the
               sha it names, so git never has to know about changeIds. An `@`
               prefix is accepted and dropped.
    rev        anything else reaches git show untouched: HEAD~2, a tag, a branch
               name, a range. A real ref or an existing path always wins over a
               changeId.

    (none)     git show with no revision prints HEAD, which is also what gg
               does

## Flags

    -h, --help   this page, even when other arguments are present
    (any other)  reaches `git show` verbatim: --stat, --name-only, -- path

## What it does

1. Resolves the arguments before the verb runs, in the shared namespace: a
   changeId becomes its sha, and everything that is already a ref, a path or a
   flag stays as it is. `--` stops the resolution, so `gg show kd -- <path>`
   reaches the path untouched.
2. Runs `git show` with the resolved arguments, so the output, the pager and the
   colour are git's own.
3. Records nothing: show is read only, so there is no oplog entry and no footer.
   git's exit code comes back untouched.

`-h` is read by gg first, so `gg show -h` prints this page and never reaches
`git show --help`.

## Examples

    gg show                HEAD in full
    gg sh kd               the commit that changeId names
    gg show HEAD~2         two commits back
    gg show kd --stat      git's own summary on top
    gg show kd -- docs/README.md    only that path of the commit
    gg -p show kd          print mode: prints `git show <sha>`, runs nothing

## Errors

None of its own: a changeId that matches no commit is left for git, which
reports the bad revision and returns its exit code.

## Git equivalent

    git show [args]

## See also

    gg log      the graph that prints the changeIds
    gg diff     the uncommitted changes, box by box
    gg changeId resolve a changeId without running a verb
    gg revert   undo one commit with a new commit
