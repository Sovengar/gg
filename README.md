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
gg                 dashboard: branch, files, commits, branches
gg st              status --short
gg stf             status including untracked
gg log             short graph log, 6 commits by default, ✓ pushed / ↑ local
gg log 20          more of it, extra args go to git log
gg log -l 40       long line: relative age and author
gg files           ls-files
gg branches        branches sorted by recency
gg reflog          reflog
gg diff -w         diff ignoring whitespace
gg aa              add -A
gg c -m "msg"      commit
gg amend           fold staged changes into HEAD
gg amend <id>      fold staged changes into another commit (git history fixup)
gg reword <id>     edit that commit's message (git history reword)
gg split <id> -- f cut that commit in two by paths (git history split, interactive)
gg abandon <id>    drop that commit (git history drop)
gg uncommit 2      move the last 2 commits back into the working copy
gg absorb          git absorb
gg update / gg up  fetch --prune --all + pull --rebase --autostash
gg push / gg pu    push, refuse on drift, offer force-with-lease
gg sync <branch>   pull --rebase origin <branch>, then push
gg sync            fails: the branch argument is required
gg abort           abort whichever operation is stuck: rebase, merge, cherry-pick
gg undo            reset --soft to the last snapshot
gg raw log -5      anything goes, still snapshotted
gg help            verb table
gg tools           external tools + install state
```

Unknown verbs go straight to git: `gg log --oneline -5` runs `git log --oneline -5`.

Help is grouped by action: inspect, staging, commit, history, remote, branches,
stash, safety, tools. Aliases share one row: `c/commit`, `pu/push`, `up/update`.
Each row shows the git command when the verb is a single git call.

Flags: `-p/--print`, `-n/--dry` print the resolved command instead of running.
Environment: `GG_ECHO=1` traces the git call, `NO_COLOR=1` disables colour.

### Handles

Every commit shows a three character handle derived from its sha, so it never
changes:

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
`gg sw @nan` switches to the branch rather than to a detached HEAD.

### External tools

Registered as `tool:<binary>` in `verbs.tsv`. Extra args are passed through.

```
gg story           gitlogue — animated history replay
gg mzt             git-machete — branch layout maintenance
gg sim rebase -i HEAD~3   git-sim — render the operation first
gg wip save        git-wip — save/inspect/list work in progress
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
`graph handle sha age subject decorations [author]`. Lanes use box characters,
the handle is red, and the ball is **green when every remote branch already has
the commit, yellow when it is local only**. Age is compact: `30s`, `5m`, `2h`,
`10d`, `3w`, `2mo`. `-l/--long` shows the full sha.

### Dashboard

Bare `gg` prints the working state and stops there:

```
trunk  n6 66533a6  ↑2 ↓0  stash 1

files
  kd  M docs/README.md
  qz  D docs/old.md
  ar  A src/feature.sh
  px  ?? notes.md

commits
  @hpi 56b174f  chore: dev env (local only)   (HEAD -> trunk)
  @mpa 8f1c2a1  feat: session helper (local only)

branches
  * @hpi trunk         ↑2 ↓0
    @k3d experiment    no upstream
    @9fa feature/login ↑0 ↓1
```

`gg dash 10` overrides how many commits are listed.

### Built on git history

Git 2.56 ships the experimental `git history` command. gg uses it when present
and falls back to `git rebase --onto` for `abandon`:

| gg | git |
|---|---|
| `gg abandon <id>` | `git history drop <id>` |
| `gg amend <id>` | `git history fixup <id>` |
| `gg reword <id>` | `git history reword <id>` |
| `gg split <id>` | `git history split <id>` |

`reword` keeps the Change-Id trailer, so the handle does not move.

### jj edit

jujutsu makes a commit the working copy: `jj edit <rev>`. git has no working
copy commit, so `gg edit <id>` is the closest equivalent — it stops an
interactive rebase on that commit, you amend it, and `gg continue` resumes.

## Semantics

- Every mutating verb snapshots `HEAD` into `refs/gg/undo/prev`, so one `gg undo` reverts any of them.
- Every mutating verb ends with a footer: `@ <sha> · ahead N · behind M`.
- `gg push` never force-pushes on its own. It reports the drift, lists the incoming
  commits, and asks: update, force-with-lease, or abort. Non-interactive runs refuse.
- User arguments are appended verbatim, so any verb accepts extra git flags.

## Layout

```
main.go                 dispatch, flags, footer, undo, raw
internal/verbs/         registry (verbs.tsv embedded) + resolution
internal/gitx/          git binary wrapper
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
