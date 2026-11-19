# gg wt

One worktree per branch: create one, list them with their path and how dirty
they are, remove one. It is the namespace behind `gg fork` and the sibling
directories `gg go` walks into.

## Usage

    gg wt new <branch> [base]
    gg wt ls
    gg wt rm <name|path> [--force]

## Aliases

    ls, list      list the worktrees
    rm, remove    remove one
    -f            the short form of --force

## Arguments

    branch   the branch the new worktree holds. Created from base when it does
             not exist yet; when it does, it is checked out as it is.
    base     where a new branch starts. HEAD by default.
    name     a branch name, an absolute worktree path, or the directory's base
             name. A relative path does not match.
    (extra)  arguments past these are ignored.

## Flags

    --force, -f   remove a worktree that has uncommitted changes

## What it does

1. `new` folds a wip commit first, then creates the sibling directory
   `<repo>.<branch>` (every `/` becomes `-`) and prints its path:

        git worktree add -b <branch> <path> <base>     branch is new
        git worktree add <path> <branch>              branch exists

   An existing branch is refused when another worktree already holds it.
2. `ls` prints one line per worktree: a `*` on the current one, the branch
   name, the path, and `clean` or `N dirty`. It reads only, so it records no
   oplog entry and prints no footer.
3. `rm` removes the directory with `git worktree remove [--force] <path>`. The
   main worktree is refused, and so is a dirty one without `--force`.
4. `new` and `rm` record an oplog entry and print the footer. A removed
   directory is not restored by `gg undo`: refs come back, another directory
   does not.

## Examples

    gg wt new side              a sibling directory on a new branch side
    gg wt new side main         the same, started from main
    gg wt ls                    branch, path and dirt per worktree
    gg wt rm side               remove it when it is clean
    gg wt rm side --force       remove it even with uncommitted changes
    gg wt rm repo.side          by the directory name, or by full path

## Errors

    gg wt new <branch> [base] | gg wt ls | gg wt rm <name|path> [--force]
        No subcommand, or `new` with no branch.
    gg wt new: not inside a work tree
        Run it from a repository.
    gg wt new: <path> already exists
        The sibling directory is taken.
    gg wt new: branch <branch> is checked out in <path>
        Another worktree holds that branch.
    gg wt ls: no worktrees
        git listed none.
    gg wt rm: no worktree named "<name>"
        The name matches no branch, path or directory.
    gg wt rm: <path> is the main worktree. Run git worktree remove from another directory.
        The main worktree is not gg's to remove.
    gg wt rm: <name> has <n> dirty file. Use --force to remove it.
        Uncommitted changes block the removal.

## Git equivalent

    git worktree add -b <branch> <path> <base>
    git worktree list --porcelain
    git worktree remove [--force] <path>

## See also

    gg fork     the same creation, under the table's verb
    gg go       walk to where a branch lives
    gg ship     squash a branch and remove its worktree
    gg discard  drop a branch and the worktree that holds it
