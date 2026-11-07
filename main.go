// Command gg is a small git front-end with jj/but flavoured verbs.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gg/internal/files"
	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/ops"
	"gg/internal/ui"
	"gg/internal/verbs"
)

const version = "0.2.0"

type mode int

const (
	modeExec mode = iota
	modePrint
)

// index is loaded once per run and reused by handles and the footer.
var index = &ids.Index{}

func handleIndex() *ids.Index {
	if ix, err := ids.Load(); err == nil {
		index = ix
	}
	return index
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	m, args := parseFlags(argv)
	if len(args) == 0 {
		args = []string{"dash"}
	}

	switch args[0] {
	case "help", "h":
		return help(args[1:])
	case "version":
		fmt.Println("gg " + version)
		return 0
	case "verbs":
		for _, name := range verbs.Names() {
			fmt.Println(name)
		}
		return 0
	case "tools":
		return tools()
	case "id":
		return showIDs(args[1:])
	case "__seq-edit":
		return ops.SeqEdit(args[1:])
	case "raw":
		if len(args) < 2 {
			ui.Errf("gg raw: nothing to run")
			return 2
		}
		return execArgv(args[1:], "raw", m)
	case "undo":
		return undo(m)
	}

	name := args[0]
	rest := args[1:]
	if name != "raw" {
		resolved, rc := resolveArgs(name, rest)
		if rc != 0 {
			return rc
		}
		rest = resolved
	}

	entry, ok := verbs.Lookup(name)
	if !ok {
		// unknown verb: straight to git, still snapshotted by the mutating set
		return execArgv(append([]string{name}, rest...), name, m)
	}
	if entry.IsFunc() {
		return execFunc(entry, name, rest, m)
	}
	if entry.IsTool() {
		return execTool(entry, rest, m)
	}
	return execArgv(entry.Argv(rest), name, m)
}

func parseFlags(argv []string) (mode, []string) {
	m := modeExec
	args := make([]string, 0, len(argv))
	for _, a := range argv {
		switch a {
		case "-p", "--print", "-n", "--dry":
			m = modePrint
		default:
			args = append(args, a)
		}
	}
	return m, args
}

// dataFlags take an argument that must never be treated as a handle.
var dataFlags = map[string]bool{
	"-m": true, "--message": true, "-F": true, "--file": true,
	"-L": true, "-S": true, "-G": true, "--grep": true,
	"--author": true, "-C": true, "-c": true, "--date": true,
	"-t": true, "--template": true, "-x": true, "--exec": true,
}

// resolveArgs replaces handle-shaped arguments with the commit or branch they
// name. Real refs and existing paths win over handles.
func resolveArgs(name string, args []string) ([]string, int) {
	out := make([]string, len(args))
	skipNext := false
	stop := false

	for i, a := range args {
		if stop || skipNext {
			out[i] = a
			skipNext = false
			continue
		}
		if a == "--" {
			stop = true
			out[i] = a
			continue
		}
		if strings.HasPrefix(a, "-") {
			if dataFlags[a] {
				skipNext = true
			}
			out[i] = a
			continue
		}
		if !ids.Shaped(strings.TrimPrefix(a, "@")) || isRef(a) || isPath(a) {
			out[i] = a
			continue
		}
		ref, status := handleIndex().Resolve(a)
		switch status {
		case ids.Resolved:
			if ref.Branch != "" {
				out[i] = ref.Branch
			} else {
				out[i] = ref.SHA
			}
		case ids.Ambiguous:
			return out, 2
		default:
			if pathVerb(name) {
				if path, ok := fileIndex().Resolve(a); ok {
					out[i] = path
					continue
				}
			}
			out[i] = a
		}
	}
	return out, 0
}

var fileIdx *files.Index

func fileIndex() *files.Index {
	if fileIdx == nil {
		fileIdx = files.Load()
	}
	return fileIdx
}

// pathVerbs are the only verbs where a file handle may replace an argument.
var pathVerbs = map[string]bool{
	"add": true, "a": true, "aa": true, "ia": true,
	"diff": true, "rs": true, "restore": true, "us": true,
	"cl": true, "cl-wd": true, "cl-sa": true, "cp": true,
	"commit": true, "c": true, "ca": true, "discard": true,
}

func pathVerb(name string) bool { return pathVerbs[name] }

// wipFold lists the verbs that fold uncommitted edits into an active wip
// commit before they run. Path-scoped verbs are never listed: they address the
// tree themselves and folding first would hand them the wrong base.
var wipFold = map[string]bool{
	"update": true, "up": true, "sync": true, "push": true, "pu": true,
	"sw": true, "switch": true, "cbr": true, "mg": true, "merge": true,
	"amend": true, "absorb": true, "edit": true, "continue": true,
	"reword": true, "rename": true, "describe": true, "desc": true,
	"squash": true, "split": true, "move": true, "abandon": true, "cp": true,
}

func isRef(token string) bool {
	_, err := gitx.Output("rev-parse", "--verify", "--quiet", token)
	return err == nil
}

func isPath(token string) bool {
	_, err := os.Stat(token)
	return err == nil
}

func undo(m mode) int {
	prev, ok := gitx.Prev()
	if !ok {
		ui.Errf("gg: nothing to undo")
		return 1
	}
	return execArgv([]string{"reset", "--soft", prev}, "undo", m)
}

func execArgv(argv []string, verb string, m mode) int {
	if m == modePrint {
		fmt.Println("git " + quote(argv))
		return 0
	}
	if verbs.IsMutating(verb) {
		gitx.Snapshot()
	}
	if wipFold[verb] {
		ops.WipAmend()
	}
	if os.Getenv("GG_ECHO") != "" {
		ui.Errf("+ git %s", quote(argv))
	}
	rc := gitx.Run(argv...)
	footer(verb)
	return rc
}

func execFunc(entry verbs.Entry, typed string, rest []string, m mode) int {
	if m == modePrint {
		fmt.Println(strings.TrimSpace("gg " + typed + " " + quote(rest)))
		return 0
	}
	if verbs.IsMutating(entry.Name) {
		gitx.Snapshot()
	}
	if wipFold[entry.Name] {
		ops.WipAmend()
	}
	if os.Getenv("GG_ECHO") != "" {
		ui.Errf("+ gg %s", strings.TrimSpace(typed+" "+quote(rest)))
	}
	rc := entry.Fn(rest)
	footer(entry.Name)
	return rc
}

func execTool(entry verbs.Entry, rest []string, m mode) int {
	argv := append(strings.Fields(entry.Snippet), rest...)
	if m == modePrint {
		fmt.Println(quote(append([]string{entry.Tool}, argv...)))
		return 0
	}
	if verbs.IsMutating(entry.Name) {
		gitx.Snapshot()
	}
	if wipFold[entry.Name] {
		ops.WipAmend()
	}
	if os.Getenv("GG_ECHO") != "" {
		ui.Errf("+ %s", quote(append([]string{entry.Tool}, argv...)))
	}
	rc := gitx.RunBin(entry.Tool, argv...)
	footer(entry.Name)
	return rc
}

// tools lists external tools with their install state.
func tools() int {
	width := 0
	for _, e := range verbs.All() {
		if e.IsTool() && len(e.Name) > width {
			width = len(e.Name)
		}
	}
	for _, e := range verbs.All() {
		if !e.IsTool() {
			continue
		}
		path, err := exec.LookPath(e.Tool)
		state := path
		if err != nil {
			state = ui.Dim("MISSING")
		}
		fmt.Printf("  %s  %s  %s\n", ui.Cyan(fmt.Sprintf("%-*s", width, e.Name)), e.Describe(), state)
	}
	return 0
}

func footer(verb string) {
	if !verbs.IsMutating(verb) {
		return
	}
	sha, ok := gitx.Head()
	if !ok {
		return
	}
	ahead, behind := "?", "?"
	if up, hasUp := gitx.Upstream(); hasUp {
		if n, ok := gitx.Count(up + "..HEAD"); ok {
			ahead = strconv.Itoa(n)
		}
		if n, ok := gitx.Count("HEAD.." + up); ok {
			behind = strconv.Itoa(n)
		}
	}
	fmt.Println(ui.Dim(fmt.Sprintf("  %s · ahead %s · behind %s", handleIndex().Short(sha), ahead, behind)))
}

func help(argv []string) int {
	all := false
	for _, a := range argv {
		if a == "-all" || a == "--all" {
			all = true
		}
	}

	entries := verbs.All()

	var order []string
	groups := make(map[string][]verbs.Entry)
	for _, e := range entries {
		if _, seen := groups[e.Category]; !seen {
			order = append(order, e.Category)
		}
		groups[e.Category] = append(groups[e.Category], e)
	}

	nameWidth, descWidth := 0, 0
	for _, e := range entries {
		if n := len(e.Display()); n > nameWidth {
			nameWidth = n
		}
		if n := len(e.Description); n > descWidth {
			descWidth = n
		}
	}
	if descWidth > 46 {
		descWidth = 46
	}

	for _, category := range order {
		fmt.Println(ui.Dim(category + ":"))
		for _, e := range groups[category] {
			if e.Hidden {
				continue
			}
			line := fmt.Sprintf("  %s  %-*s", ui.Cyan(fmt.Sprintf("%-*s", nameWidth, e.Display())), descWidth, e.Description)
			if e.Hint != "" {
				line += "  " + ui.Dim(e.Hint)
			}
			if all {
				if cmd := e.Command(); cmd != "" {
					line += "  " + ui.Dim(cmd)
				}
			}
			fmt.Println(strings.TrimRight(line, " "))
		}
		fmt.Println()
	}

	fmt.Println("  flags: -n/--dry  -p/--print   help -all adds the git command behind each verb")
	fmt.Println("  env: GG_ECHO=1 traces the git call")
	return 0
}

// showIDs prints the commit behind one or more handles.
func showIDs(args []string) int {
	if len(args) == 0 {
		if sha, ok := gitx.Head(); ok {
			fmt.Printf("%s\n", handleIndex().Short(sha))
			return 0
		}
		ui.Errf("gg: no commits yet")
		return 1
	}
	rc := 0
	for _, a := range args {
		ref, status := handleIndex().Resolve(a)
		if status != ids.Resolved {
			ui.Errf("gg: %s matches no single commit", a)
			rc = 1
			continue
		}
		line := fmt.Sprintf("%s %s", handleIndex().Handle(ref.SHA), ref.SHA)
		if ref.Branch != "" {
			line += "  (" + ref.Branch + ")"
		}
		fmt.Println(line)
	}
	return rc
}

func quote(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'\\$`*?()[]{}|&;<>~") {
			parts[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}
