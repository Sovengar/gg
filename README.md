# gg

Small git front-end with jj/but flavoured verbs. A single Go binary, no shell glue.

## Install

```sh
go build -o ~/.local/bin/gg .    # needs -buildvcs=false outside a repo
```

Completions live in `~/.config/fish/completions/gg.fish` and read `gg verbs`.

Note: on this machine `/usr/bin/go` reports a GOROOT mismatch. Build with the mise
toolchain and clear GOROOT:

```sh
GOROOT= ~/.local/share/mise/installs/go/1.27.3/bin/go build -o ~/.local/bin/gg .
```

## Usage

```
gg                 dashboard: the tree over the five newest branches
gg status/st       the tree: files, worktree lanes, common base, hints
gg status -f/stf   the same tree, up to five changed paths under each commit
gg status -a/sta   the same tree, every changed path, no five-file cap
gg log             short graph log, 6 commits by default, ✓ pushed / ↑ local
gg log 20          more of it, extra args go to git log
gg log -l 40       long line: relative age and author
gg branch          branches sorted by recency (alias: br)
gg br -r           remote branches only
gg br -a           local and remote branches in one list
gg branch -c side  create a branch and switch to it (alias: cbr; branch new works too)
gg branch -d side  delete a branch, its tip kept in refs/gg/deleted (alias: dbr)
gg branch -D side  force the delete, unmerged commits and all (alias: Dbr)
gg reflog          reflog
gg diff            diff in boxes, one per change (plain git diff when piped)
gg diff -w         diff ignoring whitespace
gg copy kd main    copy that commit onto main, back on your branch after (alias: cp)
gg copy kd         copy it onto the branch you are on
gg c -m "msg"      commit every dirty path (aliases: commit, ca)
gg c -m "msg" kd qz  commit only those changeIds
gg discard kd      drop that change, keep the rest; undoable with gg undo
gg discard         drop every edit, keep the new files
gg clear           drop every edit and delete the new files (-f skips the question)
gg new             save the changes, then absorb more until you commit
gg amend           fold changes into HEAD
gg amend <id>      fold changes into another commit (git history fixup)
gg reword <id>     edit that commit's message (aliases: rename, describe, desc)
gg split <id> -- f cut that commit in two by paths (git history split, interactive)
gg abandon <id>    drop that commit (git history drop)
gg revert <id>     undo that commit with a new commit, safe once published (alias: rv)
gg uncommit        move HEAD back into the working copy, change and all
gg uncommit <id>   same for any commit, its children replayed on its parent
gg wt new side     a worktree for a branch, one directory each
gg wt ls           each worktree, its path and how dirty it is
gg wt rm side      remove it, --force when dirty
gg absorb          git absorb
gg fetch           plain git fetch: fetch.all and fetch.prune decide the scope
gg update / gg up  fetch --prune --all + pull --rebase --autostash
gg push / gg p     push, refuse on drift, offer force-with-lease
gg sync <branch>   pull --rebase origin <branch>, then push
gg sync            fails: the branch argument is required
gg abort           abort whichever operation is stuck: rebase, merge, cherry-pick
gg undo            step back to the state before the last operation
gg undo 3          step back three operations
gg redo            step forward again after an undo
gg oplog           the operations undo can reach
gg oplog clear     drop the log, then git gc --prune=now frees the objects
gg undo --force    undo even when it removes published commits
gg help            compact verb table: canonical names per category, aliases under (alias: h)
gg help -all       full table: description, hint and the git command per verb
gg tools           external tools + install state
```

Unknown verbs go straight to git: `gg log --oneline -5` runs `git log --oneline -5`.

Help is grouped by category: Inspection, Rollback work, Register work,
Edit saved work, Branching, Operations, Remote, Tools, Others. There is no stash: the
working tree, the index and wip commits are one mechanism (`gg new`, `gg wt`).
`gg help` (or `gg h`) prints one line of names per category, then a grey `aka`
line with that category's aliases, so the whole table stays on screen. Both
lines wrap at `COLUMNS`, 80 by default. A shorthand sits next to its verb in
parentheses, e.g. `reword (desc)`; a real alias goes on the `aka` line, e.g.
`aka rename · describe`.
`gg help -p` lists the parameter hints and the flag variants, each variant named
with the command it stands for: `status -a (sta)`, `status -f (stf)`.
`gg help --all` prints the full table: description, grey parameter hint and the
git command behind each verb.

Flags: `-p/--print`, `-n/--dry` print the resolved command instead of running.
Environment: `GG_ECHO=1` traces the git call, `NO_COLOR=1` disables colour.

### Verifying

```
gofmt -l . && go vet ./... && go test ./... && bash test/integration.sh
```

`test/integration.sh` builds gg itself and runs every case against real
repositories. When the local `go` reports a toolchain mismatch (`compile:
version ... does not match go tool version`), the script retries with
`mise exec go@1.26.3`; use the same fallback by hand for `go test`.

### ChangeIds, not staging

No verb asks you to stage. Every dirty path already carries a changeId, so `gg c`
takes changeIds and the index stays git's business:

```
gg dash            kd M docs/README.md, px ?? notes.md
gg c -m "docs" kd  commit one path, leave px dirty
gg discard kd      drop a change, keep the rest
gg clear           drop every edit, then delete the new files
gg new             or park everything in a wip commit on the branch
```

`gg discard` drops edits and keeps the new files: a new file is not a change
to drop. `gg clear` drops the edits and deletes the new files, since no commit
holds them and git cannot bring one back; without `-f` it asks first, and
refuses when nobody can answer.

### Work in progress and worktrees

`gg new` turns the working tree into a commit on the current branch. While that
commit is HEAD, `gg dash` prints a `wip` badge, the verbs that rewind history
fold your changes into it first (`gg br <name>`, `gg update`, `gg sync`, `gg amend`,
`gg copy`, `gg uncommit`), and `gg push` refuses until you finish it with
`gg c -m "msg"` or take it back with `gg uncommit`. `gg c` with no message
refuses while in wip mode. There is no stash.

Work that must not share a checkout goes to its own worktree:

```
gg wt new side     sibling directory <repo>.side on branch side; folds wip first
gg wt ls           branch, path, and how dirty each worktree is
gg wt rm side      remove it; --force when it is dirty
```

### ChangeIds

Every file, commit and branch in gg carries a changeId: two to four
characters that name it.

- A commit takes its changeId from the `Change-Id` trailer, so an amend, a
  rebase or a squash keeps the same changeId while the sha changes. A commit
  without that trailer falls back to a changeId derived from its sha.
- A branch takes its changeId from the branch name, so two branches on one
  commit never share it.
- A dirty path takes its changeId from the path.

A changeId is the shortest unique prefix, never shorter than two characters:

```
* @nan 641682b (2 minutes ago) chore: dev env  (HEAD -> trunk) [Playground]
* @mpk c1f10a7 (2 minutes ago) feat: session helper  [Playground]
```

Any argument shaped like `@abc` is resolved before the verb runs, and a
two-character prefix works when it is unambiguous:

```
gg show @nan       git show trunk
gg id @nan         @nan 641682b7a6c27c7273086ea06574b451742d5792  (trunk)
gg diff @na        resolves, then diffs
```

When the commit is the tip of a local branch the branch name is used, so
`gg br @nan` switches to the branch rather than to a detached HEAD. Branch
verbs (`gg br`, `gg cbr`, `gg dbr`, `gg mg`) read a branch changeId
first, so `gg br 4d` names the branch even when another branch holds the same
commit.

### Branch list after a delete

`gg dbr` ends with the branch list it just changed: the five newest branches
and the default branch, always. The default branch is the one `origin/HEAD`
points at, else `main`, `master`, `trunk` or `develop`, else the current branch.

```
  branches (5 newest, default always):
    be b     no upstream
    pk c     no upstream
  * 2h main  no upstream  default
```

### External tools

Registered as `tool:<binary>` in `verbs.tsv`. Extra args are passed through.

```
gg story           gitlogue — animated history replay
gg mzt             git-machete — branch layout maintenance
gg sim rebase -i HEAD~3   git-sim — render the operation first
gg absorb          git-absorb — fold changes into the right commit
```

`gg tools` prints each tool verb and whether the binary is on PATH. Adding a tool
is one line in `internal/verbs/verbs.tsv`; no Go change.

### Log output

```
● n6 66533a6 25m chore: dev env (local only)  (HEAD -> trunk) [Playground]
● cn 7006a00 25m feat: session helper (local only) [Playground]
● 7m c56fca7 25m docs: add legacy notes  (origin/trunk, origin/HEAD) [Playground]
```

One format everywhere: `gg log`, `gg dash` and the graph all print
`graph changeId sha age subject decorations [author]`. Lanes use box characters,
the changeId is red, the decoration `(...)` is yellow, and the ball is **green when every remote branch already has
the commit, yellow when it is local only**. Age is compact: `30s`, `5m`, `2h`,
`10d`, `3w`, `2mo`, painted by recency: bright green under an hour, then
green, yellow and dim. A merge line also lists both sides it joined, as
`{8fb1f04 62638fe}`. `-l/--long` shows the full sha.

### Dashboard

Bare `gg` prints the tree over the five newest branches, the default branch
among them. `gg status` narrows the lanes to real worktrees: the current branch
first, then every branch a sibling worktree holds. `gg status -f` (alias `stf`)
adds, under every commit, up to five changed paths and a `+N more` tail;
`gg status -a` (alias `sta`) lists every changed path instead. No header line:
starship already shows the branch, the drift and the wip state.

The shape follows `but status`: one lane for the uncommitted files, one lane
per branch, the common base and two hints.

```
╭┄ zz [uncommitted]
┊   kd M docs/README.md
┊   qz D docs/old.md
┊   ar A src/feature.sh
┊   px A notes.md
┊
┊╭┄ n6 [trunk]
┊● n6 66533a6 2m feat: session helper (HEAD -> trunk) [Sovengar] (N)
┊● mp 8f1c2a1 1h chore: dev env [Sovengar] (G)
┊│
┊├┄ k3 [experiment]
┊● k3 41d0c11 3h feat: try the split [Sovengar] (N)
├╯
┊
┴ 0a1b2c3 (common base) 2026-11-01 Initial commit

Hint: commits are listed newest first. The first token on each line is the changeId to use in commands.
Hint: run `gg diff` to see uncommitted changes and `gg c -m "message"` to commit them
```

Untracked paths read `A`, like a branch that adds the file. The current branch
lane always carries commits: when the branch sits on the base it lists its own
last commits instead of an empty range. Another lane with nothing above the base
prints `(at the common base)`, so the default branch always keeps its row in the
dashboard. The base is the merge base with `gitbutler/target`,
the default branch or the upstream. `gg st 10` overrides how many commits each
lane lists; gitbutler's own refs never become lanes.

`gg diff` draws the same boxes `but diff` draws: a border sized to the title,
the file changeId and path, then a gutter of old and new line numbers with red
removals and green additions. When stdout is not a terminal it runs plain
`git diff`, so pipes still parse.

### Built on git history

Git 2.56 ships the experimental `git history` command. gg uses it when present
and falls back to `git rebase --onto` for `abandon`:

| gg | git |
|---|---|
| `gg abandon <id>` | `git history drop <id>` |
| `gg amend <id>` | `git history fixup <id>` |
| `gg reword <id>` | `git history reword <id>` |
| `gg split <id>` | `git history split <id>` |

`reword` keeps the Change-Id trailer, so the changeId does not move when the
sha does.

### jj edit

jujutsu makes a commit the working copy: `jj edit <rev>`. git has no working
copy commit, so `gg edit <id>` is the closest equivalent — it stops an
interactive rebase on that commit, you amend it, and `gg continue` resumes.

## Semantics

- Every mutating verb records an oplog entry before it runs: HEAD, the full worktree
  (tracked and untracked, `.gitignore` respected), and branches, tags and `refs/gg/*`.
  `refs/remotes` is never touched. `gg oplog` lists them newest first, `gg undo` restores
  the state before the last operation, `gg undo N` goes back N entries, and `gg redo`
  moves forward again when the last action was an undo. This also makes `gg discard`,
  `gg clear` and branch or tag changes revertible. The newest 100 entries are kept.
- Every mutating verb ends with a footer: `@ <sha> · ahead N · behind M`.
- `gg push` never force-pushes on its own. It reports the drift, lists the incoming
  commits, and asks: update, force-with-lease, or abort. Non-interactive runs refuse.
- User arguments are appended verbatim, so any verb accepts extra git flags.

## Layout

```
main.go                 dispatch, flags, footer
internal/verbs/         registry (verbs.tsv embedded) + resolution
internal/gitx/          git binary wrapper
internal/oplog/         worktree snapshots, undo and redo
internal/hist/          graph renderer with pager
internal/ops/           push guard, update, sync, uncommit
internal/ui/            colour, prompts, tty detection
test/integration.sh     end-to-end tests against real repositories
```

## Tests

```sh
go test ./...
./test/integration.sh
```
