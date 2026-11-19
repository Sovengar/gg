# gg shell

Install, remove or report the fish wrapper around gg. A program cannot change
the shell that started it, so `gg go` prints a directory; the wrapper is what
turns that print into a `cd`. Reach for it once, when `gg go <branch>` should
leave you in the worktree instead of printing where it is.

## Usage

    gg shell
    gg shell show
    gg shell install [--force]
    gg shell uninstall

## Arguments

    (none), show   where the wrapper is, or that it is missing. Same output.
    install        write the fish function
    uninstall      delete it

The file is `gg.fish` under the fish functions directory, read from the user
config directory: `$XDG_CONFIG_HOME/fish/functions/gg.fish`, or
`~/.config/fish/functions/gg.fish` when that variable is unset.

## Flags

    -f, --force   with install: replace a gg.fish that is not the wrapper
    -h, --help    this page

Only the first word is read, and after `install` only the next one is read, as
`-f`/`--force`. Anything further is ignored rather than refused.

## What it does

1. Resolves the path in step 1 of Arguments. A path that cannot be built is an
   error, and nothing is read or written.
2. With no argument or with `show` it looks at the file and reports one of two
   lines, both exit 0: `installed at <path>`, or `not installed. Run gg shell
   install`.
3. `install` writes the function, creating `fish/functions` when it is missing.
   A file that exists and differs from the wrapper is refused unless `--force`
   is passed; the wrapper itself is reported as `already installed`.
4. `uninstall` removes the file and reports `removed <path>`. A file that is
   already gone is not an error.
5. The wrapper runs `command gg` for everything except one case: `gg go` with no
   `--` in the arguments. Only there does it capture the output, and only when
   the call succeeded and the output is a directory does it `cd`. The exit code
   is passed through either way.
6. `gg go <branch> -- <command>` is untouched, since a `--` puts the call back
   on the plain path. Every other verb, tables and pages included, runs as it
   always does.
7. The verb is `nogit` in the table and not mutating: no oplog entry, no footer.
   It writes one file and nothing else.

## Examples

    gg shell                    where the wrapper stands
    gg shell install            write ~/.config/fish/functions/gg.fish
    gg go side                  open a new session first: this now cds
    gg shell show               installed at /home/me/.config/fish/functions/gg.fish
    gg shell install --force    replace a gg.fish that was not gg's
    gg shell uninstall          remove it again
    gg -p shell install         print the gg call, write nothing

The wrapper itself, once installed:

    function gg --description 'gg, with a worktree-aware go'
        if test "$argv[1]" != go; or contains -- -- $argv
            command gg $argv
            return $status
        end
        set -l dir (command gg $argv)
        set -l rc $status
        if test $rc -eq 0 -a -d "$dir"
            cd $dir
        end
        return $rc
    end

## Errors

    gg shell: cannot find the fish config directory: <os error>
        The user config directory could not be resolved.
    gg shell: /home/me/.config/fish/functions/gg.fish exists and is not gg's wrapper. Use --force to replace it.
        Your own gg.fish is left alone unless you insist.
    gg shell: ls. Use install, uninstall or show.
        The first word was not one of the three; the whole argument list is
        echoed back.
    gg shell: <os error>
        Creating the directory or writing the file failed.

## Git equivalent

None: gg does it in Go. It writes one fish function file; git is not involved.

## See also

    gg go       the verb the wrapper cds for
    gg wt       the worktrees go walks to
    gg help     the table, which the wrapper passes through
