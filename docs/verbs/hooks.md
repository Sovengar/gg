# gg hooks

Install the `commit-msg` hook that gives every new commit a `Change-Id`
trailer. That trailer is what keeps a commit's changeId stable across an amend,
a rebase or a squash, so run this once per clone before you start rewriting.

## Usage

    gg hooks

## Arguments

    (none)   every argument is ignored. The verb writes the hook where git
             already looks for it, so there is nothing to name.

## Flags

    -p, --print   print the gg call and write nothing
    -h, --help    this page

## What it does

1. Refuses outside a repository, because there is no `.git` to write into.
2. Refuses when `core.hooksPath` points somewhere else
   (`git config --get core.hooksPath`), since a hook written to `.git/hooks`
   would never run. The setting wins, so install the same script there yourself.
3. Locates the directory with `git rev-parse --git-path hooks`, joining a
   relative answer with the worktree root.
4. A `commit-msg` file that is already there and does not carry gg's marker line
   is left alone: the verb will not overwrite someone else's hook. Its own hook
   is rewritten, so a second run is harmless.
5. Writes `<hooks>/commit-msg` with mode 0755 and prints `installed <path>`.
6. The script is `/bin/sh`. It exits 0 when the message already holds a
   `Change-Id:` line; otherwise it appends one line, the trailer with forty hex
   characters read from `/dev/urandom`:

        Change-Id: I<40 hex characters>

   `grep`, `od`, `tr` and `printf` come from the base system, so nothing else
   has to be installed.
7. The verb is mutating in the table: it records an oplog entry and prints the
   footer, even though no commit and no ref changes. `gg undo` steps back over
   the entry, not over the hook file.

The hook runs on `git commit`, which means every commit gets the trailer: the
ones gg makes with `gg c`, and the ones a tool or your editor makes.

## Examples

    gg hooks                    install the hook, print its path
    git commit -m "feat: x"     the hook adds the trailer
    git log -1 --format=%B      the message, Change-Id included
    gg hooks                    again: same script, rewritten, still fine
    gg -p hooks                 print the call, write nothing

## Errors

    gg: not inside a git repository
        There is no repository to install into.
    gg: core.hooksPath is "/home/me/hooks", install the hook there yourself
        A custom hooks directory wins over .git/hooks; gg does not write there.
    gg: cannot locate the hooks directory
        `git rev-parse --git-path hooks` gave nothing.
    gg: /home/me/gg/.git/hooks/commit-msg already exists and is not mine; leaving it alone
        Another hook owns the file. Remove or merge it by hand.
    gg: cannot create <dir>: <os error>
        The hooks directory could not be created.
    gg: cannot write <path>: <os error>
        The script could not be written, usually a permission problem.

## Git equivalent

None: gg does it in Go. It reads `git rev-parse --git-path hooks` and writes the
`commit-msg` file itself; git runs that file on `git commit`.

## See also

    gg changeId   the id the hook keeps stable
    gg amend      a rewrite the trailer survives
    gg rebase     -i is the other rewrite it survives
    gg c          the commit that carries the trailer
    gg tools      the external tools, none of which is needed here
