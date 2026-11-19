# gg help

The compact verb table: one line of names per category, the shorthand next to
each name, and the alias line underneath. Reach for it to find a verb; add a
verb name and the same word opens that verb's page instead.

## Usage

    gg help
    gg h
    gg help -v
    gg help <verb>
    gg -h

## Aliases

    h     the short form, identical in every way

## Arguments

    <verb>   one verb name, abbreviation or alias: `help commit`, `h rename`,
             `help st`. The first bare word is the target; later words are
             ignored.
    (none)   the compact table

## Flags

    -v, --verbose   after `help`: one row per verb with its description
    -h, --help      this page. A bare `gg -h`, with no verb, prints the compact
                    table instead.

`-p`, `--print`, `-n` and `--dry` are not read here. The help words parse their
own flags before gg reads the gg-wide ones, so `gg help -p` prints the table,
and nothing is printed instead of run.

## What it does

1. Reads the arguments left to right: `-v` or `--verbose` sets the verbose view,
   `-h` or `--help` sets the target to the help word itself (unless a target came
   first), and the first word that does not start with `-` becomes the target.
2. A target wins over everything else: `gg help commit`, `gg h rename` and
   `gg st -h` all print one verb's page, and `gg help commit -v` still prints
   the page, with no description list. A hidden verb has a page too:
   `gg help dash` prints it, even though the table never names it.
3. With no target it prints the compact table. Each category is a dim label and
   its cells, and each cell is the verb with its shorthand, or the
   `display:` override from the registry when a row has one:

        Inspection  status st · log l · diff · show sh · branch br · story

   Lines wrap at `COLUMNS`, then the terminal width, falling back to 100
   columns, and the items are separated by ` · `.
4. Rows marked hidden are left out, so `dash` never appears. A variant row in
   the same category rides on its parent, so `commit -e` and `br -lr` are not
   cells of their own: `gg hh` merges them.
5. After the categories comes the alias line, aliases verb first, joined and
   wrapped the same way:

        describe -> reword, rename

6. The footer names the next view: `gg h -v  adds a description per verb, gg hh
   the parameters`.
7. With `-v` the table becomes one row per verb: the name column, padded to the
   longest name and capped at 34 columns, then the registry description. The
   footer there is `gg hh -v  adds the parameters and their hints`.
8. A word that is not a verb prints the fallback and exits 2, pointing at git
   and at the table.

The categories come from the registry, in its order: Setup, Inspection, Rollback
work, Register work, Edit saved work, Branching, Operations, Remote, Review,
Others. The verb writes nothing: no oplog entry, no footer, no git command.

## Examples

    gg help                  the compact table
    gg h                     the same
    gg help -v               one row per verb, with its description
    gg help commit           the commit page
    gg h rename              the describe page, reached through its alias
    gg help st               the status page, reached through its shorthand
    gg help dash             a hidden verb's page
    gg -h                    the compact table, from the flag

## Errors

    gg: no page for "nosuchverb".
    An unknown verb reaches git unchanged: gg nosuchverb <args>
    See every verb: gg help
        The word names no verb. All three lines go to stderr and the exit code
        is 2.

## Git equivalent

None: gg reads its own registry and prints the table itself.

## See also

    gg hh      the table with the flag variants merged into their verb
    gg hh -v   the parameters and their hints
    gg commit  a page, reachable as `gg help commit`
    gg         the dashboard, which the table lists as the empty command line
