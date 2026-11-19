# gg edit

Stop a rebase on one commit, so the working tree sits at that commit and you
can change it. It is gg's closest thing to `jj edit`: the stop is a rebase edit
point, and `gg continue` replays the rest afterwards.

## Usage

    gg edit <changeId>

## Aliases

    e     the same verb, one letter

## Arguments

    changeId   the commit to stop on. It must name a commit.

## What it does

1. A wip commit is folded first, so the rewrite starts above it.
2. It refuses while a rebase is already running:
   `gg: a rebase is already in progress; use gg continue or gg abort`.
3. It verifies the commit: `gg: <arg> does not name a commit`.
4. It runs an interactive rebase from the parent, with a scripted todo list:

        GIT_SEQUENCE_EDITOR="<gg binary> __seq-edit edit <sha>" \
            git rebase -i --autostash --empty=drop <sha>^

   gg rewrites that commit's todo line to `edit`, so git stops right after
   applying it. `--autostash` carries uncommitted edits across the rewrite.
5. While the rebase is stopped on an edit point, the following gg commands
   first snapshot the working tree into the stopped commit (`git add -A`,
   then `git commit --amend --no-edit`). There is no staging step to run and
   nothing is lost between calls. `gg abort`, `gg undo` and `gg redo` skip that
   snapshot, and so does a conflicted or clean tree.
6. A merge commit is editable too: gg adds `--rebase-merges` and puts a `break`
   right after the merge, so the stop keeps both parents and
   `git commit --amend` still works on it.
7. `gg continue` resumes the rebase and replays the rest.

## Examples

    gg edit @kd          stop on that commit
    gg e kd              same, one letter
    gg diff              look at what the stop holds
    gg continue          replay the commits above it

## Errors

    gg edit <commit>
        No argument.
    gg: a rebase is already in progress; use gg continue or gg abort
        One rebase at a time.
    gg: <arg> does not name a commit
        The argument resolves to no commit.
    gg: <sha> is not inside the rebase range
        The plan git built does not hold that commit.
    gg: cannot find my own binary: <error>
        gg could not name itself for GIT_SEQUENCE_EDITOR.

## Git equivalent

    None: gg does it in Go. Underneath it scripts the interactive rebase:

        GIT_SEQUENCE_EDITOR="gg __seq-edit edit <sha>" git rebase -i --autostash --empty=drop <sha>^

    That is the same as marking that commit `edit` by hand in
    `git rebase -i <commit>^`.

## See also

    gg continue  resume after the stop
    gg abort     drop the rebase and go back
    gg amend     fold the tree into the commit where it stopped
    gg squash    fold the commit into its parent instead
    gg rebase    replay a whole branch
