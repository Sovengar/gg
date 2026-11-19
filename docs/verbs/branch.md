# gg branch

List the local branches with their changeIds and upstream drift, or switch to
one. Nothing here creates a branch by accident: a name that exists switches, a
name that does not is an error with a direction, and creating is
`gg branch create`.

## Usage

    gg branch [name]
    gg br [name]
    gg branch create <name>
    gg branch new <name>
    gg br -lr [flags]
    gg br -a
    gg br -la [flags]
    gg br -lm [target]

## Aliases

    br    the short form, identical in every way

## Arguments

    name     a branch name or a branch changeId, resolved before the verb runs;
             a real ref wins over a changeId. A token that names neither a
             branch nor a commit is refused instead of branching off HEAD.
    target   with -lm: the branch or commit to test against, HEAD by default

## Flags

    (none)          every local branch, newest commit first
    -c, --create <name>   create the branch and switch to it
    -lr, --remotes        git branch -r, extra arguments pass on
    -a, --stacks, --stack the stacks: every branch under the one it was cut
                          from, the same list as gg bra
    -la, --all            git branch -a, extra arguments pass on
    -lm, --merged [target]  branches already merged into the target
    -d, -D, --delete      refused: deleting is gg discard's job
    -g, --global          refused: it now means -la
    -r                    refused: it now means -lr
    -h, --help            this page, even when other arguments are present

Only the first argument selects a mode. Any other flag reaches `git branch`
verbatim, so `gg br -v` is `git branch -v` and `gg br --merged main` is git's
own merged list.

## What it does

1. A bare call lists every branch from
   `git for-each-ref --sort=-committerdate refs/heads`: a star on the current
   branch, its branch changeId, the name, then `↑n ↓m` against its upstream or
   `no upstream`.
2. `-lm` asks `git branch --merged <target> --format=%(refname:short)` and draws
   only those rows, newest first, under a `merged into <target>:` line.
3. `-a` draws the stacks, `-lr` and `-la` go straight to `git branch`, and the
   other flags go to `git branch` with no gg decoration at all.
4. A name that resolves switches with `git switch`; `create`/`new`/`-c` builds
   it with `git switch -c`.
5. Before a switch or a create, an active wip commit absorbs the dirty tree, so
   the switch carries no uncommitted edits with it.

Listing only reads: no oplog entry, no footer.

## Examples

    gg br                   every local branch, newest first
    gg branch               the same, spelled out
    gg br main              switch to main
    gg br 4d                switch to the branch that changeId names
    gg br create side       create side from HEAD and switch to it
    gg cbr side             the same, as its own verb
    gg br -lr               remote branches, straight from git
    gg br -la               local and remote, straight from git
    gg br -a                the stacks
    gg br -lm               branches already merged into HEAD
    gg br -lm main          branches merged into main
    gg br -v                git branch -v, gg adds nothing

## Errors

    gg branch: no branches yet
        The repository has no local branch; exits 1.
    gg branch: no branch or commit named <name>. gg cbr <name> creates it.
        A typo never creates a branch; exits 2.
    gg branch create <name>: name the branch to create
        -c, --create, create or new without a name; exits 2.
    gg branch -d is gone. Use gg discard <branch>, with -f to drop unmerged commits.
        Deleting a branch belongs to gg discard.
    gg br -r is now -lr: remote branches
        The old flag is refused with the new spelling.
    gg br -g is now -la: local and remote branches in one list
        Same, for the global list.

## Git equivalent

    git branch                          listing
    git branch -r / git branch -a       the passthrough lists
    git branch --merged <target> --format=%(refname:short)    -lm
    git switch <name>                   switching
    git switch -c <name>                creating

## See also

    gg bra      the branches as stacks
    gg brlm     merged branches, its own verb
    gg brlr     remote branches only
    gg brla     local and remote in one list
    gg cbr      create a branch and switch to it
    gg go       go where a branch lives, worktree and all
    gg discard  delete a branch (its tip stays in refs/gg/deleted)
