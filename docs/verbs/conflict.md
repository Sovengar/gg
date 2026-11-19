# gg conflict

Resolve conflicted paths with the merge tool git is configured to use. It is
`git mergetool` with no wrapper: same arguments, same prompts, same exit code.

## Usage

    gg conflict [path...]
    gg cf [path...]

## Aliases

    cf    the short form, identical in every way

## Arguments

    path...   the conflicted files to open, one at a time
    (none)    every conflicted path

## Flags

    -p, --print        print the command and run nothing, gg-wide
    -h, --help         this page
    --tool=<tool>      any flag reaches `git mergetool` verbatim

## What it does

1. Hands every argument straight to `git mergetool`: gg parses no flag of its
   own, and `-h` is read by gg first so it never reaches git.
2. git opens each path with the configured tool. The tool comes from
   `merge.tool`, or from `mergetool.<tool>.cmd` for a custom entry; `--tool`
   picks one for a single run.
3. When the tool exits 0, git marks the path resolved, so the next `gg continue`
   or `git commit` sees it clean.
4. Nothing is recorded: `conflict` is not a mutating verb for gg, so there is no
   oplog entry and no footer. A resolution you dislike is undone with git, not
   with `gg undo`.
5. A path that holds no conflict, and a tool that is not installed, answer with
   git's own message.

## Examples

    gg conflict                 every conflicted path, one after another
    gg cf src/main.go           just that file
    gg conflict --tool=vimdiff  pick the tool for this run
    gg -p conflict              print "git mergetool" and run nothing
    gg continue                 when every conflict is resolved

## Errors

gg prints none: git's messages reach the terminal unchanged and the exit code is
git's.

## Git equivalent

    git mergetool
    git mergetool <path>

## See also

    gg continue  resume the rebase or merge once every path is resolved
    gg abort     give up on the operation instead
    gg status    the conflicted paths, listed with the rest of the tree
    gg diff      the diff, conflicts included
