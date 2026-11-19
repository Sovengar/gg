# gg describe

Edit the message of one commit and leave its content alone. The `Change-Id`
trailer stays, so the commit keeps its changeId while its sha moves. `reword` and
`rename` are the same verb.

## Usage

    gg describe <changeId>
    gg desc <changeId>
    gg reword <changeId>
    gg rename <changeId>

## Aliases

    desc             the shorthand, one row of typing less
    reword, rename   the alternate names the table prints as aliases

## Arguments

    changeId   the commit whose message changes. Resolved before the verb runs,
               so a commit id, a branch name or a sha all work.

Only the first argument is used: later ones are ignored, so a typo after the
changeId changes nothing.

## Flags

gg describe has no flag of its own. The message is edited in the editor git
opens. `-h` prints this page, `-p`/`-n` print the call instead of running it.

## What it does

1. Refuses with no changeId.
2. Needs git 2.56 or newer, then runs `git history reword <sha>`: git reopens
   that commit's message and replays everything above it.
3. Inside a wip commit the uncommitted edits are folded into the wip first, the
   way every history-moving verb does.

The rewrite is recorded in the oplog, so `gg undo` restores the old messages and
shas, and the footer shows the new HEAD. Every alias reaches the same verb, and
`gg describe -h`, `gg h reword` and `gg h rename` print this page.

## Examples

    gg describe 4d1f0a7           edit that commit's message
    gg desc kd                    the same, shorter
    gg reword ab                  the alias, same page
    gg rename ab                  the other alias
    gg -p describe ab             print the call, run nothing

## Errors

    gg describe <commit>
        The arguments were empty, so there is no commit to reword.
    gg: reword needs git >= 2.56 (git history reword)
        This git has no history command, so gg has nothing to reword with.

git reports the rest: an aborted editor leaves the history untouched.

## Git equivalent

    git history reword <commit>          git >= 2.56

## See also

    gg amend      change the content of a commit instead
    gg squash     fold a commit into its parent
    gg log        read the messages you are editing
    gg discard    drop the commit, message and all
    gg hooks      the Change-Id hook that keeps the changeId stable
