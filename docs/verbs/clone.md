# gg clone

Clone a repository, then colocate a jj repository on the new checkout, so the
git history and the jj working copy share one store. Reach for it when you want
a working copy that both tools can open.

## Usage

    gg clone <url> [dir] [flags]

## Arguments

    url    what `git clone` accepts: a path, a file:// or an ssh/git url
    dir    the checkout directory. Without it gg derives the name from the url:
           a trailing `/` is dropped, everything up to the last `/` or `:` is
           dropped, and a `.git` suffix is dropped too, unless the clone is
           bare.

## Flags

    --no-colocate   gg only: run `git clone`, then stop before jj. The flag is
                    removed from the arguments, so git never sees it.
    --bare          reaches `git clone`, and tells gg there is no working copy,
                    so jj stays out. The derived directory name keeps `.git`.
    --mirror        same decision: git builds a mirror clone, jj stays out
    -h, --help      this page, even when other arguments are present

Every other flag reaches `git clone` verbatim: `-b main`, `--depth 1`,
`--recurse-submodules`, `-o up` and the rest.

## What it does

1. Reads the positional arguments, skipping the value of every flag that takes
   one, so `gg clone -b main <url>` still finds the url.
2. With no url it prints `gg clone <url> [dir]: name the repository to clone`
   and exits 2, without calling git.
3. Runs `git clone` with every argument except `--no-colocate`, and returns
   git's exit code untouched when it fails.
4. Computes where git put the checkout with the rule above, then colocates it
   the way `gg init` does: bare prints `bare: <dir> (git only)`,
   `--no-colocate` prints `git only: <dir> (no jj)`, and a successful
   `jj git init --colocate <dir>` prints `colocated: <dir> (git + jj)`.
5. A missing jj prints
   `gg: git is done, but jj is not installed; <dir> stays git only` and exits 0.
   A failing colocation prints
   `gg: git is done, but jj could not colocate <dir>`, also with exit 0.

gg never edits the clone: the remote, the branch and every other setting are the
ones git wrote.

## Examples

    gg clone git@github.com:acme/api.git        checkout in ./api
    gg clone https://github.com/acme/api.git api    checkout in ./api (named)
    gg clone --depth 1 -b main <url> one        shallow, git takes both flags
    gg clone --no-colocate ../other             git only, no jj
    gg clone --bare /srv/repos/api.git          bare: ./api.git, no jj

## Errors

    gg clone <url> [dir]: name the repository to clone
        No url reached the positional arguments; exits 2, git is not called.
    gg: git is done, but jj is not installed; <dir> stays git only
        jj is missing from PATH; the checkout is complete, exit 0.
    gg: git is done, but jj could not colocate <dir>
        `jj git init --colocate` failed; the checkout is complete, exit 0.
        A failing `git clone` prints git's own message and returns git's code.

## Git equivalent

    git clone [flags] <url> [dir]
    jj git init --colocate <dir>

The second line only runs when jj is installed and the clone is neither bare nor
`--no-colocate`. jj must be on PATH; `gg tools` shows whether it is installed.

## See also

    gg init     the same two steps in an empty directory
    gg branch   the branches the clone brought down
    gg status   the tree of the fresh checkout
