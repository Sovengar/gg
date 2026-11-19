# gg split

Cut one commit in two, so the paths you name land in the parent half and the
rest stays in the rewritten commit. Reach for it when one commit mixed two
concerns and the history is still local.

## Usage

    gg split <changeId> [paths...]

## Arguments

    changeId   the commit to cut. A real ref or an existing path is passed
               through untouched, so name a changeId or a sha here.
    paths...   which paths the parent half takes. Without them the split is
               interactive.

## What it does

1. A wip commit is folded first, so uncommitted edits are not left behind by
   the rewrite.
2. Without an argument it stops with `gg split <commit> [paths...]` and prints
   no git call.
3. It needs the experimental `git history` command: git 2.56 or newer. Older
   git fails with `gg: split needs git >= 2.56 (git history split)`.
4. It runs `git history split <sha>`. With paths it appends `--` and then the
   paths, so `gg split @kd src/a.go` reaches
   `git history split <sha> -- src/a.go`. gg inserts that separator itself:
   writing `--` by hand sends a second one.
5. The call is recorded in the oplog and the footer prints the new HEAD as
   changeId, short sha, ahead and behind.

## Examples

    gg split @kd                  split that commit interactively
    gg split kd src/a.go          src/a.go into the parent half
    gg split @kd docs/ internal/  two paths into the parent half
    gg -p split kd src/a.go       print the gg call, run nothing

## Errors

    gg split <commit> [paths...]
        No commit argument: nothing was run.
    gg: split needs git >= 2.56 (git history split)
        The installed git has no history command.

Any other failure comes from git history split itself, in its own words.

## Git equivalent

    git history split <commit>
    git history split <commit> -- <path>

## See also

    gg squash    fold the commit into its parent instead
    gg describe  edit its message instead
    gg discard   drop the commit and its change
    gg commit    save the tree as one commit
