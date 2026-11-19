# gg fork

Put one branch in a sibling directory and work there, leaving this checkout on
the branch it has. It is the same code as `gg wt new`, under the verb the table
uses.

## Usage

    gg fork <branch>
    gg fork <branch> <base>

## Aliases

    fk    the same verb, two letters
    wt new  the same call under the worktree namespace

## Arguments

    branch   the branch the new directory holds, named as written. Extra
             arguments past the base are ignored.
    base     where a new branch starts. HEAD by default, with a wip commit
             folded in first. Ignored when the branch already exists.

## What it does

1. It folds a wip commit first, so the new directory never starts on top of it.
2. The directory is a sibling of the repository, named `<repo>.<branch>`, with
   every `/` in the branch turned into `-`.
3. An existing branch is checked out as it is; a new one is created from base:

        git worktree add <path> <branch>              branch exists
        git worktree add -b <branch> <path> <base>    branch is new

4. It prints the directory on stderr, records an oplog entry and prints the
   footer.

## Examples

    gg fork side                a sibling directory on a new branch side
    gg fk side main             the same, started from main
    gg fork fix/parser main     the directory is <repo>.fix-parser
    gg wt new side              the same call through wt
    gg go side                  walk to the directory later

## Errors

    gg wt new <branch> [base] | gg wt ls | gg wt rm <name|path> [--force]
        No branch: fork prints the worktree usage line it shares with wt.
    gg fork: not inside a work tree
        Run it from a repository.
    gg fork: <path> already exists
        The sibling directory is taken.
    gg fork: branch <branch> is checked out in <path>
        Another worktree already holds that branch.

## Git equivalent

    git worktree add -b <branch> <path> <base>
    git worktree add <path> <branch>

## See also

    gg wt       create, list and remove worktrees
    gg go       walk to where a branch lives
    gg ship     squash the branch and remove its worktree
    gg discard  drop the branch and the worktree that holds it
