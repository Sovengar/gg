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

wipfile() { git rev-parse --git-path gg/wip; }
wipsha() { cat "$(wipfile)" 2>/dev/null || echo gone; }

build() {
	if (cd "$REPO" && GOFLAGS=-buildvcs=false "$GO" build -o "$BIN" .); then
		return 0
	fi
	echo "default Go toolchain failed; retrying with mise go@1.26.3" >&2
	(cd "$REPO" && GOFLAGS=-buildvcs=false mise exec go@1.26.3 -- go build -o "$BIN" .) || {
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
assert_contains "help lists status" "$out" "status"
assert_contains "help groups a shorthand under its verb" "$out" "status st"
assert_lacks "help has no branches alias" "$out" "branches"
assert_contains "help names the aliases verb first" "$out" "reword -> rename, describe"
assert_contains "help keeps variants off the canonical line" "$(NO_COLOR=1 "$BIN" help | sed -n 1p)" "branch br · changeId id"
assert_contains "params names the short status form" "$("$BIN" help -p)" "status -a sta"
assert_contains "params names the files status form" "$("$BIN" help -p)" "status -f stf"
assert_contains "params lists br -r" "$("$BIN" help -p)" "br -r"
assert_contains "params lists br -a" "$("$BIN" help -p)" "br -a"
assert_contains "params lists update --check" "$("$BIN" help -p)" "--check fetches, then reports"
assert_contains "help -all shows git command" "$("$BIN" help -all)" "git show"
case "$out" in
*"Overview of the workspace"*) no "default help hides descriptions" ;;
*) ok "default help hides descriptions" ;;
esac
case "$out" in
*"Rollback work"*) ok "default help keeps the categories" ;;
*) no "default help keeps the categories" ;;
esac
case "$out" in
*"git show"*) no "default help hides git command" ;;
*) ok "default help hides git command" ;;
esac
assert_contains "verbs names" "$("$BIN" verbs)" "log"

echo "read-only verbs"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
assert_contains "st shows untracked" "$("$BIN" st)" "A f.txt"
assert_contains "st draws the tree" "$("$BIN" st)" "[uncommitted]"
assert_eq "print mode" "$("$BIN" -p st -u)" "gg st -u"
assert_eq "dry is print" "$("$BIN" --dry st)" "gg st"
assert_eq "unknown verb goes to git" "$("$BIN" -p blame -L 1,1 f.txt)" "git blame -L 1,1 f.txt"
assert_contains "log renders" "$("$BIN" log 1)" "init"

echo "help lists the built-ins"
out=$("$BIN" help)
assert_contains "help lists gg" "$out" "gg"
assert_contains "help lists help" "$out" "help"
case "$out" in
*raw*) no "help dropped raw" ;;
*) ok "help dropped raw" ;;
esac

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

echo "oplog fixes"
R=$(newrepo)
cd "$R" || exit 1
echo base > base.txt
git add . >/dev/null 2>&1
git commit -q -m base
echo one > a.txt
"$BIN" c -m c1 >/dev/null 2>&1
"$BIN" undo >/dev/null 2>&1
echo EDITED >> base.txt
"$BIN" redo >/dev/null 2>&1
assert_eq "redo restores the undone target" "$(cat base.txt)" "base"
"$BIN" undo >/dev/null 2>&1
assert_contains "undo after redo returns the edit" "$(cat base.txt)" "EDITED"

R=$(newrepo)
cd "$R" || exit 1
echo keep > tracked.txt
git add . >/dev/null 2>&1
git commit -q -m tracked
echo edit >> tracked.txt
echo fresh > new.txt
"$BIN" discard >/dev/null 2>&1
"$BIN" undo >/dev/null 2>&1
status=$(git status --porcelain)
case "$status" in
"A  "*) no "undo leaves the restored tree unstaged (got: $status)" ;;
*) ok "undo leaves the restored tree unstaged" ;;
esac
assert_contains "undo leaves the tracked edit unstaged" "$status" " M tracked.txt"
assert_contains "undo leaves the new file untracked" "$status" "?? new.txt"
assert_contains "undo keeps the restored content" "$(cat tracked.txt)" "edit"

R=$(newrepo)
cd "$R" || exit 1
echo one > a.txt
"$BIN" c -m first >/dev/null 2>&1
echo two > b.txt
"$BIN" c -m second >/dev/null 2>&1
assert_eq "consecutive verbs snapshot twice" "$("$BIN" oplog | wc -l | tr -d ' ')" "2"
assert_eq "temporary index files are removed" "$(find .git -maxdepth 1 -name 'gg-tmp-index*' | wc -l | tr -d ' ')" "0"
assert_contains "oplog clear explains git gc" "$("$BIN" oplog clear)" "git gc --prune=now"
assert_eq "oplog clear empties the log" "$("$BIN" oplog)" "no operations yet"
if "$BIN" oplog >/dev/null 2>&1; then ok "empty oplog exits 0"; else no "empty oplog exits 0"; fi

R=$(mktemp -d)
cd "$R" || exit 1
git init -q -b tb .
out=$("$BIN" oplog)
rc=$?
assert_eq "oplog on a fresh repo exits 0" "$rc" "0"
assert_contains "oplog on a fresh repo says no operations" "$out" "no operations yet"

BARE=$(mktemp -d)/remote.git
git init -q --bare -b tb "$BARE"
R=$(mktemp -d)
git clone -q "$BARE" "$R" 2>/dev/null
cd "$R" || exit 1
git commit -q --allow-empty -m one
git push -q -u origin tb 2>/dev/null
# an entry whose state predates the published commit
echo work > w.txt
git add w.txt >/dev/null 2>&1
"$BIN" c -m two >/dev/null 2>&1
git commit -q --allow-empty -m published
git push -q 2>/dev/null
out=$("$BIN" undo 2>&1)
rc=$?
assert_eq "undo refuses to remove a published commit" "$rc" "1"
assert_contains "undo guard names the branch" "$out" "tb"
assert_contains "undo guard asks for force" "$out" "--force"
assert_eq "the refused undo leaves HEAD alone" "$(git log -1 --format=%s)" "published"
"$BIN" undo --force >/dev/null 2>&1
assert_eq "undo --force bypasses the guard" "$(git log -1 --format=%s)" "one"

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

echo "bare gg and changeIds"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
changeId=$("$BIN" log 1 | awk '{print $2}')
case "$changeId" in
??|???|????) ok "log shows a bare changeId" ;;
*) no "log shows a bare changeId (got: $changeId)" ;;
esac
assert_contains "changeId resolves the changeId" "$("$BIN" changeId "$changeId")" "$changeId"
assert_contains "id still resolves it" "$("$BIN" id "$changeId")" "$changeId"
sha_len=$("$BIN" id "$changeId" | awk '{print length($2)}')
assert_eq "id prints a full sha" "$sha_len" "40"
assert_contains "changeId resolves inside a verb" "$("$BIN" -p show "$changeId")" "git show"
if "$BIN" id zzz >/dev/null 2>&1; then no "unknown changeId is rejected"; else ok "unknown changeId is rejected"; fi
assert_contains "explicit @changeId still works" "$("$BIN" id "@$changeId")" "$changeId"

echo "log default and dashboard"
R=$(newrepo)
cd "$R" || exit 1
for i in $(seq 1 8); do git commit -q --allow-empty -m "commit $i"; done
assert_eq "log defaults to 6" "$("$BIN" log | grep -c '^●')" "6"
assert_eq "log takes a count" "$("$BIN" log 3 | grep -c '^●')" "3"
assert_contains "log marks unpushed with a yellow ball" "$(GG_COLOR=always "$BIN" log 1)" "95m●"
assert_contains "log long shows the author" "$("$BIN" log -l 1)" "["
out=$("$BIN")
assert_contains "dashboard has the uncommitted lane" "$out" "[uncommitted]"
assert_contains "dashboard closes on the common base" "$out" "(common base)"
assert_contains "dashboard prints the hints" "$out" "Hint: commits are listed newest first"
assert_contains "dash status keeps the tree" "$("$BIN" status)" "[uncommitted]"

echo "the three status views"
R=$(newrepo)
cd "$R" || exit 1
git checkout -q -b main
git commit -q --allow-empty -m "main work"
git checkout -q -b work main
for f in 1 2 3 4 5 6 7; do echo "$f" > "f$f.txt"; done
git add . >/dev/null 2>&1
git commit -q -m "seven files"
for b in b1 b2 b3 b4 b5 b6; do
	git checkout -q "$b" 2>/dev/null || git checkout -q -b "$b"
	git commit -q --allow-empty -m "$b work"
done
git checkout -q work
assert_eq "dashboard draws five lanes" "$("$BIN" | grep -c '^┊[╭├]┄')" "5"
assert_contains "dashboard always draws the default branch" "$("$BIN")" "[main]"
assert_contains "dashboard starts on the uncommitted lane" "$("$BIN" | head -1)" "[uncommitted]"
assert_lacks "st keeps only the worktree lanes" "$("$BIN" st)" "[b1]"
assert_lacks "st lists no files" "$("$BIN" st)" " more"
assert_contains "stf marks the capped file list" "$("$BIN" stf)" "+2 more"
assert_eq "stf lists five changed files" "$("$BIN" stf | grep -c '^┊   [AMD] ')" "5"
assert_eq "st -a lists every changed file" "$("$BIN" st -a | grep -c '^┊   [AMD] ')" "7"
assert_lacks "st -a lifts the file cap" "$("$BIN" st -a)" "+2 more"
assert_eq "sta is st -a" "$("$BIN" sta)" "$("$BIN" st -a)"
assert_eq "stf is st -f" "$("$BIN" stf)" "$("$BIN" st -f)"

R=$(newrepo)
cd "$R" || exit 1
for i in 1 2 3; do echo "$i" > "g$i.txt"; git add . >/dev/null 2>&1; git commit -q -m "c$i"; done
assert_eq "st shows the branch commits on the default branch" "$("$BIN" st | grep -c '^┊●')" "4"
assert_contains "stf lists the files of those commits" "$("$BIN" stf)" "A g3.txt"

echo "aliases and abort"
R=$(newrepo)
cd "$R" || exit 1
git commit -q --allow-empty -m second
help=$("$BIN" help)
assert_contains "help groups commit" "$help" "commit"
assert_contains "help groups push" "$help" "push"
assert_contains "help lists the commit shorthand" "$help" "commit c"
assert_contains "help lists the absorb shorthand" "$help" "absorb ab"
assert_contains "params lists the commit -a form" "$("$BIN" help -p)" "commit -a ca"
assert_contains "help lists the copy shorthand" "$help" "copy cp"
assert_contains "help lists the update shorthand" "$help" "update up"
assert_contains "help lists the push shorthand" "$help" "push p"
assert_contains "help groups update" "$help" "update"
changeId=$("$BIN" log 1 | awk '{print $2}')
"$BIN" edit "$changeId" >/dev/null 2>&1
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
assert_contains "help groups by category" "$("$BIN" help)" "Edit saved work"
assert_contains "help shows Register work" "$("$BIN" help)" "Register work"
assert_contains "help shows Branching" "$("$BIN" help)" "Branching"
assert_contains "help -all prints the categories with a colon" "$("$BIN" help -all)" "Register work:"
assert_lacks "help drops the old branching category" "$("$BIN" help)" "Branching and Committing"
assert_lacks "help drops Others" "$("$BIN" help)" "Others:"
assert_contains "help shows reword" "$("$BIN" help)" "reword"
assert_contains "help shows the git command with -all" "$("$BIN" help -all)" "git show"
assert_contains "amend prints" "$("$BIN" -p amend)" "gg amend"
assert_contains "reword prints" "$("$BIN" -p reword ab)" "gg reword ab"
assert_contains "split prints" "$("$BIN" -p split ab -- f.txt)" "gg split ab -- f.txt"

echo "renamed verbs"
assert_contains "reflog verb" "$("$BIN" -p reflog)" "reflog"
assert_eq "branches verb is gone" "$("$BIN" -p branches)" "git branches"
assert_contains "br alias" "$("$BIN" -p br trunk)" "gg br trunk"
assert_contains "branch verb" "$("$BIN" -p branch)" "gg branch"
assert_contains "log alias renders" "$("$BIN" -p log 3)" "gg log 3"
assert_eq "dw is gone" "$("$BIN" -p dw)" "git dw"
assert_contains "diff takes -w" "$("$BIN" -p diff -w)" "gg diff -w"

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
assert_contains "p is push" "$("$BIN" -p p)" "gg p"
assert_eq "pu is gone" "$("$BIN" -p pu)" "git pu"
assert_eq "rs is gone" "$("$BIN" -p rs)" "git rs"
assert_eq "fetch falls through to git" "$("$BIN" -p fetch)" "git fetch"
assert_eq "brr lists remotes" "$("$BIN" -p brr)" "git branch -r"
assert_eq "bra lists both" "$("$BIN" -p bra)" "git branch -a"
assert_contains "up is update" "$("$BIN" -p up)" "gg up"
if "$BIN" -p sync >/dev/null 2>&1; then ok "sync prints without args"; else no "sync prints without args"; fi
if "$BIN" sync >/dev/null 2>&1; then no "sync requires a branch"; else ok "sync requires a branch"; fi

echo "no stash surface"
out=$("$BIN" help)
assert_lacks "help has no Stash section" "$out" "Stash"
assert_lacks "help has no stasha" "$out" "stasha"
assert_lacks "help has no stashs" "$out" "stashs"
assert_lacks "help has no wip verb" "$out" "wip  "
assert_contains "help lists new" "$out" "new"
assert_contains "help lists wt" "$out" "wt"
assert_contains "help --all describes new" "$("$BIN" help --all)" "absorb more until you commit"
assert_contains "help --all describes wt" "$("$BIN" help --all)" "parallel branches, one worktree each"
assert_contains "help lists discard" "$out" "discard"
assert_contains "help lists the clear shorthand" "$out" "clear cl"
assert_contains "help lists the revert shorthand" "$out" "revert rv"
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
assert_eq "new marks the wip" "$(wipsha)" "$(git rev-parse HEAD)"
assert_eq "new leaves no shared marker" "$(git rev-parse --verify --quiet refs/gg/wip || echo gone)" "gone"
echo edit >> f.txt
"$BIN" br tb >/dev/null 2>&1
assert_eq "a rewinding verb folds the edit" "$(git rev-list --count HEAD)" "2"
assert_eq "the fold leaves the tree clean" "$(git status --porcelain)" ""
assert_eq "the fold keeps the wip marker" "$(wipsha)" "$(git rev-parse HEAD)"
echo edit2 >> f.txt
"$BIN" c -m "real" >/dev/null 2>&1
assert_eq "c finalizes the wip" "$(git log -1 --format=%s)" "real"
assert_eq "finalize drops the marker" "$(wipsha)" "gone"
assert_eq "finalize keeps one commit" "$(git rev-list --count HEAD)" "2"
assert_contains "finalize keeps the edits" "$(cat f.txt)" "edit2"

"$BIN" new -m "wip two" >/dev/null 2>&1
if "$BIN" c >/dev/null 2>&1; then no "c without a message refuses in wip mode"; else ok "c without a message refuses in wip mode"; fi
assert_eq "the refused c keeps the marker" "$(wipsha)" "$(git rev-parse HEAD)"
echo late >> f.txt
"$BIN" uncommit >/dev/null 2>&1
assert_eq "uncommit takes the wip back" "$(git status --porcelain | tr -d ' ')" "Mf.txt"
assert_eq "uncommit drops the marker" "$(wipsha)" "gone"
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
echo more >> f.txt
head_before=$(git rev-parse HEAD)
out=$("$BIN" push </dev/null 2>&1)
rc=$?
assert_eq "push refuses the wip" "$rc" "1"
assert_contains "push names the wip" "$out" "HEAD is a wip commit"
assert_eq "push leaves HEAD alone" "$(git rev-parse HEAD)" "$head_before"
assert_contains "push does not fold the tree" "$(git status --porcelain)" "f.txt"
"$BIN" c -m "real" >/dev/null 2>&1
if "$BIN" push </dev/null >/dev/null 2>&1; then ok "push works after finalize"; else no "push works after finalize"; fi
out=$("$BIN" new 2>&1)
assert_lacks "a fresh wip is not published" "$out" "already on"
git push -q origin tb >/dev/null 2>&1
out=$("$BIN" new 2>&1)
rc=$?
assert_eq "new refuses a pushed wip" "$rc" "1"
assert_contains "new names the pushed upstream" "$out" "already on"
echo later >> f.txt
pushed=$(git rev-parse HEAD)
"$BIN" new --force >/dev/null 2>&1
if git merge-base --is-ancestor "$pushed" HEAD; then no "new --force rewrites the pushed wip"; else ok "new --force rewrites the pushed wip"; fi
"$BIN" uncommit >/dev/null 2>&1

echo "new refuses what it cannot keep safe"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
if "$BIN" new --help >/dev/null 2>&1; then no "new refuses an unknown flag"; else ok "new refuses an unknown flag"; fi
assert_eq "the refused flag committed nothing" "$(git rev-list --count HEAD)" "1"
assert_eq "the refused flag kept the tree" "$(git status --porcelain | tr -d ' ')" "??f.txt"
if "$BIN" new -m >/dev/null 2>&1; then no "new refuses a bare -m"; else ok "new refuses a bare -m"; fi
mkdir -p .git/rebase-merge
out=$("$BIN" new 2>&1)
rc=$?
assert_eq "new refuses during a rebase" "$rc" "1"
assert_contains "new names the rebase" "$out" "rebase"
rmdir .git/rebase-merge

echo hi > f.txt
git add . >/dev/null 2>&1
git commit -q -m base
side="$R.side"
git worktree add -q "$side" -b side >/dev/null 2>&1
(cd "$side" && echo side > f.txt && git commit -qam side)
echo main > f.txt
git commit -qam main
git merge side >/dev/null 2>&1
out=$("$BIN" new 2>&1)
rc=$?
assert_eq "new refuses an unmerged tree" "$rc" "1"
assert_contains "new names the merge" "$out" "merge"
if [ -f .git/MERGE_HEAD ]; then ok "the merge stays open"; else no "the merge stays open"; fi
assert_contains "the file keeps the markers" "$(cat f.txt)" "<<<<<<<"
rm -f .git/MERGE_HEAD
out=$("$BIN" new 2>&1)
rc=$?
assert_eq "new refuses the unmerged entries" "$rc" "1"
assert_contains "new names the conflicts" "$out" "conflicts"
git reset --hard >/dev/null 2>&1
git worktree remove --force "$side" >/dev/null 2>&1

echo "a wip marker per worktree"
R=$(newrepo)
cd "$R" || exit 1
echo a > a.txt
git add . >/dev/null 2>&1
git commit -q -m a
for w in w1 w2; do git worktree add -q "$R.$w" -b "$w" >/dev/null 2>&1; done
(cd "$R.w1" && echo one >> a.txt && "$BIN" new -m w1 >/dev/null 2>&1)
w1head=$(git -C "$R.w1" rev-parse HEAD)
(cd "$R.w2" && echo two >> a.txt && "$BIN" new -m w2 >/dev/null 2>&1)
w1mark=$(cat "$(git -C "$R.w1" rev-parse --absolute-git-dir)/gg/wip" 2>/dev/null)
assert_eq "the second worktree keeps the first marker" "$w1mark" "$w1head"
w1first=$(git -C "$R.w1" rev-parse HEAD)
(cd "$R.w1" && echo more >> a.txt && "$BIN" new >/dev/null 2>&1)
if git -C "$R.w1" merge-base --is-ancestor "$w1first" HEAD; then
	no "the first worktree folds its own wip"
else
	ok "the first worktree folds its own wip"
fi

echo "worktrees"
R=$(newrepo)
cd "$R" || exit 1
side="$(dirname "$R")/$(basename "$R").side"
"$BIN" wt new side >/dev/null 2>&1
if [ -f "$side/.git" ]; then ok "wt new creates a sibling worktree"; else no "wt new creates a sibling worktree"; fi
if [ -d "$side" ]; then ok "wt new made the directory"; else no "wt new made the directory"; fi
assert_contains "wt ls lists the branch" "$("$BIN" wt ls)" "side"
assert_contains "dash draws a lane for the worktree" "$("$BIN" dash)" "[side]"
assert_contains "dash names the other worktree" "$("$BIN" dash)" "side"
assert_lacks "wt ls prints no footer" "$("$BIN" wt ls)" "ahead"
git worktree add -q "$R.otherwt" -b other 2>/dev/null
out=$("$BIN" wt new other 2>&1)
rc=$?
assert_eq "wt new refuses a busy branch" "$rc" "1"
assert_contains "wt new names the busy worktree" "$out" "checked out in"
git worktree remove --force "$R.otherwt" >/dev/null 2>&1
if "$BIN" wt rm tb >/dev/null 2>&1; then no "wt rm refuses the main worktree"; else ok "wt rm refuses the main worktree"; fi
echo dirty > "$side/x.txt"
if "$BIN" wt rm side >/dev/null 2>&1; then no "wt rm refuses a dirty worktree"; else ok "wt rm refuses a dirty worktree"; fi
"$BIN" wt rm side --force >/dev/null 2>&1
if [ -d "$side" ]; then no "wt rm --force removes it"; else ok "wt rm --force removes it"; fi
assert_lacks "wt ls forgot it" "$("$BIN" wt ls)" "side"

echo "the wip fold waits for a stopped operation"
R=$(newrepo)
cd "$R" || exit 1
echo hi > f.txt
"$BIN" new >/dev/null 2>&1
echo edit >> f.txt
head_before=$(git rev-parse HEAD)
mkdir -p .git/rebase-merge
"$BIN" update >/dev/null 2>&1
assert_eq "a stopped rebase blocks the fold" "$(git rev-parse HEAD)" "$head_before"
assert_contains "the tree stays dirty during a rebase" "$(git status --porcelain)" "f.txt"
rmdir .git/rebase-merge

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

echo "commit and discard by changeId"
R=$(newrepo)
cd "$R" || exit 1
echo one > a.txt
echo two > b.txt
git add . >/dev/null 2>&1
git commit -q -m files
echo edit >> a.txt
echo fresh > c.txt
h=$(NO_COLOR=1 "$BIN" dash | awk '$4=="a.txt"{print $2}')
"$BIN" c -m "only a" "$h" >/dev/null 2>&1
assert_eq "commit takes one changeId" "$(git show --name-only --format= HEAD)" "a.txt"
assert_contains "the other path stays dirty" "$(git status --porcelain)" "c.txt"
h=$(NO_COLOR=1 "$BIN" dash | awk '$4=="c.txt"{print $2}')
"$BIN" c -m "add c" "$h" >/dev/null 2>&1
assert_eq "commit takes an untracked changeId" "$(git show --name-only --format= HEAD)" "c.txt"
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

out=$("$BIN" help)
assert_contains "help lists clear" "$out" "clear"
assert_contains "help lists revert" "$out" "revert"

echo "old surface is gone"
out=$("$BIN" help)
assert_lacks "help has no Changes section" "$out" "Changes:"
assert_lacks "help has no cl-wd" "$out" "cl-wd"
assert_lacks "help has no cl-sa" "$out" "cl-sa"
if "$BIN" cl-wd >/dev/null 2>&1; then no "cl-wd fails"; else ok "cl-wd fails"; fi
if "$BIN" cl-sa >/dev/null 2>&1; then no "cl-sa fails"; else ok "cl-sa fails"; fi
if "$BIN" ia >/dev/null 2>&1; then no "ia fails"; else ok "ia fails"; fi
if "$BIN" aa >/dev/null 2>&1; then no "aa fails"; else ok "aa fails"; fi
assert_lacks "help has no hidden add" "$("$BIN" help -all)" "la patch"
if "$BIN" us >/dev/null 2>&1; then no "us fails"; else ok "us fails"; fi
assert_contains "cl-wd names discard" "$("$BIN" cl-wd 2>&1)" "gg discard"
if "$BIN" discard -a >/dev/null 2>&1; then no "discard -a fails"; else ok "discard -a fails"; fi
assert_contains "discard -a explains" "$("$BIN" discard -a 2>&1)" "no -a"
if "$BIN" clear -a >/dev/null 2>&1; then no "clear -a fails"; else ok "clear -a fails"; fi

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
if "$BIN" copy >/dev/null 2>&1; then no "copy requires a changeId"; else ok "copy requires a changeId"; fi

echo "branch switch and delete"
R=$(newrepo)
cd "$R" || exit 1
git branch side
"$BIN" br side >/dev/null 2>&1
assert_eq "br switches to a branch" "$(git rev-parse --abbrev-ref HEAD)" "side"
assert_eq "br never creates a branch" "$(git for-each-ref refs/heads --format='%(refname:short)' | wc -l | tr -d ' ')" "2"
if "$BIN" br nope >/dev/null 2>&1; then no "br refuses an unknown branch"; else ok "br refuses an unknown branch"; fi
assert_contains "br points at cbr" "$("$BIN" br nope 2>&1)" "gg cbr nope"
"$BIN" branch -c fresh >/dev/null 2>&1
assert_eq "branch -c creates and switches" "$(git rev-parse --abbrev-ref HEAD)" "fresh"
"$BIN" branch new oldform >/dev/null 2>&1
assert_eq "branch new still creates" "$(git rev-parse --abbrev-ref HEAD)" "oldform"
"$BIN" br tb >/dev/null 2>&1
recent=$(NO_COLOR=1 "$BIN" dbr side 2>/dev/null)
assert_eq "dbr deletes a merged branch" "$(git for-each-ref refs/heads/side --format='%(refname:short)')" ""
assert_eq "dbr keeps a backup ref" "$(git for-each-ref refs/gg/deleted/side --format='%(refname)')" "refs/gg/deleted/side"
assert_contains "dbr lists the newest branches" "$recent" "branches (5 newest"
assert_contains "dbr keeps the default branch in view" "$recent" "tb"
assert_eq "dbr lists a branch once" "$(printf '%s\n' "$recent" | grep -c tb)" "1"
git branch unmerged side >/dev/null 2>&1
if "$BIN" br -d unmerged >/dev/null 2>&1; then no "br -d refuses unmerged work"; else ok "br -d refuses unmerged work"; fi
"$BIN" br -D unmerged >/dev/null 2>&1
assert_eq "br -D forces the delete" "$(git for-each-ref refs/heads/unmerged --format='%(refname:short)')" ""
git branch forced side >/dev/null 2>&1
"$BIN" Dbr forced >/dev/null 2>&1
assert_eq "Dbr forces the delete" "$(git for-each-ref refs/heads/forced --format='%(refname:short)')" ""
if "$BIN" dbr tb >/dev/null 2>&1; then no "dbr refuses the current branch"; else ok "dbr refuses the current branch"; fi

echo "branch changeIds stay apart on one commit"
R=$(newrepo)
cd "$R" || exit 1
git branch twin
changeIds=$(NO_COLOR=1 "$BIN" br | awk '{if ($1 == "*") {h = $2; n = $3} else {h = $1; n = $2} if (n == "tb" || n == "twin") print h}' | sort -u)
count=$(printf '%s\n' "$changeIds" | grep -c .)
if [ "$count" -eq 2 ]; then ok "two branches on one commit get different changeIds"; else no "two branches on one commit get different changeIds (got: $changeIds)"; fi
twin=$(NO_COLOR=1 "$BIN" br | awk '$1 == "twin" || $2 == "twin" {print ($1 == "*") ? $2 : $1}')
"$BIN" br "$twin" >/dev/null 2>&1
assert_eq "br switches by branch changeId" "$(git rev-parse --abbrev-ref HEAD)" "twin"
assert_contains "br list shows a changeId per branch" "$(NO_COLOR=1 "$BIN" br | grep twin)" "$twin"

echo "a changeId survives a rewrite"
R=$(newrepo)
cd "$R" || exit 1
"$BIN" hooks >/dev/null 2>&1
echo one > a.txt
git add . >/dev/null 2>&1
git commit -q -m one
cid=$(NO_COLOR=1 "$BIN" log 1 | awk '{print $2}')
sha1=$(git rev-parse HEAD)
echo edit >> a.txt
git add . >/dev/null 2>&1
git commit -q --amend --no-edit
sha2=$(git rev-parse HEAD)
cid2=$(NO_COLOR=1 "$BIN" log 1 | awk '{print $2}')
if [ "$sha1" != "$sha2" ]; then ok "amend moved the sha"; else no "amend moved the sha"; fi
assert_eq "the changeId survived the amend" "$cid2" "$cid"
echo two > b.txt
git add . >/dev/null 2>&1
git commit -q -m two
git reset -q --hard HEAD~1
git cherry-pick -q "$sha2" >/dev/null 2>&1
cid3=$(NO_COLOR=1 "$BIN" log 1 | awk '{print $2}')
assert_eq "the changeId survived a cherry-pick" "$cid3" "$cid"

echo "oplog restores refs"
R=$(newrepo)
cd "$R" || exit 1
git branch gone
gone_sha=$(git rev-parse refs/heads/gone)
"$BIN" dbr gone >/dev/null 2>&1
assert_eq "dbr removed the branch" "$(git for-each-ref refs/heads/gone --format='%(refname)')" ""
"$BIN" undo >/dev/null 2>&1
assert_eq "undo restores a deleted branch at the same sha" "$(git rev-parse refs/heads/gone)" "$gone_sha"
assert_contains "undo lists the restored branch" "$(git branch --format='%(refname:short)')" "gone"
assert_eq "undo drops the deletion backup" "$(git for-each-ref refs/gg/deleted/gone --format='%(refname)')" ""

"$BIN" cbr created >/dev/null 2>&1
assert_eq "cbr created the branch" "$(git rev-parse --abbrev-ref HEAD)" "created"
"$BIN" undo >/dev/null 2>&1
assert_eq "undo removes a created branch" "$(git for-each-ref refs/heads/created --format='%(refname)')" ""
assert_eq "undo returns after removing a created branch" "$(git rev-parse --abbrev-ref HEAD)" "tb"


git branch other
"$BIN" br other >/dev/null 2>&1
assert_eq "br switched branches" "$(git rev-parse --abbrev-ref HEAD)" "other"
"$BIN" undo >/dev/null 2>&1
assert_eq "undo returns to the original branch" "$(git rev-parse --abbrev-ref HEAD)" "tb"

"$BIN" br new added >/dev/null 2>&1
assert_eq "br new created the branch" "$(git rev-parse --abbrev-ref HEAD)" "added"
"$BIN" undo >/dev/null 2>&1
assert_eq "undo removes a branch made by br new" "$(git for-each-ref refs/heads/added --format='%(refname)')" ""
assert_eq "undo returns after removing a br new branch" "$(git rev-parse --abbrev-ref HEAD)" "tb"

echo "update --check"
R=$(newrepo)
git init -q --bare "$R-upstream.git"
cd "$R" || exit 1
git remote add origin "$R-upstream.git"
git push -q -u origin tb
out=$("$BIN" up --check)
assert_contains "check reports behind and ahead" "$out" "0 behind · 0 ahead"
assert_contains "check calls the rebase clean" "$out" "rebase: clean"
before=$("$BIN" oplog | wc -l)
"$BIN" up --check >/dev/null 2>&1
assert_eq "check records no oplog entry" "$("$BIN" oplog | wc -l)" "$before"
C=$(mktemp -d)
git clone -q -b tb "$R-upstream.git" "$C"
(cd "$C" && git commit -q --allow-empty -m other && git push -q origin tb)
out=$("$BIN" up --check)
assert_contains "check counts the upstream commit" "$out" "1 behind"
assert_eq "check leaves HEAD alone" "$(git log -1 --format=%s)" "init"

echo
echo "passed $pass, failed $fail"
[ "$fail" -eq 0 ]
