# gg update

Fetch every remote with pruning, then rebase the current branch on its upstream
with the dirt autostashed. `--check` fetches, reports what a real run would face
and touches nothing.

## Usage

    gg update [--check] [git args...]
    gg up [--check] [git args...]

## Aliases

    up    the short form, identical in every way

## Arguments

    git args...   reach `git pull --rebase --autostash` verbatim
    (none)        the whole argument list

## Flags

    -c, --check   fetch, then report and stop: no rebase, no oplog entry, no footer
    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

`--check` is read only as the first argument; anywhere else it reaches git.

## What it does

1. Reads `--check` / `-c` from the front and leaves every other argument for
   the pull.
2. When HEAD is a wip commit and the tree is dirty, the edits are folded into it
   first (`git add -A`, then `git commit --amend --no-edit`), because the rebase
   would otherwise replay a stale base. A stopped operation skips the fold: git
   owns the tree then.
3. Runs `git fetch --prune --all`, so every remote is fetched and gone branches
   are pruned.
4. With `--check` it reads `@{upstream}`, then prints one line and stops:

        origin/main  3 behind · 1 ahead   rebase: clean

   The `clean` or `conflicts` word comes from
   `git merge-tree --write-tree HEAD <upstream>`, so it predicts the rebase
   rather than trying it. A branch with no upstream fails instead.
5. Without `--check` it runs `git pull --rebase --autostash` with your extra
   arguments appended.
6. Records an oplog entry before it runs and prints the footer after it: the new
   HEAD as changeId, short sha, ahead and behind. A `--check` run does neither,
   so it never consumes an undo step.

## Examples

    gg up                       fetch, then rebase on origin/main
    gg update                   the same, spelled out
    gg up --check               1 behind · 0 ahead   rebase: clean
    gg up -q                    -q reaches git pull
    gg -p up                    print "gg up" and run nothing
    gg up                       a conflicting rebase stops; gg abort gives up on it

## Errors

    gg update --check: this branch has no upstream
        The check needs an upstream to compare against.

A conflicting rebase prints git's own messages; the rebase stays stopped, so
`gg continue` or `gg abort` is the next step.

## Git equivalent

    git fetch --prune --all && git pull --rebase --autostash
    git fetch --prune --all && git merge-tree --write-tree HEAD origin/main
                                  for --check

## See also

    gg push    publish, with the drift guards
    gg sync    rebase on a named branch, then push
    gg abort   give up on a conflicting rebase
    gg remote  the configured remotes
