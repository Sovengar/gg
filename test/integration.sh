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
assert_contains "help lists st" "$out" "st/sta/stf"
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

echo "uncommit"
R=$(newrepo)
cd "$R" || exit 1
"$BIN" c --allow-empty -m one >/dev/null
"$BIN" c --allow-empty -m two >/dev/null
"$BIN" uncommit 2 >/dev/null
assert_eq "uncommit 2 of 3" "$(git log --oneline | wc -l | tr -d ' ')" "1"
if "$BIN" uncommit 99 2>/dev/null; then no "uncommit guards count"; else ok "uncommit guards count"; fi

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
assert_contains "help groups c/commit" "$help" "c/commit"
assert_contains "help groups pu/push" "$help" "pu/push"
assert_contains "help groups up/update" "$help" "up/update"
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
assert_contains "help groups by category" "$("$BIN" help)" "history"
assert_contains "help shows reword" "$("$BIN" help)" "reword"
assert_contains "help shows the git command with -all" "$("$BIN" help -all)" "git status --short"
assert_contains "amend prints" "$("$BIN" -p amend)" "gg amend"
assert_contains "reword prints" "$("$BIN" -p reword ab)" "gg reword ab"
assert_contains "split prints" "$("$BIN" -p split ab -- f.txt)" "gg split ab -- f.txt"

echo "renamed verbs"
assert_contains "files verb" "$("$BIN" -p files)" "ls-files"
assert_contains "reflog verb" "$("$BIN" -p reflog)" "reflog"
assert_contains "branches verb" "$("$BIN" -p branches)" "branch --sort"
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

echo
echo "passed $pass, failed $fail"
[ "$fail" -eq 0 ]
