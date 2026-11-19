# gg merge

Merge a branch into the one you are on, fast-forward only by default. A merge
commit is a decision, so gg never makes one unless you ask for it.

## Usage

    gg merge <branch> [flags]
    gg merge --abort

## Aliases

    mg    the same verb, two letters

## Arguments

    branch   a branch or any commit-ish. A branch changeId names its branch. The
             first argument that does not start with `-` is read as the branch,
             so write the branch before a flag that takes a value.

## Flags

    None of its own. Any merge flag turns the fast-forward default off and
    reaches git verbatim: --no-ff, --ff, --ff-only, --squash, -m/--message,
    -F/--file, -e/--edit, --no-edit, -s/--strategy, -X/--strategy-option,
    --no-commit, --commit, --no-verify, --allow-unrelated-histories,
    --log/--no-log, and the recovery flags --abort, --continue, --quit.

## What it does

1. A wip commit is folded first, so uncommitted edits ride into the merge.
2. The branch is resolved next. A name that is neither a branch nor a commit
   stops before git runs: `gg merge: no branch or commit named <name>`.
3. With no merge flag of your own and no `merge.ff` in the config, gg adds
   `--ff-only`:

        git merge --ff-only <args>

   A flag of your own, or a configured `merge.ff`, means the repository owns the
   policy, so the plain `git merge <args>` runs.
4. When gg added `--ff-only` and git refuses, it says so and points at the
   alternatives; nothing was merged.
5. The call is recorded in the oplog and the footer prints the new HEAD.

## Examples

    gg merge side               fast-forward onto side
    gg mg --no-ff side          keep a merge commit on purpose
    gg merge side -m "join"     a merge commit with its own message
    gg merge --abort            drop a conflicted merge
    gg -p merge side            print the gg call, run nothing

## Errors

    gg merge: no branch or commit named <name>
        The first non-flag argument resolves to nothing. This is also what
        happens when a flag value comes first: `gg merge -m "msg" side` reads
        `msg` as the branch.
    gg merge: not a fast-forward, so nothing merged.
        The added --ff-only refused; the branch is unchanged.
    gg merge: <branch> --no-ff keeps a merge commit, gg rebase <branch> replays it, gg land <branch> pushes it.
        The hint printed right below that refusal.

## Git equivalent

    git merge --ff-only <branch>     the default call
    git merge --no-ff <branch>       with a flag of your own, git decides

## See also

    gg rebase    replay the branch instead of merging it
    gg land      fast-forward the target and push it
    gg abort     unstick a merge that conflicted
    gg conflict  resolve conflicts with the configured tool
