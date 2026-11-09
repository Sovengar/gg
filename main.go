// Command gg is a small git front-end with jj/but flavoured verbs.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"

	"gg/internal/gitx"
	"gg/internal/idmap"
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

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	// the help words parse their own flags: -v there means verbose, not dry
	if len(argv) > 0 {
		switch argv[0] {
		case "help", "h", "hh":
			return help(argv[0], argv[1:])
		}
	}

	m, args := parseFlags(argv)
	if len(args) == 0 {
		args = []string{"dash"}
	}

	switch args[0] {
	case "help", "h", "hh":
		return help(args[0], args[1:])
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

// resolveArgs replaces changeId-shaped arguments with the thing they name. Real
// refs and existing paths win over changeIDs. Every kind shares one namespace,
// so a token names one thing and the verb never has to guess.
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
		token := strings.TrimPrefix(a, "@")
		if branchVerb(name) && !isRef(a) {
			if branch, ok := idmap.Shared().BranchBy(token); ok {
				out[i] = branch
				continue
			}
		}
		if anyVerb(name) && !isRef(a) && !isPath(a) {
			typed, status := resolveAny(a)
			if status != 0 {
				return out, status
			}
			out[i] = typed
			continue
		}
		if !ids.Shaped(token) || isRef(a) || isPath(a) {
			out[i] = a
			continue
		}
		if pathVerb(name) {
			if path, ok := idmap.Shared().FileBy(token); ok {
				out[i] = path
				continue
			}
		}
		if sha, ok := idmap.Shared().CommitBy(token); ok {
			out[i] = sha
			continue
		}
		out[i] = a
	}
	return out, 0
}

// anyVerbs take a changeId of any kind: a dirty path, a commit or a branch.
var anyVerbs = map[string]bool{"discard": true}

func anyVerb(name string) bool { return anyVerbs[name] }

// resolveAny types the argument so the verb knows what it names. The kind
// prefixes path:, commit: and branch: force one kind.
func resolveAny(arg string) (string, int) {
	a := strings.TrimPrefix(arg, "@")
	for _, kind := range []string{"path", "commit", "branch"} {
		rest, ok := strings.CutPrefix(a, kind+":")
		if !ok {
			continue
		}
		return kind + ":" + resolveKind(kind, rest), 0
	}
	if a == "zz" {
		return a, 0
	}
	if fileID, hunkID, ok := strings.Cut(a, ":"); ok && hunkID != "" {
		if path, found := idmap.Shared().FileBy(fileID); found {
			return "hunk:" + path + ":" + hunkID, 0
		}
	}
	if path, ok := idmap.Shared().FileBy(a); ok {
		return "path:" + path, 0
	}
	if branch, ok := idmap.Shared().BranchBy(a); ok {
		return "branch:" + branch, 0
	}
	if sha, ok := idmap.Shared().CommitBy(a); ok {
		return "commit:" + sha, 0
	}
	if sha, ok := kindValue("commit", a); ok {
		return "commit:" + sha, 0
	}
	return a, 0
}

// resolveKind turns the tail of a typed argument into the value the kind needs:
// a path, a branch name or a sha.
func resolveKind(kind, name string) string {
	if v, ok := kindValue(kind, name); ok {
		return v
	}
	return name
}

// kindValue is the one lookup each kind knows how to do.
func kindValue(kind, name string) (string, bool) {
	switch kind {
	case "path":
		return idmap.Shared().FileBy(name)
	case "branch":
		if b, ok := idmap.Shared().BranchBy(name); ok {
			return b, true
		}
		if _, err := gitx.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+name); err == nil {
			return name, true
		}
	case "commit":
		if sha, ok := idmap.Shared().CommitBy(name); ok {
			return sha, true
		}
		if sha, err := gitx.Output("rev-parse", "--verify", "--quiet", name+"^{commit}"); err == nil && sha != "" {
			return sha, true
		}
	}
	return "", false
}

// pathVerbs are the only verbs where a file changeId may replace an argument.
var pathVerbs = map[string]bool{
	"diff": true, "rs": true, "restore": true,
	"commit": true, "c": true, "ce": true, "discard": true,
}

func pathVerb(name string) bool { return pathVerbs[name] }

// branchVerbs take a branch name or a branch changeId.
var branchVerbs = map[string]bool{
	"branch": true, "br": true, "branches": true,
	"cbr": true,
	"mg":  true, "merge": true,
	"rebase": true, "rb": true,
	"land": true, "ship": true,
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
	"sw":      "Use gg go <name> to reach a branch, or gg cbr <name> to create and switch.",
	"switch":  "Use gg go <name> to reach a branch, or gg cbr <name> to create and switch.",
	"dbr":     "Use gg discard <branch>, which drops its worktree too.",
	"Dbr":     "Use gg discard <branch> --force, which drops unmerged commits and the worktree.",
	"reap":    "Use gg discard <branch>, or gg discard zz for every uncommitted change.",
	"abandon": "Use gg discard <commit>.",
	"clear":   "Use gg discard zz, which drops every edit and deletes the new files.",
	"cl":      "Use gg discard zz, which drops every edit and deletes the new files.",
	"cl-wd":   "Use gg discard to drop edits, and gg discard zz to delete new files.",
	"cl-sa":   "It took the index. Use gg discard or gg discard zz.",
	"us":      "It took the index. Use gg discard or gg discard zz.",
	"raw":     "An unknown verb still reaches git: gg <verb>. The call is no longer recorded for undo.",
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
	"uncommit": true, "rebase": true, "rb": true,
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
	case "tag", "t":
		return ops.IsTagList(rest)
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
	fmt.Println(ui.Dim(fmt.Sprintf("  %s · ahead %s · behind %s", shortCommit(sha), ahead, behind)))
}

// help renders the table the word and the flags ask for. Each -v opens the
// rows up one step: h, hh, then the parameter hints.
func help(word string, argv []string) int {
	verbose := false
	for _, a := range argv {
		switch a {
		case "-v", "--verbose":
			verbose = true
		}
	}
	switch {
	case word == "hh" && verbose:
		return helpParams()
	case word == "hh":
		return helpTable(true)
	case verbose:
		return helpDescribed()
	}
	return helpTable(false)
}

// helpDescribed prints one row per verb: the name column and its description,
// with no parameters. gg h -v is this view.
func helpDescribed() int {
	entries := verbs.All()
	nameWidth := 0
	for _, e := range entries {
		if e.Hidden || e.Variant {
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
		if e.Hidden || e.Variant {
			continue
		}
		if e.Category != category {
			if category != "" {
				fmt.Println()
			}
			category = e.Category
			fmt.Println(ui.Dim(category))
		}
		fmt.Printf("  %s  %s\n", displayPadded(e, nameWidth), e.Description)
	}

	fmt.Println()
	fmt.Println(ui.Dim("  gg hh -v  adds the parameters and their hints"))
	return 0
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
	fmt.Println(ui.Dim("  gg h -v  adds a description per verb"))
	return 0
}

// helpTable prints one line of names per category, then a footer that names
// the aliases verb first, "describe -> reword, rename", and the hint to the
// longer views. Lines stay wrapped at COLUMNS so the whole table fits on
// screen. With merged set, each verb carries the flag variants that belong to
// it: "status st, -a sta, -f stf". gg h is this view, gg hh is the merged one.
func helpTable(merged bool) int {
	var order []string
	names := make(map[string][]cell)
	var aliases []cell
	extras := map[string][]string{}
	if merged {
		extras = variantExtras()
	}
	for _, e := range verbs.All() {
		if e.Hidden || e.Variant {
			continue // variants ride on their parent's cell
		}
		if _, seen := names[e.Category]; !seen {
			order = append(order, e.Category)
		}
		names[e.Category] = append(names[e.Category], nameCell(e, extras[e.Name]))
		if len(e.Aliases) > 0 {
			aliases = append(aliases, aliasCell(e))
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
	}

	fmt.Println()
	if len(aliases) > 0 {
		for _, line := range wrapCells(aliases, termWidth()-gutter) {
			fmt.Printf("  %s\n", joinCells(line))
		}
	}
	if merged {
		fmt.Println(ui.Dim("  gg hh -v  adds the parameters and their hints"))
	} else {
		fmt.Println(ui.Dim("  gg h -v  adds a description per verb, gg hh the parameters"))
	}
	return 0
}

// variantExtras maps a verb to the flags its variant rows carry, in registry
// order: status -> "-a sta", "-f stf".
// variantExtras maps a verb to the flags its variant rows carry, in registry
// order: status -> "-a sta", "-f stf". A row that is only a positional
// argument, like branch <name>, has no flag to merge and stays in hh -v.
func variantExtras() map[string][]string {
	extras := make(map[string][]string)
	for _, e := range verbs.All() {
		if e.Hidden || !e.Variant {
			continue
		}
		parent, ok := e.Parent()
		if !ok {
			continue
		}
		flag := e.VariantFlag()
		if flag == "" || strings.HasPrefix(flag, "<") {
			continue
		}
		extras[parent.Name] = append(extras[parent.Name], flag)
	}
	return extras
}

// cell is one help item: its plain text and the same text painted.
type cell struct{ plain, styled string }

// nameCell paints a verb name cyan and its note dim, the flag variants
// included when there are any.
func nameCell(e verbs.Entry, extras []string) cell {
	name, note := e.DisplayParts()
	plain := e.Display()
	if len(extras) > 0 {
		note += ", " + strings.Join(extras, ", ")
		plain += ", " + strings.Join(extras, ", ")
	}
	styled := ui.Cyan(name)
	if note != "" {
		styled += ui.Dim(note)
	}
	return cell{plain: plain, styled: styled}
}

// displayCell is nameCell without the flag variants.
func displayCell(e verbs.Entry) cell { return nameCell(e, nil) }

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

// showIDs prints the commit behind one or more changeIDs.
func showIDs(args []string) int {
	if len(args) == 0 {
		if sha, ok := gitx.Head(); ok {
			fmt.Printf("%s\n", shortCommit(sha))
			return 0
		}
		ui.Errf("gg: no commits yet")
		return 1
	}
	rc := 0
	for _, a := range args {
		sha, ok := idmap.Shared().CommitBy(strings.TrimPrefix(a, "@"))
		if !ok {
			ui.Errf("gg: %s matches no single commit", a)
			rc = 1
			continue
		}
		line := fmt.Sprintf("%s %s", idmap.Shared().CommitPlain(sha), sha)
		if branch := branchAt(sha); branch != "" {
			line += "  (" + branch + ")"
		}
		fmt.Println(line)
	}
	return rc
}

// shortCommit is a changeId and the short sha, the shape every footer uses.
func shortCommit(sha string) string {
	id := idmap.Shared().CommitPlain(sha)
	if len(sha) > 7 {
		sha = sha[:7]
	}
	return id + " " + sha
}

// branchAt names a local branch that points at the sha, for the id verb.
func branchAt(sha string) string {
	out, err := gitx.Output("for-each-ref", "--points-at", sha, "refs/heads", "--format=%(refname:short)")
	if err != nil {
		return ""
	}
	name, _, _ := strings.Cut(out, "\n")
	return name
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
