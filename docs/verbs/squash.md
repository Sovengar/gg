# gg squash

Fold one commit into its parent. The commit disappears, its change stays under
the parent's message, and the commits after it are replayed on the result.

## Usage

    gg squash <changeId>

## Arguments

    changeId   the commit to fold. It must name a commit; the argument is
               resolved to a sha before the rebase starts.

## What it does

1. A wip commit is folded first, so an uncommitted edit rides along rather than
   being left behind by the rewrite.
2. It refuses while a rebase is already running:
   `gg: a rebase is already in progress; use gg continue or gg abort`.
3. It verifies the commit first: `gg: <arg> does not name a commit`.
4. A merge commit cannot fold into a single parent:
   `gg: <sha> is a merge commit; a merge cannot fold into its parent`.
5. It runs an interactive rebase from the parent with a scripted todo list:

        GIT_SEQUENCE_EDITOR="<gg binary> __seq-edit fixup <sha>" \
            git rebase -i --autostash --empty=drop <sha>^

   gg rewrites that commit's todo line to `fixup`, so git folds it into the
   parent and keeps the parent's message. `--autostash` carries uncommitted
   edits across the rewrite and `--empty=drop` drops a commit that ends up
   empty.

## Examples

    gg squash @kd          fold that commit into the one before it
    gg squash kd           same, by the short changeId
    gg -p squash kd        print the gg call, run nothing

## Errors

    gg squash <commit>
        No argument.
    gg: a rebase is already in progress; use gg continue or gg abort
        Finish or abort the running rebase first.
    gg: <arg> does not name a commit
        The argument resolves to no commit.
    gg: <sha> is a merge commit; a merge cannot fold into its parent
        Use gg edit to amend the merge instead.
    gg: <sha> is not inside the rebase range
        The plan git built does not hold that commit.

## Git equivalent

    None: gg does it in Go. Underneath it scripts the interactive rebase:

        GIT_SEQUENCE_EDITOR="gg __seq-edit fixup <sha>" git rebase -i --autostash --empty=drop <sha>^

    That is the same as marking that commit `fixup` by hand in
    `git rebase -i <commit>^`.

## See also

    gg edit      stop the rebase on a commit instead of folding it
    gg amend     fold the working tree into a commit
    gg describe  keep the commit, change its message
    gg rebase    replay a whole branch
