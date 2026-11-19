# gg remote

Print the configured remotes, or hand the arguments to git so `add`, `rename`,
`rm` and `set-url` keep working. The remotes live in git config, not in a ref.

## Usage

    gg remote [git remote args...]

## Arguments

    (none)                     every remote: name, fetch url, push url
    add|rename|rm|set-url|show|prune|update ...   reach `git remote` verbatim

## Flags

    -p, --print   print the call and run nothing (a gg-wide flag)
    -h, --help    this page

## What it does

1. With no argument it runs `git remote -v`, which prints each name twice, once
   for fetch and once for push.

       origin  /tmp/up.git (fetch)
       origin  /tmp/up.git (push)

2. With arguments it runs `git remote <args>`, so the write forms git offers are
   all reachable without a second verb:

       gg remote add up /tmp/up.git          add one
       gg remote rename up upstream          rename it
       gg remote set-url up /tmp/other.git   repoint it
       gg remote rm up                       drop it
       gg remote prune origin                prune stale remote branches

3. Nothing is recorded in the oplog: a remote is a config entry, not a ref, so
   `gg undo` cannot bring a removed remote back. Re-add it by hand or with
   `gg remote add`.
4. Not a mutating verb for gg either, so no footer is printed.

## Examples

    gg remote                   names and urls
    gg remote add up /tmp/up.git
    gg remote set-url up /tmp/other.git
    gg remote -v                also verbatim: fetch and push urls for every name
    gg remote                   after rm: the name is gone, and undo cannot bring it back
    gg -p remote                print "gg remote" and run nothing

## Errors

None of its own. A remote git rejects, and a missing name, answer with git's
message and git's exit code.

## Git equivalent

    git remote -v                 no arguments
    git remote <args>             with arguments

## See also

    gg push    publish to a remote
    gg update  fetch every remote, then rebase
    gg fork    a parallel worktree, not a remote
    gg clone   a clone, with a colocated jj repo
