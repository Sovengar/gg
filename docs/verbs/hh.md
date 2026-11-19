# gg hh

The table with the flag variants merged into the verb they belong to: one cell
reads `status st, -a sta, -f stf` instead of listing the variants as if they
were verbs. Add `-v` and the names give way to the parameters each verb takes.

## Usage

    gg hh
    gg hh -v
    gg hh <verb>

## Arguments

    <verb>   one verb name, abbreviation or alias. The first bare word is the
             target and opens that verb's page, exactly as `gg help <verb>` does.
    (none)   the merged table

## Flags

    -v, --verbose   one row per verb with its parameters, instead of the table
    -h, --help      this page

The word `hh` is not an abbreviation of `help`: it is its own entry, and it
reaches the same reader with the merged view asked for. `-p`, `--print`, `-n` and
`--dry` are not read here; the help words parse their own flags.

## What it does

1. Reads the arguments the way `gg help` does: `-v` or `--verbose`, `-h` or
   `--help`, and the first bare word as the target.
2. A target wins: `gg hh commit` prints the commit page and no table, and the
   exit code is that page's.
3. Otherwise it prints the compact table with the variants folded in. Every
   variant row in the same category as its parent contributes the part of its
   display that follows the parent's name, so the parent's cell grows a note:

        Inspection  status st, -a sta, -f stf · log l · diff · ...
        Register work  commit c, -e ce · fpr · new · empty · copy cp · absorb ab

   A variant in another category stays a row of its own, because a category
   boundary has to show in the table.
4. Hidden rows are left out, and so is `dash`. The alias line and the wrapping
   are the ones `gg help` uses: aliases verb first, items joined by ` · `, lines
   broken at `COLUMNS`, then the terminal width, falling back to 100 columns.
5. With `-v` it prints one row per verb that has a parameter note or is a
   variant row: the name column, then the registry hint. A variant with no hint
   of its own reads `same verb, other flag`. This is the only view where a
   variant gets a line of its own, so the rows `br -lr` and `br -lm` read their
   notes here and nowhere else.
6. The footers point at the neighbouring view: `gg hh -v  adds the parameters and
   their hints`, and, in the verbose one, `gg h -v  adds a description per
   verb`.
7. It writes nothing: no oplog entry, no footer, no git command. The four views
   stand in a ladder: `gg h`, `gg hh`, `gg h -v`, `gg hh -v`.

## Examples

    gg hh                  the table, one cell per verb, variants merged
    gg hh -v               the parameters and their hints, one line per verb
    gg hh commit           the commit page
    gg hh -h               this page
    gg help                the same table with the variants kept apart

## Errors

    gg: no page for "nosuchverb".
    An unknown verb reaches git unchanged: gg nosuchverb <args>
    See every verb: gg help
        The target names no verb. All three lines go to stderr, exit 2.

## Git equivalent

None: gg reads its own registry and prints the table itself.

## See also

    gg help      the compact table, variants kept apart
    gg h -v      a description per verb
    gg commit    a page, reachable as `gg hh commit`
    gg status    the verb whose three cells merge first
