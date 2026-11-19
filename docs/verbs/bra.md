# gg bra

Draw the local branches as stacks: every branch under the branch it was cut
from, so a chain reads top to bottom. Reach for it when you keep several
branches on top of each other and want the chain instead of a flat list.

## Usage

    gg bra
    gg br -a

Takes no arguments.

## What it does

1. Reads the local branches with
   `git for-each-ref --sort=-committerdate refs/heads`, as every gg branch view
   does.
2. Derives every parent in Go, storing nothing: the parent of a branch is the
   deepest branch tip its own history already contains, tested with
   `git merge-base --is-ancestor`. Two branches cut from the same commit contain
   neither tip, so they stay siblings instead of naming each other.
3. Treats two things as floors, never as a child of another branch: the default
   branch (`origin/HEAD`, else `main`, `master`, `trunk` or `develop`) and
   gitbutler's own `gitbutler/*` refs. The default branch is listed first.
4. Prints a dim `branch stacks (N):` line, then one row per branch: a star on the
   current branch, the branch changeId, the name indented under its parent, then
   `base` for a root or `↑n on <parent>` for a branch sitting on another. The
   upstream drift and a `default` mark close the row when they apply.
5. Opens an indent level only when a branch has a single child: two children are
   parallel branches, and indenting them would claim they sit on each other.

Because the parent is derived and never stored, a stack built with plain git or
with another tool reads exactly the same here. The view only reads: no oplog
entry, no footer.

## Examples

    gg bra              the stacks, newest branch first
    gg br -a            the same, spelled as the flag
    gg -p bra           print mode: prints `gg bra`, runs nothing

A branch cut from the default branch is a root; a branch stacked on it indents
under it:

    branch stacks (2):
      * n6 main  base  no upstream  default
        k3 └ side  ↑2 on main  no upstream

## Errors

    gg br -a: no branches yet
        The repository has no local branch to stack; exits 1.

## Git equivalent

None: gg does it in Go. The layout comes from `git for-each-ref`,
`git merge-base --is-ancestor` and `git rev-list --count`.

## See also

    gg branch   the flat list, with the same changeIds and drift
    gg brlm     branches already merged into a target
    gg status   the stacks drawn as lanes in the tree
    gg dash     the same tree over the newest branches
