# gg changeId

Resolve a changeId to the sha behind it. It prints the full 40 characters, and
the local branch that points at the commit, so a token read in `gg log`,
`gg status`, `gg dash` or `gg diff` can be handed to git or to another tool.

## Usage

    gg changeId [changeId...]
    gg id [changeId...]

## Aliases

    id    the short form, identical in every way

## Arguments

    changeId...   one or more tokens. Any unique prefix of the commit's
                  reverse-hex changeId resolves; `@` is accepted as a prefix and
                  dropped. `zz` never resolves: it names the uncommitted area.
    (none)        HEAD

Only commits are resolved here. A file changeId or a branch changeId does not
find its commit, so this verb answers `matches no single commit` for them; the
verbs that take any kind of changeId are `gg discard` and the path verbs.

The arguments are read by the verb itself, not resolved before it runs, so the
printed sha is always the one behind the token you typed.

## Flags

    -h, --help   this page

`-p`/`--print` and `-n`/`--dry` change nothing: the verb only reads, and prints
its lines either way.

## What it does

1. With no argument it prints HEAD as a changeId and an abbreviated sha, the
   shape every gg footer uses:

        kd 641682b

   A repository with no commit has no HEAD to resolve, and the run stops.
2. With arguments it prints one line per token: the changeId, the full sha, and
   in parentheses the first local branch that points at that commit.

        kd 641682b7a6c27c7273086ea06574b451742d5792  (trunk)

3. The changeId column is the drawn id, the one the tree prints: the unique
   prefix padded to three characters, so the column keeps a steady width.
4. A token that resolves to nothing prints its error and the run keeps going
   with the next one, then exits 1. A token that fits two commits is ambiguous,
   which is the same error here.
5. Nothing is written: no oplog entry, no footer, no git command of its own.
   Reading the history is the whole verb.

## Examples

    gg changeId kd              the sha and the branch behind that id
    gg id @nan                  the same, with the prefix gg accepts
    gg changeId kd qz mx        three lines, one per token
    gg changeId                 HEAD, as id and short sha
    git show $(gg id kd | awk '{print $2}')   hand the sha to git

## Errors

    gg: no commits yet
        No argument was passed and HEAD does not resolve.
    gg: kd matches no single commit
        The token names no commit, or names more than one. An ambiguous prefix
        needs another character, as `gg log` prints it.

## Git equivalent

None: gg does it in Go. It mints the changeId from each commit's `Change-Id`
trailer, or from the sha when the trailer is missing, and prints the sha itself.
The two reads behind a line are:

    git log --branches --tags --remotes HEAD
            --format=%H%x1f%(trailers:key=Change-Id,valueonly) -n 2000
    git for-each-ref --points-at <sha> --format=%(refname:short) refs/heads

## See also

    gg log       the graph, with a changeId on every commit
    gg status    the tree, with a changeId on every dirty path
    gg show      print one commit, resolved the same way
    gg discard   the verb that takes a changeId of any kind
    gg hh -v     the changeId parameter note, next to every verb
