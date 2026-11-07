// Package verbs resolves gg verbs to either a git argv or a Go function.
package verbs

import (
	_ "embed"
	"strings"

	"gg/internal/dash"
	"gg/internal/hist"
	"gg/internal/ops"
)

//go:embed verbs.tsv
var registry string

// Entry is one gg verb. Fn means a Go function, Tool means an external
// binary, anything else runs git with Snippet + args.
type Entry struct {
	Name        string // canonical name, the first alias
	Label       string // aliases as written in the registry, e.g. "commit|c"
	Snippet     string
	Description string
	Hint        string // grey parameter note shown after the description
	Category    string
	Tool        string
	Hidden      bool
	NoGit       bool // no git equivalent to show in help -all
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
	"log":      hist.Run,
	"reword":   ops.Reword,
	"split":    ops.Split,
	"amend":    ops.Amend,
	"dash":     dash.Run,
	"push":     ops.Push,
	"update":   ops.Update,
	"sync":     ops.Sync,
	"uncommit": ops.Uncommit,
	"move":     ops.Move,
	"abandon":  ops.Abandon,
	"edit":     ops.Edit,
	"squash":   ops.Squash,
	"continue": ops.Continue,
	"abort":    ops.Abort,
	"hooks":    ops.InstallHooks,
	"wt":       ops.Worktree,
	"new":      ops.New,
	"commit":   ops.Commit,
	"discard":  ops.Discard,
}

// mutating verbs get a snapshot and a footer.
var mutating = map[string]bool{
	"commit": true, "c": true, "ca": true, "amend": true, "uncommit": true,
	"split": true, "squash": true, "absorb": true, "empty": true,
	"move": true, "publish": true, "sync": true, "update": true, "undo": true,
	"abandon": true, "cp": true, "push": true, "pu": true, "up": true,
	"edit": true, "continue": true, "hooks": true, "abort": true, "wt": true,
	"new": true, "discard": true, "rs": true,
	"reword": true, "rename": true, "describe": true, "desc": true,
}

var table = parse(registry)

func parse(src string) map[string]Entry {
	out := make(map[string]Entry)
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
		aliases := strings.Split(name, "|")
		canonical := strings.TrimSpace(aliases[0])
		entry := Entry{
			Name:        canonical,
			Label:       name,
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
			if field == "nogit" {
				entry.NoGit = true
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
		for _, alias := range aliases {
			if alias = strings.TrimSpace(alias); alias != "" {
				out[alias] = entry
			}
		}
	}
	return out
}

// Lookup resolves a verb name.
func Lookup(name string) (Entry, bool) {
	e, ok := table[name]
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

// Names returns verb names in registry order.
func Names() []string {
	var names []string
	for _, line := range strings.Split(registry, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, _, _ := strings.Cut(line, "\t")
		for _, alias := range strings.Split(strings.TrimSpace(name), "|") {
			if alias = strings.TrimSpace(alias); alias != "" {
				names = append(names, alias)
			}
		}
	}
	return names
}

// IsMutating reports whether a verb changes history.
func IsMutating(name string) bool { return mutating[name] }

// All returns one entry per registry row, aliases grouped.
func All() []Entry {
	var out []Entry
	seen := make(map[string]bool)
	for _, alias := range Names() {
		e, ok := Lookup(alias)
		if !ok || seen[e.Label] {
			continue
		}
		seen[e.Label] = true
		out = append(out, e)
	}
	return out
}

// Describe is the one-line help text for the verb.
// Display is the name column in help: the canonical name first, the shorter
// variants in parentheses, or the explicit display: override when set.
func (e Entry) Display() string {
	if e.Shown != "" {
		return e.Shown
	}
	aliases := strings.Split(e.Label, "|")
	for i := range aliases {
		aliases[i] = strings.TrimSpace(aliases[i])
	}
	if len(aliases) < 2 {
		return aliases[0]
	}
	return aliases[0] + " (" + strings.Join(aliases[1:], ", ") + ")"
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
