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
gg init            git init here, then colocate a jj repo on it (--no-colocate, --bare skip jj)
gg clone <url>     git clone, then colocate a jj repo on it (then an optional dir)
gg                 dashboard: the tree over the five newest branches
gg status/st       the tree: files, worktree lanes, common base, hints
gg status -f/stf   the same tree, up to five changed paths under each commit
gg status -a/sta   the same tree, every changed path, no five-file cap
gg log             short graph log, 6 commits by default, ✓ pushed / ↑ local
gg log 20          more of it, extra args go to git log
gg log -l 40       long line: relative age and author
gg branch          branches sorted by recency (alias: br)
gg br -lr          remote branches only
gg br -a           the stacks: each branch under the one it was cut from
gg br -la          local and remote branches in one list
gg br -lm          branches already merged into HEAD (br -lm main: into main)
gg branch -c side  create a branch and switch to it (alias: cbr; branch new works too)
gg discard side    delete a branch, its tip kept in refs/gg/deleted
gg discard <id>    drop that commit too (git history drop)
gg discard side -f force a delete the tree or history would refuse
gg reflog          reflog
gg tag             tags newest version first, with the message (alias: t)
gg tag -m v1.2.0   create one; -d deletes, -s signs
gg remote          configured remotes, names and urls
gg remote add up ..  add, rename, rm and set-url reach git
gg diff            diff in boxes, one per change (plain git diff when piped)
gg diff -w         diff ignoring whitespace
gg copy kd main    copy that commit onto main, back on your branch after (alias: cp)
gg copy kd         copy it onto the branch you are on
gg c -m "msg"      commit every dirty path (alias: commit; ce excludes)
gg c -m "msg" kd qz  commit only those changeIds
gg ce -m "msg" kd    commit every dirty path except kd (commit -e; alias: ce)
gg discard kd      drop that change, keep the rest; undoable with gg undo
gg discard         drop every edit, keep the new files
gg clear           drop every edit and delete the new files (-f skips the question)
gg new             save the changes, then absorb more until you commit
gg amend           fold changes into HEAD
gg amend <id>      fold changes into another commit (git history fixup)
gg describe <id>   edit that commit's message (aliases: reword, rename, desc)
gg split <id> -- f cut that commit in two by paths (git history split, interactive)
gg revert <id>     undo that commit with a new commit, safe once published (alias: rv)
gg uncommit        move HEAD back into the working copy, change and all
gg uncommit <id>   same for any commit, its children replayed on its parent
gg rebase <base>   replay the branch onto another base (alias: rb); -i keeps working
gg wt new side     a worktree for a branch, one directory each
gg wt ls           each worktree, its path and how dirty it is
gg wt rm side      remove it, --force when dirty
gg merge <branch>  merge it, fast-forward only; --no-ff keeps a merge commit
gg conflict/cf     git mergetool: resolve conflicts with the configured tool
gg absorb          git absorb
gg pr              open a PR for this branch; a stacked branch bases on its parent
gg pr -d           the same, as a draft (extra args go to gh verbatim)
gg pr review 42    mount that PR in its own worktree, commits intact (alias: rpr)
gg pr review 42 edit  the same, with the PR diff live in the tree (alias: epr)
gg land <branch>   fast-forward the target onto it and push: no PR, no CI
gg land -b rel     land on that target instead; -y skips the confirmation
gg fetch           plain git fetch: fetch.all and fetch.prune decide the scope
gg update / gg up  fetch --prune --all + pull --rebase --autostash
gg push / gg p     push; force a rewrite of your own commits only
gg sync <branch>   pull --rebase origin <branch>, then push
gg sync            fails: the branch argument is required
gg abort           abort whichever operation is stuck: rebase, merge, cherry-pick
gg abort --keep    abort a rebase and keep the live worktree changes (short: -k)
gg undo            step back to the state before the last operation
gg undo 3          step back three operations
gg redo            step forward again after an undo
gg oplog           the operations undo can reach
gg oplog clear     drop the log, then git gc --prune=now frees the objects
gg undo --force    undo even when it removes published commits
gg help            compact verb table: canonical names per category, aliases under (alias: h)
gg h -v            one row per verb, with its description
gg hh              the same table with the flag variants merged into their verb
gg hh -v           one row per verb and flag variant, with the parameter hints
gg tools           external tools + install state
```

Unknown verbs go straight to git: `gg log --oneline -5` runs `git log --oneline -5`.

Help is grouped by category: Inspection, Rollback work, Register work,
Edit saved work, Branching, Operations, Remote, Others. There is no stash: the
working tree, the index and wip commits are one mechanism (`gg new`, `gg wt`).
`gg help` (or `gg h`) prints one line of names per category, then a grey
footer with the aliases, verb first: `describe -> reword, rename`, and the
hint to the longer views. Every line wraps at `COLUMNS`, 80 by default. A
shorthand sits grey next to its verb, no parentheses: `describe desc`,
`commit c`. A real alias goes on the footer arrow line.
`gg hh` merges each flag variant into the cell of the verb it flags, name and
flag together: `status st, -a sta, -f stf`, `branch br, -r, -a, -g, -c
cbr`. A variant lives on one row, its parameters in `hh -v`,
and a positional row with no flag of its own (`branch <name>`) stays there.
`gg h -v` prints one row per verb with its description, no parameters.
`gg hh -v` prints one row per verb and flag variant, with the grey parameter
hint: `status -a sta  st -a shows all, st -f stops at five`.
There is no `hhh`: the description lives in `h -v`, the parameters in `hh -v`.

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
gg discard kdm     drop a change, keep the rest
gg clear           drop every edit, then delete the new files
gg new             or park everything in a wip commit on the branch
```

A dirty path also carries its hunks: each region of the diff gets a two or three
character id, and the handle is `file:hunk`. `gg diff` prints one box per hunk
with that handle, and `gg discard <file>:<hunk>` drops only that region.

```
gg diff            kdm:6a docs/README.md, kdm:89 docs/README.md
gg discard kdm:6a  drop that hunk, keep the other one
```

`gg discard` drops edits and keeps the new files: a new file is not a change
to drop. It also takes a commit changeId or a branch changeId; see "One discard,
three kinds". `gg clear` drops the edits and deletes the new files, since no commit
holds them and git cannot bring one back; without `-f` it asks first, and
refuses when nobody can answer.

### Work in progress and worktrees

`gg new` turns the working tree into a commit on the current branch. While that
commit is HEAD, `gg dash` prints a `wip` badge, the verbs that rewind history
fold your changes into it first (`gg go <name>`, `gg update`, `gg sync`, `gg amend`,
`gg copy`, `gg uncommit`), and `gg push` refuses until you finish it with
`gg c -m "msg"` or take it back with `gg uncommit`. `gg c` with no message
refuses while in wip mode. There is no stash.

`gg new` refuses when it cannot keep the tree safe. It stops while git owns the
tree (`MERGE_HEAD`, `rebase-merge`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`), because a
commit there would swallow the operation. It stops on an unmerged path, because
`git add -A` would commit the conflict markers. It stops when the wip commit is
already on the upstream, because folding rewrites a published commit; `--force`
takes that risk on purpose. Any argument other than `-m` and `--force` is an
error: `gg new` commits the whole tree, so a typo must not become a commit.

The wip marker is a file under the worktree's own git dir, so two worktrees keep
two wips and neither one drops the other's marker.

Work that must not share a checkout goes to its own worktree:

```
gg wt new side     sibling directory <repo>.side on branch side; folds wip first
gg wt ls           branch, path, and how dirty each worktree is
gg wt rm side      remove it; --force when it is dirty
gg go side         go where it lives: its worktree, or this checkout
gg fork side       the same directory, under the verb the table uses (alias: fk)
gg ship side       squash the branch into one commit on the target, drop its worktree
gg discard side    remove the worktree and the branch it holds
```

`gg st` and bare `gg` also read `wt list --format json`, so each worktree lane
carries what Worktrunk knows: `↑N` commits the main branch lacks, `↓N` commits
it already has, `⚑` changes in that worktree's tree. With a single worktree
there is nothing to compare, so gg skips the call; without the `wt` binary the
markers disappear and nothing else changes.

### Shipping a branch

`gg ship` is the composing end of a branch: it turns its commits into one,
replays it on the local target, fast-forwards the target and removes the
worktree. It never pushes. `gg land` is the publishing end: it fast-forwards the
target and pushes, and it keeps the commits. `gg pr` is the review path.

```
gg ship side       squash side into one commit on the default branch
gg ship side -m "feat: side work"   name the message when it holds several commits
```

`gg go <branch>` walks to where a branch lives: its worktree when one holds it,
and this checkout when none does. A branch that does not exist yet is created,
and gg asks whether it should live in its own worktree. `gg go <branch> -- cmd`
runs the command inside that directory instead, which is what an agent needs.

A program cannot change the shell that started it, so `gg go` prints the
directory. `gg shell install` writes the fish function that cds there:

```
gg shell install    library function in ~/.config/fish/functions/gg.fish
gg shell show       where it is, or that it is missing
gg shell uninstall  remove it
```

Without the wrapper, `cd (gg go side)` does the same thing.

`gg ship` is the composing end of a branch: it turns the commits into one, replays
it on the local target and fast-forwards it, and never pushes. `gg land` is the
publishing end: it fast-forwards the target and pushes, and it keeps the commits.
A draft pushed with `gg pr` stars the review path, so ship stays local on purpose.

### One discard, three kinds

`gg discard` takes a changeId of any kind and drops what it names. The resolver
types it first, so a path changeId, a commit changeId and a branch changeId
cannot be confused; a plain name is sorted by what it is: an existing path, then
a branch, then a commit. `path:`, `commit:` and `branch:` force one kind, and a
token that names two kinds fails with the candidates instead of guessing.

```
gg discard kd          the edit of that path, the rest of the tree stays
gg discard <commit>    the commit, its children replayed on its parent
gg discard side        the branch and the worktree that holds it
gg discard zz          every uncommitted edit, keeping new files
```

The price of each kind is asked for with the same flags: `-f` drops unmerged
commits or a dirty worktree, `-y` skips the questions, `-r` also deletes the
upstream copy. A branch that tracks an upstream asks about it, naming the remote
it actually tracks; a branch with no upstream never asks. A worktree goes only
when its question is answered: `gg undo` restores refs, not another directory.

### ChangeIds

Every file, commit and branch in gg carries a changeId: three characters that
name it, growing to four when two of them share all three.

- A commit takes its changeId from the `Change-Id` trailer, so an amend, a
  rebase or a squash keeps the same changeId while the sha changes. A commit
  without that trailer falls back to a changeId derived from its sha.
- A branch takes its changeId from the branch name, so two branches on one
  commit never share it.
- A dirty path takes its changeId from the path.

A changeId is the shortest unique prefix, never shorter than three characters:

```
* @nan 641682b (2 minutes ago) chore: dev env  (HEAD -> trunk) [Playground]
* @mpk c1f10a7 (2 minutes ago) feat: session helper  [Playground]
```

Any argument shaped like `@abc` is resolved before the verb runs, and a
two-character prefix works when it is unambiguous:

```
gg show @nan       git show trunk
gg changeId @nan   @nan 641682b7a6c27c7273086ea06574b451742d5792  (trunk)
gg diff @na        resolves, then diffs
```

When the commit is the tip of a local branch the branch name is used, so
`gg br @nan` switches to the branch rather than to a detached HEAD. Branch
verbs (`gg go`, `gg br`, `gg cbr`, `gg discard`, `gg merge`, `gg rb`) read a branch changeId
first, so `gg br 4d` names the branch even when another branch holds the same
commit.

### Branch list after a delete

`gg discard` of a branch ends with the branch list it just changed: the five newest branches
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
gg hunk            Hunk — review a diff in a TUI
gg tuicr           tuicr — code review TUI
gg but             GitButler CLI
gg jj              Jujutsu
gg lazygit         lazygit — terminal git UI
gg prdash          prdash — PR dashboard
gg gitdash         gitdash — workspace dashboard
gg conflict        git mergetool — resolve conflicts with the configured tool (alias: cf)
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
and falls back to `git rebase --onto` for `gg discard <commit>`:

| gg | git |
|---|---|
| `gg discard <id>` | `git history drop <id>` |
| `gg amend <id>` | `git history fixup <id>` |
| `gg describe <id>` | `git history reword <id>` |
| `gg split <id>` | `git history split <id>` |

`describe` keeps the Change-Id trailer, so the changeId does not move when the
sha does.

### jj edit

jujutsu makes a commit the working copy: `jj edit <rev>`. git has no working
copy commit, so `gg edit <id>` is the closest equivalent — it stops an
interactive rebase on that commit, and `gg continue` resumes. Every gg command
first snapshots the working tree into the stopped commit, the way jj snapshots
the working copy, so there is no `git commit --amend` to run by hand.

A merge commit is editable the same way: gg replays the history with
`--rebase-merges` and stops right after the merge, so `git commit --amend`
keeps both parents and `gg continue` replays the rest. `gg squash` refuses a
merge, since a merge cannot fold into a single parent.

## Semantics

- Every mutating verb records an oplog entry before it runs: HEAD, the full worktree
  (tracked and untracked, `.gitignore` respected), and branches, tags and `refs/gg/*`.
  `refs/remotes` is never touched. `gg oplog` lists them newest first, `gg undo` restores
  the state before the last operation, `gg undo N` goes back N entries, and `gg redo`
  moves forward again when the last action was an undo. This also makes `gg discard`,
  `gg clear` and branch or tag changes revertible. The newest 100 entries are kept.
- Every mutating verb ends with a footer: `@ <sha> · ahead N · behind M`.
- `gg push` force-pushes on its own only when the force would drop nothing but your
  own commits, and no other remote branch holds them; the lease and
  `--force-if-includes` still guard the race. On anything else it reports the drift,
  lists the incoming commits, and asks: update, force-with-lease, or abort.
  Non-interactive runs refuse. A rewrite that is shared with another branch, or that
  drops someone else's commit, always asks.
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
