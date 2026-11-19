# gg init

Create a git repository where you are, then colocate a jj repository on the
same directory, so the git history and the jj working copy share one store.
Reach for it when a project starts from nothing and you want both tools on it.

## Usage

    gg init [flags] [dir]

## Arguments

    dir    the directory to create the repository in. It reaches `git init` as
           its own path argument, and gg reads the same name to know what to
           colocate. Without it the repository is the current directory.

## Flags

    --no-colocate   gg only: run `git init`, then stop before jj. The flag is
                    removed from the arguments, so git never sees it.
    --bare          reaches `git init`, and tells gg there is no working copy,
                    so jj stays out
    -h, --help      this page, even when other arguments are present

Every other flag reaches `git init` verbatim, for example `-b trunk`.

## What it does

1. Runs `git init` with every argument except `--no-colocate`. A failing git
   returns git's own exit code, untouched.
2. Reads the directory: the first positional argument, else `.`.
3. Asks git whether that repository is bare
   (`git -C <dir> rev-parse --is-bare-repository`). A bare one prints
   `bare: <dir> (git only)` and jj is never called.
4. With `--no-colocate` it prints `git only: <dir> (no jj)` and stops.
5. With no `jj` on PATH it prints
   `gg: git is done, but jj is not installed; <dir> stays git only` and exits 0:
   the git repository exists, only the second half is missing.
6. Otherwise it runs `jj git init --colocate <dir>`. Success prints
   `colocated: <dir> (git + jj)`; a failure prints
   `gg: git is done, but jj could not colocate <dir>`, also with exit 0.

The closing note is dim and always says what the directory now holds. gg adds
no config of its own: every setting is the one git and jj write.

## Examples

    gg init                    git here, then a colocated jj repo
    gg init --no-colocate      plain git, no jj
    gg init --bare             a bare repository, no working copy, no jj
    gg init -b trunk           git takes -b, jj is colocated afterwards
    gg init /srv/repo          create the repository at that path
    gg -p init                 print mode: prints `gg init`, runs nothing

## Errors

    gg: git is done, but jj is not installed; <dir> stays git only
        jj is missing from PATH; the git repository is complete, exit 0.
    gg: git is done, but jj could not colocate <dir>
        `jj git init --colocate` failed; the git repository is untouched,
        exit 0.
        A failing `git init` prints git's own message and returns git's code.

## Git equivalent

    git init [flags] [dir]
    jj git init --colocate <dir>

The second line only runs when jj is installed and the repository is neither
bare nor `--no-colocate`.

## See also

    gg clone    the same two steps on an existing repository
    gg status   see what the fresh repository holds
    gg commit   save the first change
