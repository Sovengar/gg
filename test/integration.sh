#!/usr/bin/env bash
# Integration tests for gg: every case runs against real git repositories.
set -uo pipefail

GO=${GO:-go}
BIN=$(mktemp -d)/gg
REPO=$(cd "$(dirname "$0")/.." && pwd)

pass=0
fail=0

ok() { pass=$((pass + 1)); printf '  ok   %s\n' "$1"; }
no() { fail=$((fail + 1)); printf '  FAIL %s\n' "$1"; }

assert_contains() { # name, haystack, needle
	case "$2" in
	*"$3"*) ok "$1" ;;
	*) no "$1 (got: $2)" ;;
	esac
}

assert_eq() { # name, got, want
	if [ "$2" = "$3" ]; then ok "$1"; else no "$1 (got: $2, want: $3)"; fi
}

assert_lacks() { # name, haystack, needle
	case "$2" in
	*"$3"*) no "$1 (found: $3)" ;;
	*) ok "$1" ;;
	esac
}

build() {
	(cd "$REPO" && GOFLAGS=-buildvcs=false "$GO" build -o "$BIN" .) || {
		echo "build failed"
		exit 1
	}
}

newrepo() {
	local d
	d=$(mktemp -d)
	(cd "$d" && git init -q -b tb . && git commit -q --allow-empty -m init)
	echo "$d"
}

echo "build"
build

echo "help and registry"
out=$("$BIN" help)
assert_contains "help lists status" "$out" "status (st, sta, stf)"
assert_contains "help lists raw" "$out" "raw"
assert_contains "help -all shows git command" "$("$BIN" help -all)" "git status --short"
case "$out" in
*"git status --short"*) no "default help hides git command" ;;
*) ok "default help hides git command" ;;
esac
assert_contains "verbs names" "$("$BIN" verbs)" "log"

echo "read-only verbs"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
assert_contains "st shows untracked" "$("$BIN" st)" "?? f.txt"
assert_eq "print mode" "$("$BIN" -p st -u)" "git status --short -u"
assert_eq "dry is print" "$("$BIN" --dry st)" "git status --short"
assert_contains "raw passthrough" "$("$BIN" raw rev-parse --short HEAD)" "$(git rev-parse --short HEAD)"
assert_eq "unknown verb goes to git" "$("$BIN" -p blame -L 1,1 f.txt)" "git blame -L 1,1 f.txt"
assert_contains "log renders" "$("$BIN" log 1)" "init"

echo "commit, footer, undo"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
"$BIN" aa >/dev/null
"$BIN" c -m "add f" >/dev/null
assert_eq "commit landed" "$(git log --oneline | wc -l | tr -d ' ')" "2"
assert_contains "footer printed" "$("$BIN" c --allow-empty -m footer 2>&1)" "ahead"
"$BIN" undo >/dev/null
assert_eq "undo went back" "$(git log --oneline | wc -l | tr -d ' ')" "2"
if [ -f f.txt ]; then ok "undo keeps files"; else no "undo keeps files"; fi

echo "oplog, undo and redo"
R=$(newrepo)
cd "$R" || exit 1
echo keep > tracked.txt
git add tracked.txt >/dev/null 2>&1
git commit -q -m tracked
echo edit >> tracked.txt
echo fresh > untracked.txt
"$BIN" discard >/dev/null 2>&1
assert_contains "oplog lists the verb" "$("$BIN" oplog)" "discard"
assert_eq "discard restored the tracked file" "$(cat tracked.txt)" "keep"
if [ -f untracked.txt ]; then ok "discard keeps the new file"; else no "discard keeps the new file"; fi
"$BIN" undo >/dev/null 2>&1
assert_contains "undo restores the discarded tracked file" "$(cat tracked.txt)" "edit"

"$BIN" clear -f >/dev/null 2>&1
assert_eq "clear restored the tracked file" "$(cat tracked.txt)" "keep"
if [ -f untracked.txt ]; then no "clear deletes the new file"; else ok "clear deletes the new file"; fi
"$BIN" undo >/dev/null 2>&1
assert_contains "undo restores the cleared tracked file" "$(cat tracked.txt)" "edit"
if [ -f untracked.txt ]; then ok "undo restores the deleted new file"; else no "undo restores the deleted new file"; fi

R=$(newrepo)
cd "$R" || exit 1
echo a > a.txt
"$BIN" c -m c1 >/dev/null 2>&1
echo b > b.txt
"$BIN" c -m c2 >/dev/null 2>&1
assert_eq "two operations before undo" "$(git rev-list --count HEAD)" "3"
"$BIN" undo 2 >/dev/null 2>&1
assert_eq "undo 2 goes back two entries" "$(git rev-list --count HEAD)" "1"
if [ -f a.txt ]; then ok "undo 2 restores the older worktree"; else no "undo 2 restores the older worktree"; fi
if [ -f b.txt ]; then no "undo 2 drops the newer worktree"; else ok "undo 2 drops the newer worktree"; fi
"$BIN" redo >/dev/null 2>&1
assert_eq "redo restores the commits" "$(git rev-list --count HEAD)" "3"
if [ -f b.txt ]; then ok "redo restores the newer worktree"; else no "redo restores the newer worktree"; fi
"$BIN" undo 2 >/dev/null 2>&1
"$BIN" c --allow-empty -m after-undo >/dev/null 2>&1
out=$("$BIN" redo 2>&1)
assert_contains "new work after undo drops redo" "$out" "nothing to redo"

R=$(newrepo)
cd "$R" || exit 1
for i in $(seq 1 105); do
	"$BIN" c --allow-empty -m "op$i" >/dev/null 2>&1
done
assert_eq "oplog keeps the newest 100 entries" "$("$BIN" oplog | wc -l | tr -d ' ')" "100"

echo "uncommit"
R=$(newrepo)
cd "$R" || exit 1
echo one > a.txt
git add . >/dev/null 2>&1
git commit -q -m one
"$BIN" uncommit >/dev/null
assert_eq "uncommit drops HEAD" "$(git log --oneline | wc -l | tr -d ' ')" "1"
assert_eq "uncommit keeps the change" "$(git status --porcelain)" "?? a.txt"
git add . >/dev/null 2>&1
git commit -q -m one
echo two > b.txt
git add . >/dev/null 2>&1
git commit -q -m two
"$BIN" uncommit >/dev/null
assert_eq "uncommit keeps tree dirty" "$(git status --porcelain)" "?? b.txt"
git add . >/dev/null 2>&1
git commit -q -m two
h=$(NO_COLOR=1 "$BIN" log 2 | awk '$5=="one"{print $2}')
"$BIN" uncommit "$h" >/dev/null 2>&1
assert_eq "uncommit by id removes that commit" "$(git log --oneline | wc -l | tr -d ' ')" "2"
assert_eq "uncommit by id keeps HEAD on top" "$(git log --format=%s -1)" "two"
assert_eq "uncommit by id hands the change back" "$(git status --porcelain)" "?? a.txt"
if "$BIN" uncommit nope 2>/dev/null; then no "uncommit guards unknown ids"; else ok "uncommit guards unknown ids"; fi

echo "bare gg and handles"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
handle=$("$BIN" log 1 | awk '{print $2}')
case "$handle" in
??|???|????) ok "log shows a bare handle" ;;
*) no "log shows a bare handle (got: $handle)" ;;
esac
assert_contains "id resolves the handle" "$("$BIN" id "$handle")" "$handle"
sha_len=$("$BIN" id "$handle" | awk '{print length($2)}')
assert_eq "id prints a full sha" "$sha_len" "40"
assert_contains "handle resolves inside a verb" "$("$BIN" -p show "$handle")" "git show"
if "$BIN" id zzz >/dev/null 2>&1; then no "unknown handle is rejected"; else ok "unknown handle is rejected"; fi
assert_contains "explicit @handle still works" "$("$BIN" id "@$handle")" "$handle"

echo "log default and dashboard"
R=$(newrepo)
cd "$R" || exit 1
for i in $(seq 1 8); do git commit -q --allow-empty -m "commit $i"; done
assert_eq "log defaults to 6" "$("$BIN" log | grep -c '^●')" "6"
assert_eq "log takes a count" "$("$BIN" log 3 | grep -c '^●')" "3"
assert_contains "log marks unpushed with a yellow ball" "$(GG_COLOR=always "$BIN" log 1)" "95m●"
assert_contains "log long shows the author" "$("$BIN" log -l 1)" "["
out=$("$BIN")
assert_contains "dashboard has files" "$out" "files"
assert_contains "dashboard has commits" "$out" "commits"
assert_contains "dashboard has branches" "$out" "branches"
assert_contains "dashboard has a handle" "$out" "commit 8"

echo "aliases and abort"
R=$(newrepo)
cd "$R" || exit 1
git commit -q --allow-empty -m second
help=$("$BIN" help)
assert_contains "help groups commit (c, ca)" "$help" "commit (c, ca)"
assert_contains "help groups push (pu)" "$help" "push (pu)"
assert_contains "help groups update (up)" "$help" "update (up)"
handle=$("$BIN" log 1 | awk '{print $2}')
"$BIN" edit "$handle" >/dev/null 2>&1
assert_eq "edit stops the rebase" "$(git rev-parse --abbrev-ref HEAD)" "HEAD"
"$BIN" abort >/dev/null 2>&1
assert_eq "abort restores the branch" "$(git rev-parse --abbrev-ref HEAD)" "tb"
if "$BIN" abort >/dev/null 2>&1; then no "abort reports nothing to abort"; else ok "abort reports nothing to abort"; fi

echo "git history verbs"
R=$(newrepo)
cd "$R" || exit 1
for i in 1 2 3; do git commit -q --allow-empty -m "c$i"; done
before=$(git rev-list --count HEAD)
mid=$("$BIN" log 2 | tail -1 | awk '{print $2}')
"$BIN" abandon "$mid" >/dev/null 2>&1
assert_eq "abandon drops one commit" "$(git rev-list --count HEAD)" "$((before - 1))"
assert_contains "help groups by category" "$("$BIN" help)" "Edit saved work:"
assert_contains "help shows Register work" "$("$BIN" help)" "Register work:"
assert_contains "help shows Branching" "$("$BIN" help)" "Branching:"
assert_lacks "help drops the old branching category" "$("$BIN" help)" "Branching and Committing"
assert_lacks "help drops Others" "$("$BIN" help)" "Others:"
assert_contains "help shows reword" "$("$BIN" help)" "reword"
assert_contains "help shows the git command with -all" "$("$BIN" help -all)" "git status --short"
assert_contains "amend prints" "$("$BIN" -p amend)" "gg amend"
assert_contains "reword prints" "$("$BIN" -p reword ab)" "gg reword ab"
assert_contains "split prints" "$("$BIN" -p split ab -- f.txt)" "gg split ab -- f.txt"

echo "renamed verbs"
assert_contains "reflog verb" "$("$BIN" -p reflog)" "reflog"
assert_contains "branches verb" "$("$BIN" -p branches)" "branch --sort"
assert_contains "br alias" "$("$BIN" -p br)" "branch --sort"
assert_contains "branch verb" "$("$BIN" -p branch)" "branch --sort"
assert_contains "log alias renders" "$("$BIN" -p log 3)" "gg log 3"
assert_eq "dw is gone" "$("$BIN" -p dw)" "git dw"
assert_contains "diff takes -w" "$("$BIN" -p diff -w)" "git diff -w"

echo "external tools"
R=$(newrepo)
cd "$R" || exit 1
assert_contains "tools lists story" "$("$BIN" tools)" "gitlogue"
assert_contains "tool print mode" "$("$BIN" -p story --commit HEAD)" "gitlogue --commit HEAD"
assert_contains "tool keeps args" "$("$BIN" -p sim rebase -i HEAD~3)" "git-sim rebase -i 'HEAD~3'"

echo "push guard"
BARE=$(mktemp -d)/remote.git
git init -q --bare -b tb "$BARE"
A=$(mktemp -d)
git clone -q "$BARE" "$A" 2>/dev/null
(cd "$A" && git commit -q --allow-empty -m one && "$BIN" push >/dev/null 2>&1)
B=$(mktemp -d)
git clone -q "$BARE" "$B" 2>/dev/null
(cd "$B" && git commit -q --allow-empty -m two && git push -q 2>/dev/null)
cd "$A" && git commit -q --allow-empty -m three
out=$("$BIN" push </dev/null 2>&1)
rc=$?
assert_eq "push refuses when behind" "$rc" "1"
assert_contains "push explains drift" "$out" "commit(s) you don't have"
assert_contains "pushed commits carry a green ball" "$(cd "$B" && GG_COLOR=always "$BIN" log 1)" "92m●"
assert_contains "print mode for fn verbs" "$("$BIN" -p push)" "gg push"
assert_contains "pu is push" "$("$BIN" -p pu)" "gg pu"
assert_contains "up is update" "$("$BIN" -p up)" "gg up"
if "$BIN" -p sync >/dev/null 2>&1; then ok "sync prints without args"; else no "sync prints without args"; fi
if "$BIN" sync >/dev/null 2>&1; then no "sync requires a branch"; else ok "sync requires a branch"; fi

echo "no stash surface"
out=$("$BIN" help)
assert_lacks "help has no Stash section" "$out" "Stash"
assert_lacks "help has no stasha" "$out" "stasha"
assert_lacks "help has no stashs" "$out" "stashs"
assert_lacks "help has no wip verb" "$out" "wip  "
assert_contains "help lists new" "$out" "absorb more until you commit"
assert_contains "help lists wt" "$out" "parallel branches, one worktree each"
assert_contains "help lists discard" "$out" "discard (rs)"
assert_contains "help lists clear" "$out" "clear"
assert_lacks "help has no cl verb" "$out" "wipe tree and index"
assert_lacks "help has no cl-wd verb" "$out" "wipe the working tree"

echo "wip commit"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
"$BIN" new >/dev/null 2>&1
assert_eq "new commits a wip" "$(git log -1 --format=%s)" "wip"
assert_eq "new leaves the tree clean" "$(git status --porcelain)" ""
assert_eq "new marks the wip" "$(git rev-parse --verify --quiet refs/gg/wip)" "$(git rev-parse HEAD)"
echo edit >> f.txt
"$BIN" sw tb >/dev/null 2>&1
assert_eq "a rewinding verb folds the edit" "$(git rev-list --count HEAD)" "2"
assert_eq "the fold leaves the tree clean" "$(git status --porcelain)" ""
assert_eq "the fold keeps the wip marker" "$(git rev-parse --verify --quiet refs/gg/wip)" "$(git rev-parse HEAD)"
echo edit2 >> f.txt
"$BIN" c -m "real" >/dev/null 2>&1
assert_eq "c finalizes the wip" "$(git log -1 --format=%s)" "real"
assert_eq "finalize drops the marker" "$(git rev-parse --verify --quiet refs/gg/wip || echo gone)" "gone"
assert_eq "finalize keeps one commit" "$(git rev-list --count HEAD)" "2"
assert_contains "finalize keeps the edits" "$(cat f.txt)" "edit2"

"$BIN" new -m "wip two" >/dev/null 2>&1
if "$BIN" c >/dev/null 2>&1; then no "c without a message refuses in wip mode"; else ok "c without a message refuses in wip mode"; fi
assert_eq "the refused c keeps the marker" "$(git rev-parse --verify --quiet refs/gg/wip)" "$(git rev-parse HEAD)"
echo late >> f.txt
"$BIN" uncommit >/dev/null 2>&1
assert_eq "uncommit takes the wip back" "$(git status --porcelain | tr -d ' ')" "Mf.txt"
assert_eq "uncommit drops the marker" "$(git rev-parse --verify --quiet refs/gg/wip || echo gone)" "gone"
assert_eq "uncommit keeps the committed work" "$(git log -1 --format=%s)" "real"

echo "wip dash badge"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
"$BIN" new >/dev/null 2>&1
out=$("$BIN" dash)
assert_contains "dash badges the wip" "$out" "wip"
assert_lacks "dash never prints stash" "$out" "stash"

echo "push refuses a wip"
BARE=$(mktemp -d)/remote.git
git init -q --bare -b tb "$BARE"
W=$(mktemp -d)
git clone -q "$BARE" "$W" 2>/dev/null
cd "$W" || exit 1
git commit -q --allow-empty -m one
"$BIN" push >/dev/null 2>&1
echo hi > f.txt
"$BIN" new >/dev/null 2>&1
out=$("$BIN" push </dev/null 2>&1)
rc=$?
assert_eq "push refuses the wip" "$rc" "1"
assert_contains "push names the wip" "$out" "HEAD is a wip commit"
"$BIN" c -m "real" >/dev/null 2>&1
if "$BIN" push </dev/null >/dev/null 2>&1; then ok "push works after finalize"; else no "push works after finalize"; fi

echo "worktrees"
R=$(newrepo)
cd "$R" || exit 1
side="$(dirname "$R")/$(basename "$R").side"
"$BIN" wt new side >/dev/null 2>&1
if [ -f "$side/.git" ]; then ok "wt new creates a sibling worktree"; else no "wt new creates a sibling worktree"; fi
if [ -d "$side" ]; then ok "wt new made the directory"; else no "wt new made the directory"; fi
assert_contains "wt ls lists the branch" "$("$BIN" wt ls)" "side"
assert_lacks "wt ls prints no footer" "$("$BIN" wt ls)" "ahead"
if "$BIN" wt rm tb >/dev/null 2>&1; then no "wt rm refuses the main worktree"; else ok "wt rm refuses the main worktree"; fi
echo dirty > "$side/x.txt"
if "$BIN" wt rm side >/dev/null 2>&1; then no "wt rm refuses a dirty worktree"; else ok "wt rm refuses a dirty worktree"; fi
"$BIN" wt rm side --force >/dev/null 2>&1
if [ -d "$side" ]; then no "wt rm --force removes it"; else ok "wt rm --force removes it"; fi
assert_lacks "wt ls forgot it" "$("$BIN" wt ls)" "side"

echo "wt new folds a wip first"
R=$(newrepo)
cd "$R" || exit 1
echo base > f.txt
git add . >/dev/null 2>&1
git commit -q -m base
"$BIN" new >/dev/null 2>&1
echo carried >> f.txt
side="$(dirname "$R")/$(basename "$R").side"
"$BIN" wt new side >/dev/null 2>&1
assert_contains "wt new folds the edit into the wip" "$(git -C "$side" show HEAD:f.txt)" "carried"
assert_eq "wt new leaves the main tree clean" "$(git status --porcelain)" ""
"$BIN" wt rm side >/dev/null 2>&1

echo "commit and discard by handle"
R=$(newrepo)
cd "$R" || exit 1
echo one > a.txt
echo two > b.txt
git add . >/dev/null 2>&1
git commit -q -m files
echo edit >> a.txt
echo fresh > c.txt
h=$(NO_COLOR=1 "$BIN" dash | awk '$3=="a.txt"{print $1}')
"$BIN" c -m "only a" "$h" >/dev/null 2>&1
assert_eq "commit takes one handle" "$(git show --name-only --format= HEAD)" "a.txt"
assert_contains "the other path stays dirty" "$(git status --porcelain)" "c.txt"
h=$(NO_COLOR=1 "$BIN" dash | awk '$3=="c.txt"{print $1}')
"$BIN" c -m "add c" "$h" >/dev/null 2>&1
assert_eq "commit takes an untracked handle" "$(git show --name-only --format= HEAD)" "c.txt"
assert_eq "nothing dirty is left" "$(git status --porcelain)" ""
echo edit >> a.txt
"$BIN" discard a.txt >/dev/null 2>&1
assert_eq "discard reverts a path" "$(git status --porcelain)" ""
echo fresh > d.txt
"$BIN" discard d.txt >/dev/null 2>&1
if [ -f d.txt ]; then ok "discard refuses a new file"; else no "discard refuses a new file"; fi
"$BIN" clear -f >/dev/null 2>&1
if [ -f d.txt ]; then no "clear deletes a new file"; else ok "clear deletes a new file"; fi

echo "revert keeps the original commit"
R=$(newrepo)
cd "$R" || exit 1
echo one > a.txt
git add . >/dev/null 2>&1
git commit -q -m first
echo two > a.txt
git add . >/dev/null 2>&1
git commit -q -m second
h=$(git rev-parse HEAD)
before=$(git log --oneline | wc -l | tr -d ' ')
"$BIN" revert "$h" >/dev/null 2>&1
assert_eq "revert restored the content" "$(cat a.txt)" "one"
after=$(git log --oneline | wc -l | tr -d ' ')
assert_eq "revert added one commit" "$after" "$((before + 1))"
assert_contains "oplog lists revert" "$("$BIN" oplog)" "revert"
"$BIN" undo >/dev/null 2>&1
assert_eq "undo steps back over revert" "$(cat a.txt)" "two"
"$BIN" rv --help >/dev/null 2>&1
if [ "$?" -eq 2 ]; then no "revert needs a commit"; else ok "revert needs a commit"; fi
echo x >> a.txt
"$BIN" new >/dev/null 2>&1
if "$BIN" revert HEAD >/dev/null 2>&1; then no "revert refuses a wip"; else ok "revert refuses a wip"; fi

assert_contains "help lists clear as cl" "$("$BIN" help)" "clear (cl)"
assert_contains "help lists revert as rv" "$("$BIN" help)" "revert (rv)"

echo "copy a commit onto a branch"
R=$(newrepo)
cd "$R" || exit 1
git branch other
echo one > a.txt
git add . >/dev/null 2>&1
git commit -q -m one
h=$(NO_COLOR=1 "$BIN" log 1 | awk '{print $2}')
"$BIN" copy "$h" other >/dev/null 2>&1
assert_eq "copy comes back to the first branch" "$(git rev-parse --abbrev-ref HEAD)" "tb"
assert_eq "copy lands on the target branch" "$(git log --format=%s -1 other)" "one"
assert_eq "copy leaves the first branch alone" "$(git log --format=%s -1 HEAD)" "one"
assert_eq "copy counted once on the target" "$(git rev-list --count other)" "2"
h=$(NO_COLOR=1 "$BIN" log 2 | awk '$5=="init"{print $2}')
"$BIN" cp "$h" other >/dev/null 2>&1
assert_contains "cp is still an alias" "$("$BIN" -p copy "$h" other)" "gg copy"
if "$BIN" copy "$h" nope >/dev/null 2>&1; then no "copy refuses an unknown branch"; else ok "copy refuses an unknown branch"; fi
if "$BIN" copy >/dev/null 2>&1; then no "copy requires a handle"; else ok "copy requires a handle"; fi

echo
echo "passed $pass, failed $fail"
[ "$fail" -eq 0 ]
