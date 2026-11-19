# gg go

Go where a branch lives: its own worktree when one holds it, and this checkout
when none does. It prints the directory instead of changing the shell, so a
wrapper is needed to actually cd there.

## Usage

    gg go <branch>
    gg go <branch> -- <command> [args...]
    gg go [-- <command>]          ask which worktree to enter

## Flags

    -w, --worktree   put the branch in its own sibling worktree
    -i, --in-place   create and switch in this checkout
    --               stop flag parsing; the rest is the command to run
    -p, --print      print the gg call and change nothing

## What it does

1. It reads the branch, then the flags. An unknown flag is an error and so is a
   second branch name.
2. A worktree that already holds the branch wins: gg prints that path, whatever
   `-w` or `-i` say.
3. A branch that exists with no worktree: a wip commit is folded first, then
   `git switch <branch>`, and the repository root is printed. With `-w` it
   makes the sibling worktree instead and prints that path.
4. A branch that does not exist needs a default branch to start from. gg reads
   `origin/HEAD`, then `main`, `master`, `trunk` and `develop`. With no mode
   flag it asks:

        gg go: <branch> does not exist. Put it in its own worktree? [y/N]:

   `y` creates the sibling worktree, anything else creates the branch here with
   `git switch -c <branch> <base>`. A run with no terminal takes the in-place
   answer without asking.
5. With `-- <command>` it runs that command in the directory, with the inherited
   stdin, stdout and stderr, and returns its exit code instead of printing the
   path.
6. The sibling directory is next to the repository and named after it:
   `<repo>.<branch>`, with every `/` in the branch turned into `-`.

`gg go` is not a mutating verb in the table, so no footer prints. The paths that
create or switch record an oplog entry; walking into an existing worktree
records nothing.

## Examples

    gg go side                    print where side lives
    gg go side -- go test ./...   run the tests there, exit code kept
    gg go side -w                 give that branch its own directory
    gg go feature/new             ask, then create the branch
    cd (gg go side)               fish: cd there without the wrapper
    gg shell install              install the wrapper that cds for you

## Errors

    gg go: unknown flag <flag>
        Only -w, -i and -p are gg's own; `--` ends the flags.
    gg go: name one branch
        Two branch names were given.
    gg go: not inside a work tree
        Run it from a repository.
    gg go: this repository has a single worktree; name a branch
        Nothing to choose from, so the prompt is skipped.
    gg go: name a branch, or one of these worktrees:
        The list of worktrees, printed before the prompt.
    gg go: no terminal to ask on; name the branch
        The prompt needs a terminal; name the branch.
    gg go: <branch> does not exist, and this repository names no default branch to start it from
        No origin/HEAD and no main, master, trunk or develop.

## Git equivalent

    git switch <branch>                   an existing branch here
    git switch -c <branch> <base>         a new branch here
    git worktree add <path> <branch>      an existing branch elsewhere
    git worktree add -b <branch> <path>   a new branch elsewhere

## See also

    gg wt       create, list and remove worktrees
    gg fork     the same creation, under the table's verb
    gg shell    the fish wrapper that cds where go points
    gg branch   list branches, or switch to one
    gg ship     squash a branch and drop its worktree
