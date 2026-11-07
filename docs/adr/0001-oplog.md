# ADR 0001: Oplog, undo and redo

## Status

Accepted.

Supersedes the single-ref undo mechanism (`refs/gg/undo/prev`).

## Context

`gg` is a small git front end. Many verbs change the worktree or the history and record an entry: `discard`, `uncommit`, `amend`, `squash`, `split`, `move`, `abandon`, `update`, `push`. A user needs a way back after each one.

The first mechanism stored one commit in `refs/gg/undo/prev`. `gitx.Snapshot` wrote the ref. `gitx.Prev` read it. Undo ran `reset --soft`.

That mechanism had limits:

- It held one step only.
- It did not snapshot the worktree. Untracked files were never saved.
- `reset --soft` left the changes in the index.
- `gg discard` could not be reverted, because the discarded state was never stored.
- The next mutating verb overwrote the ref.

The repository contract says the index is not a user surface. `README.md` says "the index stays git's business". Undo must restore the worktree directly, without asking the user to stage or stash.

Inspection verbs do not record: `status`, `log`, `diff`, `show`, `branch`, `id`, `oplog`, `undo` and `redo`. The mutating set lives in `internal/verbs/verbs.go` and is documented in `internal/verbs/verbs.tsv`. `main.go` checks it once per run, before dispatch to git or to a Go function.

A snapshot must also survive a failed verb, a deleted file and a branch switch. The state is the whole checkout, not only HEAD. That requirement drives the design.

A wrong `amend`, `squash` or `discard` rewrites the branch. The old content must stay reachable until the user decides. A one-step ref cannot hold enough states for that.

Dry runs (`-p`, `--print`, `-n`, `--dry`) print the command and return before `Record`. A dry run does not change the log.

## Decision

The oplog is a chain of commits. It is reachable from `refs/gg/oplog/tip`. Three refs describe the chain:

- `refs/gg/oplog/tip` points at the newest entry.
- `refs/gg/oplog/cur` marks the entry the worktree currently reflects.
- `refs/gg/oplog/redo` marks the entry an undo left behind. Redo or a new snapshot clears it.

An entry is built with `git commit-tree`:

- P1 is the previous entry.
- P2 is HEAD at snapshot time, when HEAD exists and differs from P1.
- The message is the verb name.
- The tree is the worktree snapshot.

The committer identity is the marker `gg oplog <gg-oplog@localhost>`. `p1` and `p2` use this marker to tell an entry from a normal commit.

The worktree snapshot uses a temporary index at `<git-dir>/gg-tmp-index`. The steps are `read-tree HEAD`, `add -A`, `write-tree`. The temporary index is removed after use. The real index and HEAD are not touched while recording. When the repository has no HEAD yet, the base is the empty tree from `git mktree`.

`Record(verb)` runs before every verb in the mutating set. The set is gated by `verbs.IsMutating`. The call sites are `execArgv` and `execFunc` in `main.go`, and also `execTool`. `Record` returns early when the current directory is not inside a repository.

The chain grows by one commit per recorded verb. Entries are ordinary commit objects. Standard git commands can read them, and git deduplicates their trees and blobs.

`undo` and `redo` are not in the mutating set. They change the refs and the worktree directly.

`gg oplog` lists `cur` and its ancestors, newest first. Each line has the short sha, the relative age and the verb. `cur` and `tip` can differ after an undo: `tip` stays on the newest entry, `cur` moves back to the restored one. The list follows `cur`, so the entries after it are not shown.

`gg undo [N]` works in this order:

1. Snapshot the current state. Do this only when it differs from `cur`. This records an extra `undo` entry when the worktree is dirty.
2. Find the Nth ancestor of `cur`.
3. Restore it with `git reset --hard <P2>` and then `git read-tree -u --reset <entry>`.
4. Point `cur` at the restored entry and `redo` at the old tip.

When the worktree already matches `cur`, the newest entry is not a step. `undo N` then moves N entries back from `cur`. The relative age uses compact units: `5m`, `2h`, `10d`, `3w`.

`gg redo` restores the entry stored in `refs/gg/oplog/redo` and clears that ref. The ref holds one sha, so redo moves forward one step, not a stack of steps. A new `Record` clears the ref as well.

`trim()` keeps the log bounded. When the chain grows past 100 entries, it rebuilds the newest 100. Each rebuilt entry keeps the same tree, P2, message and original committer date. `tip` and `cur` move to the rebuilt chain. `undo` refuses a count larger than the chain and exits 1.

The snapshot includes untracked files. Therefore `gg discard` and every other mutating verb is revertible with `gg undo`. No stash and no extra commit are needed for that.

## Consequences

### Good

- Undo now covers the full worktree: tracked edits, new untracked files and deletions.
- `gg discard` is revertible. The user does not need to commit or stash before a risky verb.
- The history is inspectable with standard git commands, for example `git log refs/gg/oplog/tip`.
- `gg oplog` is read-only. It does not add an entry.
- Undo does not need a clean worktree. It records the dirty state first, then restores.
- The log holds many steps. `gg undo N` moves back N entries in one call.
- Git does the storage. Unchanged files are shared blobs, so repeated snapshots are cheap in content.
- The real index and HEAD are untouched while recording. A failed snapshot does not corrupt the repository.
- The marker identity keeps entries separate from normal commits. `p1` and `p2` stay unambiguous.
- `redo` is explicit. A new snapshot drops the redo target, so the user cannot redo into a stale state.

### Bad

- Every mutating verb runs a full `add -A` of the worktree. The cost grows with the number of files, not with the number of changes.
- Each entry adds one tree object. Each changed file adds a new blob. Storage grows with the edit history of the worktree.
- `gg oplog` shows only the reachable chain. Entries dropped by `trim` are not listed.
- Pruning happens only through `trim` and `git gc`. Dropped entries stay in `.git` until git collects them.
- After the log reaches 100 entries, every `Record` rebuilds the whole chain: 100 `commit-tree` calls per mutating verb.
- The log is not a durable backup. It is local state under `refs/gg/` and it is not pushed.
- `Record` runs before the verb validates its input. A verb that fails still writes an entry, and that entry can block or shift `undo`.
- `undo` and `redo` are not in the mutating set. Only `undo` records a snapshot, and only when the worktree differs. `redo` records nothing.

### Risks and known gaps

1. **HIGH — `redo` destroys local edits.** `Redo` calls `restore` without recording the current state. Uncommitted tracked edits made between `undo` and `redo` are lost silently. Reproduced: run `undo`, then `echo EDITED >> base.txt`, then `gg redo`; `base.txt` comes back without `EDITED`.
2. **MEDIUM — restored changes are left staged.** `undo` and `redo` restore with `read-tree -u`, so files appear staged (`A a.txt`). This contradicts the contract that the index is never a user surface. Reproduced: after `undo`, `git status --short` shows `A  second.txt`.
3. **MEDIUM — secret exposure.** Every untracked, non-ignored file is written into git objects, and it stays reachable from the oplog ref. A discarded secret survives in `.git` until `trim` drops the entry and `git gc` prunes it. Reproduced: a discarded secret file is still in the tree of `refs/gg/oplog/cur`.
4. **MEDIUM — an undo can remove published commits.** Restore runs `reset --hard`, so the current branch ref moves backwards. Commits that were pushed can disappear from the branch. There is no guard like the one `push` uses. Reproduced: after `undo`, `git rev-list --count HEAD` fell from 2 to 1.
5. **LOW — ignored files are not captured.** Files matched by `.gitignore` are not in the snapshot and cannot be restored. Reproduced with `*.secret`.
6. **LOW — the temporary index path is fixed.** Two `gg` processes in the same repository can clobber each other's snapshot. Evidence: `snapshotTree` always uses `<git-dir>/gg-tmp-index`.
7. **LOW — an empty log is an error.** `gg oplog` with no entries prints `gg: no operations recorded` and exits 1. Reproduced in a fresh repository.
8. **LOW — `trim` does not move the redo ref.** A redo target can fall outside the rebuilt chain. Evidence: `trim` sets `tip` and `cur` only.

### Alternatives considered

- **Reflog of a single ref.** Rejected. It records only HEAD, it has no worktree state, and git can trim it.
- **A plain text file in `.git`.** Rejected. It gives no content deduplication and needs manual parsing and manual cleanup.
- **`git stash create` objects.** Rejected. It adds a second mechanism. The pre-operation snapshot already covers the worktree, including untracked files, and the redo ref already covers the forward step.

## References

- `internal/oplog/oplog.go` — refs, `Record`, `List`, `Undo`, `Redo`, `restore`, `snapshotTree`, `commitEntry`, `trim`.
- `internal/oplog/oplog_test.go` — chain order, undo/redo, redo clearing, retention.
- `main.go` — `execArgv`, `execFunc`, `execTool` call `oplog.Record`.
- `internal/verbs/verbs.tsv` — verb registry; the comment on mutating verbs.
- `internal/verbs/verbs.go` — `mutating` map and `IsMutating`.
- `internal/gitx/gitx.go` — `Run`, `Output`, `Head`, and `InRepo`.
- `test/integration.sh` — oplog, undo, redo and retention cases.
