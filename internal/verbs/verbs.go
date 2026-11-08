// Package verbs resolves gg verbs to either a git argv or a Go function.
package verbs

import (
	_ "embed"
	"strconv"
	"strings"

	"gg/internal/dash"
	"gg/internal/diffview"
	"gg/internal/hist"
	"gg/internal/oplog"
	"gg/internal/ops"
	"gg/internal/ui"
)

//go:embed verbs.tsv
var registry string

// Entry is one gg verb. Fn means a Go function, Tool means an external
// binary, anything else runs git with Snippet + args.
type Entry struct {
	Name        string   // canonical name, the first token
	Label       string   // names as written in the registry, e.g. "commit|c|=reword|rename"
	Abbrevs     []string // short forms: types on the verb line, e.g. c
	Aliases     []string // other names: types on the aka line, e.g. rename
	Snippet     string
	Description string
	Hint        string // grey parameter note shown after the description
	Category    string
	Tool        string
	Hidden      bool
	Variant     bool   // a flag on another verb, listed by gg hh only
	DisplayOnly bool   // a help row with no name to type, e.g. branch -D
	NoGit       bool   // no git equivalent behind the verb
	Shown       string // name column override in help, e.g. status (st, sta, stf)
	Fn          func(args []string) int
}

// IsFunc reports whether the verb is implemented in Go instead of a git call.
func (e Entry) IsFunc() bool { return e.Fn != nil }

// IsTool reports whether the verb delegates to an external binary.
func (e Entry) IsTool() bool { return e.Tool != "" }

// Argv returns the git arguments for a plain entry, followed by user args.
func (e Entry) Argv(rest []string) []string {
	argv := strings.Fields(e.Snippet)
	return append(argv, rest...)
}

var functions = map[string]func([]string) int{
	"log":          hist.Run,
	"reword":       ops.Reword,
	"split":        ops.Split,
	"amend":        ops.Amend,
	"dash":         dash.Run,
	"tree":         dash.Tree,
	"treeFull":     dash.TreeFull,
	"treeAll":      dash.TreeAll,
	"diff":         diffview.Run,
	"branches":     ops.Branches,
	"stack":        func([]string) int { return ops.Stack() },
	"deleteBranch": ops.DeleteBranch,
	"deleteBranchForce": func(args []string) int {
		return ops.DeleteBranch(append([]string{"-D"}, args...))
	},
	"push":         ops.Push,
	"update":       ops.Update,
	"sync":         ops.Sync,
	"uncommit":     ops.Uncommit,
	"move":         ops.Move,
	"abandon":      ops.Abandon,
	"revert":       ops.Revert,
	"edit":         ops.Edit,
	"squash":       ops.Squash,
	"continue":     ops.Continue,
	"abort":        ops.Abort,
	"hooks":        ops.InstallHooks,
	"wt":           ops.Worktree,
	"prNew":        ops.PRNew,
	"prReview":     ops.PRReview,
	"prReviewEdit": ops.PRReviewEdit,
	"land":         ops.Land,
	"new":          ops.New,
	"commit":       ops.Commit,
	"commitExclude": func(args []string) int {
		return ops.Commit(append([]string{"-e"}, args...))
	},
	"init":           ops.Init,
	"clone":          ops.Clone,
	"discard":        ops.Discard,
	"clear":          ops.Clear,
	"copy":           ops.Copy,
	"rebase":         ops.Rebase,
	"tag":            ops.Tag,
	"remote":         ops.Remote,
	"mergedBranches": ops.MergedBranches,
	"undo":           undo,
	"redo":           redo,
	"oplog":          oplogList,
}

func undo(args []string) int {
	n := 1
	force := false
	for _, arg := range args {
		switch arg {
		case "--force", "-f":
			force = true
		default:
			v, err := strconv.Atoi(arg)
			if err != nil || v < 1 {
				ui.Errf("gg undo: the count must be a positive number")
				return 2
			}
			n = v
		}
	}
	return oplog.Undo(n, force)
}

func redo(args []string) int { return oplog.Redo() }

func oplogList(args []string) int {
	if len(args) > 0 && args[0] == "clear" {
		return oplog.Clear()
	}
	return oplog.List()
}

// mutating verbs get an oplog entry and a footer.
var mutating = map[string]bool{
	"commit": true, "c": true, "ce": true, "amend": true, "uncommit": true,
	"split": true, "squash": true, "absorb": true, "empty": true,
	"move": true, "publish": true, "sync": true, "update": true,
	"abandon": true, "copy": true, "cp": true, "push": true, "p": true, "up": true,
	"edit": true, "continue": true, "hooks": true, "abort": true,
	"new": true, "discard": true, "clear": true, "cl": true,
	"revert": true, "rv": true,
	"reword": true, "rename": true, "describe": true, "desc": true,
	"cbr": true, "dbr": true,
	"mg": true, "wt": true,
	"pr": true, "rpr": true, "epr": true,
}

var table, rows = parse(registry)

func parse(src string) (map[string]Entry, []Entry) {
	out := make(map[string]Entry)
	var list []Entry
	category := "other"
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ---") && strings.HasSuffix(line, "---") {
			category = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "# ---"), "---"))
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, rest, _ := strings.Cut(line, "\t")
		snippet, tail, _ := strings.Cut(rest, "\t")
		description, fields, _ := strings.Cut(tail, "\t")
		name = strings.TrimSpace(name)
		snippet = strings.TrimSpace(snippet)
		if name == "" {
			continue
		}
		canonical, abbrevs, aliases := splitLabel(name)
		entry := Entry{
			Name:        canonical,
			Label:       name,
			Abbrevs:     abbrevs,
			Aliases:     aliases,
			Snippet:     snippet,
			Description: strings.TrimSpace(description),
			Category:    category,
		}
		for _, field := range strings.Split(fields, "\t") {
			field = strings.TrimSpace(field)
			if h, ok := strings.CutPrefix(field, "hint:"); ok {
				entry.Hint = strings.TrimSpace(h)
				continue
			}
			if d, ok := strings.CutPrefix(field, "display:"); ok {
				entry.Shown = strings.TrimSpace(d)
				continue
			}
			if field == "hidden" {
				entry.Hidden = true
			}
			if field == "variant" {
				entry.Variant = true
			}
			if field == "nogit" {
				entry.NoGit = true
			}
			if field == "displayonly" {
				entry.DisplayOnly = true
			}
		}
		if fn, ok := strings.CutPrefix(snippet, "fn:"); ok {
			entry.Fn = functions[fn]
			entry.Snippet = ""
		}
		if spec, ok := strings.CutPrefix(snippet, "tool:"); ok {
			fields := strings.Fields(spec)
			if len(fields) > 0 {
				entry.Tool = fields[0]
			}
			entry.Snippet = strings.Join(fields[1:], " ")
		}
		list = append(list, entry)
		for _, alias := range strings.Split(name, "|") {
			alias = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(alias), "="))
			if alias != "" {
				out[alias] = entry
			}
		}
	}
	return out, list
}

// Lookup resolves a verb name.
func Lookup(name string) (Entry, bool) {
	e, ok := table[name]
	if ok && e.DisplayOnly {
		return e, false
	}
	if ok && !e.IsFunc() && !e.IsTool() && e.Snippet == "" {
		return e, false
	}
	if ok && e.IsTool() && e.Tool == "" {
		return e, false
	}
	// a fn: entry with no Go implementation is a registry bug
	if ok && e.IsFunc() && e.Fn == nil {
		return e, false
	}
	return e, ok
}

// splitLabel cuts "describe|desc|=reword|rename" into the canonical name, its
// abbreviations and its aliases. A "=" starts the alias section.
func splitLabel(label string) (name string, abbrevs, aliases []string) {
	inAliases := false
	for i, token := range strings.Split(label, "|") {
		token = strings.TrimSpace(token)
		if rest, ok := strings.CutPrefix(token, "="); ok {
			inAliases = true
			token = strings.TrimSpace(rest)
		}
		if token == "" {
			continue
		}
		switch {
		case i == 0 && !inAliases:
			name = token
		case inAliases:
			aliases = append(aliases, token)
		default:
			abbrevs = append(abbrevs, token)
		}
	}
	return name, abbrevs, aliases
}

// Names returns verb names in registry order.
func Names() []string {
	var names []string
	for _, e := range rows {
		if e.DisplayOnly {
			continue
		}
		for _, alias := range strings.Split(e.Label, "|") {
			alias = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(alias), "="))
			if alias != "" {
				names = append(names, alias)
			}
		}
	}
	return names
}

// IsMutating reports whether a verb changes history.
func IsMutating(name string) bool { return mutating[name] }

// All returns one entry per registry row, aliases grouped.
func All() []Entry { return append([]Entry(nil), rows...) }

// Describe is the one-line help text for the verb.
// Display is the name column in help: the canonical name, then its
// abbreviations, or the explicit display: override when set. No parentheses:
// every character counts at 80 columns. Aliases stay on the aka line.
func (e Entry) Display() string {
	if e.Shown != "" {
		return e.Shown
	}
	if len(e.Abbrevs) == 0 {
		return e.Name
	}
	return e.Name + " " + strings.Join(e.Abbrevs, ", ")
}

// DisplayParts cuts Display() into the plain name and the rest of the column,
// the shorthands and the flags, which help paints grey.
func (e Entry) DisplayParts() (name, note string) {
	display := e.Display()
	if i := strings.Index(display, " "); i >= 0 {
		return display[:i], display[i:]
	}
	return display, ""
}

// Parent resolves the verb a variant row flags: the first word of Display.
func (e Entry) Parent() (Entry, bool) {
	word, _, _ := strings.Cut(e.Display(), " ")
	p, ok := table[word]
	return p, ok
}

// VariantFlag is what a variant row adds to its parent's cell, the display
// text minus the parent verb: "commit -e ce" gives "-e ce".
func (e Entry) VariantFlag() string {
	_, flag, found := strings.Cut(e.Display(), " ")
	if !found {
		return ""
	}
	return strings.TrimSpace(flag)
}

// Command is the git command behind the verb, empty when there is none to show.
func (e Entry) Command() string {
	if e.NoGit {
		return ""
	}
	if e.IsTool() {
		return e.Tool
	}
	if e.IsFunc() {
		return ""
	}
	return "git " + e.Snippet
}

func (e Entry) Describe() string {
	if e.IsFunc() {
		return "gg " + e.Name + " (built-in)"
	}
	if e.IsTool() {
		return strings.TrimSpace(e.Tool + " " + e.Snippet + " (external)")
	}
	return "git " + e.Snippet
}
