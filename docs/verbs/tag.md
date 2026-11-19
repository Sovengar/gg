# gg tag

Tags by version, or hands the call to git tag so create, delete and sign keep
working. Bare it lists every tag newest version first, with its message.

## Usage

    gg tag
    gg tag <name>
    gg tag <name> -m <message>
    gg tag -d <name>
    gg tag -s <name> [-m <message>]

## Aliases

    t     the same verb, one letter

## Arguments

    name   the tag to create, delete or inspect. Every argument goes to git, so
           a commit to tag, a glob for -l and a message for -m all work.

## Flags

    None of its own. These reach git tag unchanged:
    -m <message>   an annotated tag with that message
    -d             delete the tag
    -s             a signed tag
    -a             an annotated tag
    -l, --list     list matching tags
    -n             print the message next to each tag
    -v, --verify   verify a signed tag

The global -p/--print is read by gg first: it prints the git command and runs
nothing, and never reaches git tag.

## What it does

1. With no arguments it runs the listing call:

        git tag --sort=-v:refname -n -l

   Newest version first, so `v1.10.0` sorts above `v1.9.0`, each tag followed
   by its message.
2. With arguments it runs `git tag` plus those arguments verbatim, so create,
   delete, sign and verify are git's own.
3. A call made only of the listing flags -l, --list, -n, -v and --verify
   records nothing and prints no footer. A tag name or a pattern in the same
   call is not a listing call any more: gg records an oplog entry and prints the
   footer, even though git only lists.
4. Any other call is recorded in the oplog before it runs. Tags are part of the
   snapshot, so `gg undo` brings a deleted tag back, and a create is undoable
   too.

## Examples

    gg tag                    every tag, newest version first
    gg t v1.2.0               a lightweight tag on HEAD
    gg t v1.2.0 -m "release"  an annotated tag with a message
    gg t -d v1.2.0            delete it; gg undo brings it back
    gg t -s v1.2.0 -m "signed"   a signed tag, gpg configured
    gg t -l                   list the tags
    gg t -l 'v1.*'            list the matching tags
    gg -p t -d v1.2.0         print the git command, run nothing

## Errors

    None of its own: git tag owns the arguments and prints the failures,
    such as a tag that already exists or a missing signing key.

## Git equivalent

    git tag --sort=-v:refname -n -l
    git tag <args>

## See also

    gg undo     step back over a tag change
    gg oplog    the operations undo can reach
    gg reflog   where HEAD has been
    gg log      the commits the tags point at
