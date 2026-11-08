// Command gg is a small git front-end with jj/but flavoured verbs.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"

	"gg/internal/files"
	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/oplog"
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

// index is loaded once per run and reused by changeIDs and the footer.
var index = &ids.Index{}

func changeIndex() *ids.Index {
	if ix, err := ids.Load(); err == nil {
		index = ix
	}
	return index
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	// help parses its own flags: -p there means params, not print mode
	if len(argv) > 0 && (argv[0] == "help" || argv[0] == "h") {
		return help(argv[1:])
	}

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
	case "changeId", "id":
		return showIDs(args[1:])
	case "__seq-edit":
		return ops.SeqEdit(args[1:])
	}

	name := args[0]
	rest := args[1:]
	if !noResolve[name] {
		resolved, rc := resolveArgs(name, rest)
		if rc != 0 {
			return rc
		}
		rest = resolved
	}

	entry, ok := verbs.Lookup(name)
	if !ok {
		// a verb gg dropped says where its work went, instead of failing in git
		if hint, gone := removed[name]; gone {
			ui.Errf("gg: %s is gone. %s", name, hint)
			return 2
		}
		// unknown verb: straight to git, still recorded by the mutating set
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

// dataFlags take an argument that must never be treated as a changeId.
var dataFlags = map[string]bool{
	"-m": true, "--message": true, "-F": true, "--file": true,
	"-L": true, "-S": true, "-G": true, "--grep": true,
	"--author": true, "-C": true, "-c": true, "--date": true,
	"-t": true, "--template": true, "-x": true, "--exec": true,
}

// resolveArgs replaces changeId-shaped arguments with the commit or branch they
// name. Real refs and existing paths win over changeIDs.
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
		// "branch new <name>" reads new as a subcommand, never as a changeId
		if i == 0 && a == "new" && newSubcommand(name) {
			out[i] = a
			continue
		}
		if branchVerb(name) && !isRef(a) {
			// a branch changeId names the branch itself, not the commit it points at
			switch bname, st := branchIndex().Resolve(strings.TrimPrefix(a, "@")); st {
			case ids.Resolved:
				out[i] = bname
				continue
			case ids.Ambiguous:
				return out, 2
			}
		}
		if !ids.Shaped(strings.TrimPrefix(a, "@")) || isRef(a) || isPath(a) {
			out[i] = a
			continue
		}
		ref, status := changeIndex().Resolve(a)
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

var brIdx *ids.BranchIndex

func branchIndex() *ids.BranchIndex {
	if brIdx != nil {
		return brIdx
	}
	ix, err := ids.LoadBranches()
	if err != nil {
		ix = &ids.BranchIndex{}
	}
	brIdx = ix
	return brIdx
}

func fileIndex() *files.Index {
	if fileIdx == nil {
		fileIdx = files.Load()
	}
	return fileIdx
}

// pathVerbs are the only verbs where a file changeId may replace an argument.
var pathVerbs = map[string]bool{
	"diff": true, "rs": true, "restore": true,
	"commit": true, "c": true, "ca": true, "discard": true,
}

func pathVerb(name string) bool { return pathVerbs[name] }

// branchVerbs take a branch name or a branch changeId.
var branchVerbs = map[string]bool{
	"branch": true, "br": true, "branches": true,
	"cbr": true,
	"dbr": true, "Dbr": true, "mg": true, "merge": true,
}

func branchVerb(name string) bool { return branchVerbs[name] }

// newSubcommand lists the branch verbs where a leading "new" is the create
// subcommand, never a branch that happens to be called new.
var newSubcommandVerbs = map[string]bool{
	"branch": true, "br": true, "branches": true,
}

func newSubcommand(name string) bool { return newSubcommandVerbs[name] }

// removed verbs name the verb that took their work, so an old habit fails with
// a direction instead of a git error.
var removed = map[string]string{
	"sw":     "Use gg br <name> to switch, or gg cbr <name> to create and switch.",
	"switch": "Use gg br <name> to switch, or gg cbr <name> to create and switch.",
	"cl-wd":  "Use gg discard to drop edits, and gg clear to delete new files.",
	"cl-sa":  "It took the index. Use gg discard or gg clear.",
	"us":     "It took the index. Use gg discard or gg clear.",
	"raw":    "An unknown verb still reaches git: gg <verb>. The call is no longer recorded for undo.",
}

// noResolve are verbs whose arguments are not changeIDs.
var noResolve = map[string]bool{
	"undo": true, "redo": true, "oplog": true,
	"pr": true, "rpr": true, "epr": true,
}

// wipFold lists the verbs that fold uncommitted edits into an active wip
// commit before they run. Path-scoped verbs are never listed: they address the
// tree themselves and folding first would hand them the wrong base. push is
// absent too: it refuses a wip commit outright, so folding would rewrite HEAD
// for a command that changes nothing.
var wipFold = map[string]bool{
	"update": true, "up": true, "sync": true,
	"cbr": true, "mg": true, "merge": true,
	"amend": true, "absorb": true, "edit": true, "continue": true,
	"reword": true, "rename": true, "describe": true, "desc": true,
	"squash": true, "split": true, "move": true, "abandon": true, "copy": true, "cp": true,
	"uncommit": true,
}

// noSnapshot lists the verbs that must see the tree exactly as the user left
// it. Snapshotting first would let gg abort or gg undo step over, or reset,
// commits that already absorbed the edits they are meant to drop.
var noSnapshot = map[string]bool{
	"abort": true, "undo": true, "redo": true,
}

func isRef(token string) bool {
	_, err := gitx.Output("rev-parse", "--verify", "--quiet", token)
	return err == nil
}

func isPath(token string) bool {
	_, err := os.Stat(token)
	return err == nil
}

func execArgv(argv []string, verb string, m mode) int {
	if m == modePrint {
		fmt.Println("git " + quote(argv))
		return 0
	}
	if verbs.IsMutating(verb) {
		oplog.Record(verb)
	}
	if wipFold[verb] {
		ops.WipAmend()
	}
	if !noSnapshot[verb] {
		ops.StopAmend()
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
	// wt ls only reads, so it records no entry and prints no footer.
	if entry.Name == "wt" && len(rest) > 0 && (rest[0] == "ls" || rest[0] == "list") {
		return entry.Fn(rest)
	}
	// a --check run only reports: no oplog entry, no footer
	if verbs.IsMutating(entry.Name) && !reportsOnly(entry.Name, rest) {
		oplog.Record(entry.Name)
	}
	if wipFold[entry.Name] {
		ops.WipAmend()
	}
	if !noSnapshot[entry.Name] {
		ops.StopAmend()
	}
	if os.Getenv("GG_ECHO") != "" {
		ui.Errf("+ gg %s", strings.TrimSpace(typed+" "+quote(rest)))
	}
	rc := entry.Fn(rest)
	if !reportsOnly(entry.Name, rest) {
		footer(entry.Name)
	}
	return rc
}

// reportsOnly is a verb called in check mode: it reports, it changes nothing.
// Only the verbs with a check flag read -c that way; elsewhere -c means create.
func reportsOnly(verb string, rest []string) bool {
	switch verb {
	case "update", "up", "push", "p":
	default:
		return false
	}
	for _, a := range rest {
		if a == "--check" || a == "-c" {
			return true
		}
	}
	return false
}

func execTool(entry verbs.Entry, rest []string, m mode) int {
	argv := append(strings.Fields(entry.Snippet), rest...)
	if m == modePrint {
		fmt.Println(quote(append([]string{entry.Tool}, argv...)))
		return 0
	}
	// a --check run only reports: no oplog entry, no footer
	if verbs.IsMutating(entry.Name) && !reportsOnly(entry.Name, rest) {
		oplog.Record(entry.Name)
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
	fmt.Println(ui.Dim(fmt.Sprintf("  %s · ahead %s · behind %s", changeIndex().Short(sha), ahead, behind)))
}

func help(argv []string) int {
	for _, a := range argv {
		switch a {
		case "-all", "--all", "-a":
			return helpAll()
		case "-p", "--params":
			return helpParams()
		}
	}
	return helpCompact()
}

// helpParams lists each verb with its parameters: the hint and the variants.
func helpParams() int {
	entries := verbs.All()
	nameWidth := 0
	for _, e := range entries {
		if e.Hidden || (e.Hint == "" && !e.Variant) {
			continue
		}
		if n := len(e.Display()); n > nameWidth {
			nameWidth = n
		}
	}
	if nameWidth > 34 {
		nameWidth = 34
	}

	category := ""
	for _, e := range entries {
		if e.Hidden || (e.Hint == "" && !e.Variant) {
			continue
		}
		if e.Category != category {
			if category != "" {
				fmt.Println()
			}
			category = e.Category
			fmt.Println(ui.Dim(category))
		}
		note := e.Hint
		if note == "" && e.Variant {
			note = "same verb, other flag"
		}
		fmt.Printf("  %s  %s\n", displayPadded(e, nameWidth), ui.Dim(note))
	}

	fmt.Println()
	fmt.Println(ui.Dim("  gg help --all  adds descriptions and the git command behind each verb"))
	return 0
}

// helpCompact prints one line of canonical names per category, then one line
// naming the aliases of that category, verb first: "reword -> rename, describe".
// Both lines stay wrapped at COLUMNS so the whole table fits on screen.
func helpCompact() int {
	var order []string
	names := make(map[string][]cell)
	aliases := make(map[string][]cell)
	for _, e := range verbs.All() {
		if e.Hidden || e.Variant {
			continue // variants carry a flag, they belong to gg help -p
		}
		if _, seen := names[e.Category]; !seen {
			order = append(order, e.Category)
		}
		names[e.Category] = append(names[e.Category], displayCell(e))
		if len(e.Aliases) > 0 {
			aliases[e.Category] = append(aliases[e.Category], aliasCell(e))
		}
	}

	labelWidth := 0
	for _, category := range order {
		if n := len(category); n > labelWidth {
			labelWidth = n
		}
	}
	const gutter = 4 // two spaces, two spaces after the label
	indent := strings.Repeat(" ", gutter+labelWidth)
	width := termWidth() - gutter - labelWidth

	for _, category := range order {
		lines := wrapCells(names[category], width)
		fmt.Printf("  %s  %s\n", ui.Cyan(fmt.Sprintf("%-*s", labelWidth, category)), lines1(lines))
		for _, line := range lines[1:] {
			fmt.Printf("%s%s\n", indent, joinCells(line))
		}
		if len(aliases[category]) == 0 {
			continue
		}
		for _, line := range wrapCells(aliases[category], width) {
			fmt.Printf("%s%s\n", indent, joinCells(line))
		}
	}

	fmt.Println()
	fmt.Println(ui.Dim("  gg help --all  adds descriptions, hints and the git command behind each verb"))
	return 0
}

// cell is one help item: its plain text and the same text painted.
type cell struct{ plain, styled string }

// displayCell paints a verb name cyan and its shorthand note dim.
func displayCell(e verbs.Entry) cell {
	name, note := e.DisplayParts()
	styled := ui.Cyan(name)
	if note != "" {
		styled += ui.Dim(note)
	}
	return cell{plain: e.Display(), styled: styled}
}

// aliasCell names one verb's aliases: the verb, an arrow and the aliases.
func aliasCell(e verbs.Entry) cell {
	also := strings.Join(e.Aliases, ", ")
	return cell{
		plain:  e.Name + " -> " + also,
		styled: ui.Cyan(e.Name) + ui.Dim(" -> "+also),
	}
}

// displayPadded is displayCell padded to width, padding left plain.
func displayPadded(e verbs.Entry, width int) string {
	c := displayCell(e)
	if pad := width - len(c.plain); pad > 0 {
		c.styled += strings.Repeat(" ", pad)
	}
	return c.styled
}

// lines1 renders the first wrapped line of cells.
func lines1(lines [][]cell) string { return joinCells(lines[0]) }

// joinCells glues cells with the item separator, each one painted.
func joinCells(cells []cell) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = c.styled
	}
	return strings.Join(parts, " · ")
}

// wrapCells lays cells out left to right, breaking between cells at width.
// Widths are measured on the plain text, so the paint never shifts a column.
func wrapCells(cells []cell, width int) [][]cell {
	if width < 20 {
		width = 20
	}
	var lines [][]cell
	var line []cell
	size := 0
	for _, c := range cells {
		n := utf8.RuneCountInString(c.plain)
		switch {
		case len(line) == 0:
			line, size = []cell{c}, n
		case size+n+3 <= width:
			line = append(line, c)
			size += n + 3
		default:
			lines = append(lines, line)
			line, size = []cell{c}, n
		}
	}
	return append(lines, line)
}

// wrapItems lays items out as one string, breaking between items at width.
func wrapItems(items []string, width int) []string {
	if width < 20 {
		width = 20
	}
	var lines []string
	line := ""
	for _, item := range items {
		switch {
		case line == "":
			line = item
		case utf8.RuneCountInString(line)+utf8.RuneCountInString(item)+3 <= width:
			line += " · " + item
		default:
			lines = append(lines, line)
			line = item
		}
	}
	return append(lines, line)
}

// termWidth reads COLUMNS, falling back to the usual 80 columns.
func termWidth() int {
	if raw := os.Getenv("COLUMNS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 80
}

// helpAll prints the full table: description, hint and git command per verb.
func helpAll() int {
	entries := verbs.All()

	var order []string
	groups := make(map[string][]verbs.Entry)
	for _, e := range entries {
		if e.Hidden {
			continue // an all-hidden category prints no header
		}
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
			line := fmt.Sprintf("  %s  %-*s", displayPadded(e, nameWidth), descWidth, e.Description)
			if e.Hint != "" {
				line += "  " + ui.Dim(e.Hint)
			}
			if cmd := e.Command(); cmd != "" {
				line += "  " + ui.Dim(cmd)
			}
			fmt.Println(strings.TrimRight(line, " "))
		}
		fmt.Println()
	}

	fmt.Println("  flags: -n/--dry  -p/--print   gg help --all shows all of this")
	fmt.Println("  env: GG_ECHO=1 traces the git call")
	return 0
}

// showIDs prints the commit behind one or more changeIDs.
func showIDs(args []string) int {
	if len(args) == 0 {
		if sha, ok := gitx.Head(); ok {
			fmt.Printf("%s\n", changeIndex().Short(sha))
			return 0
		}
		ui.Errf("gg: no commits yet")
		return 1
	}
	rc := 0
	for _, a := range args {
		ref, status := changeIndex().Resolve(a)
		if status != ids.Resolved {
			ui.Errf("gg: %s matches no single commit", a)
			rc = 1
			continue
		}
		line := fmt.Sprintf("%s %s", changeIndex().ChangeID(ref.SHA), ref.SHA)
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
