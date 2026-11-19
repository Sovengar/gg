# gg tools

List every external binary gg delegates to, with the path found on PATH and
MISSING for the ones that are not installed. Reach for it before `gg sim`,
`gg hunk` or `gg prdash`: those verbs run the binary and add nothing, so a
missing one is otherwise a silent exit.

## Usage

    gg tools

## Arguments

    (none)   any argument is ignored: the list is read from the registry, not
             from the command line.

## Flags

    -h, --help   this page

`-p`/`--print` and `-n`/`--dry` are read before the verb runs, but this one only
lists, so they change nothing. The verb exits 0 even when a tool is missing.

## What it does

1. Walks the `tool:` rows of the registry in registry order, so the list is
   stable and follows the categories: story, mzt, prdash, sim, hunk, tuicr, but,
   jj, lazygit, gitdash.
2. Pads the name column to the longest verb name, which is `lazygit` and
   `gitdash` at seven characters, then prints three columns per line: the verb
   in cyan, the binary it runs, and where that binary was found.
3. Names the binary from the registry row, so the row reads
   `gitlogue (external)` for `gg story` and `git-sim (external)` for `gg sim`. A
   built-in verb and a verb that runs git are not in this list.
4. Resolves each binary with the usual PATH lookup and prints the path it would
   run. A failed lookup prints MISSING instead, dimmed.
5. Runs no git command, so it works outside a repository, and writes nothing:
   no oplog entry, no footer.

Colour follows the rest of gg: `NO_COLOR=1` or `GG_COLOR=never` prints the list
without the escape codes, which is what a script wants.

## Examples

    gg tools                    every external tool and its state
    gg tools | grep MISSING     the ones you would have to install
    GG_COLOR=never gg tools     plain text, for a script

A sample of the shape, with `gitlogue` installed and `git-sim` not:

      story    gitlogue (external)  /usr/bin/gitlogue
      sim      git-sim (external)  MISSING

## Errors

None of its own, and none of the binaries are run: the list only looks them up.
The exit code is 0 whether or not everything is installed.

## Git equivalent

None: gg reads its own registry and the PATH.

## See also

    gg sim       render an operation, when git-sim is installed
    gg hunk      review a diff in Hunk
    gg prdash    the pull request dashboard
    gg hh -v     the parameter note behind every external verb
    gg help      the table every verb is listed in
