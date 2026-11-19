// Verb help pages: one markdown file per registry row under docs/verbs.
package main

import (
	"embed"
	"fmt"
	"strings"

	"gg/internal/ui"
	"gg/internal/verbs"
)

//go:embed docs/verbs/*.md
var verbDocs embed.FS

// verbDocPath is where the page for one registry row lives: the file is named
// after the canonical verb, so every abbreviation and alias reaches it.
func verbDocPath(e verbs.Entry) string { return "docs/verbs/" + e.Name + ".md" }

// verbDoc returns the page for a row, and whether the file exists.
func verbDoc(e verbs.Entry) (string, bool) {
	raw, err := verbDocs.ReadFile(verbDocPath(e))
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// helpTarget reports the verb a `-h`/`--help` argument asks about. The flag
// wins wherever it sits, so `gg commit foo -h` still shows the commit page.
func helpTarget(args []string) (string, bool) {
	for i, a := range args {
		if a != "-h" && a != "--help" {
			continue
		}
		if i == 0 {
			return "", false // a bare -h asks for the table, not a page
		}
		return args[0], true
	}
	return "", false
}

// verbHelp prints the page for one verb, or a fallback built from the registry
// when no page exists yet.
func verbHelp(name string) int {
	entry, ok := verbs.Lookup(name)
	if !ok {
		if hint, gone := removed[name]; gone {
			ui.Errf("gg: %s is gone. %s", name, hint)
			return 2
		}
		ui.Errf("gg: no page for %q.", name)
		ui.Errf("An unknown verb reaches git unchanged: gg %s <args>", name)
		ui.Errf("See every verb: gg help")
		return 2
	}
	page, ok := verbDoc(entry)
	if !ok {
		renderFallback(entry)
		return 0
	}
	renderVerbDoc(page)
	return 0
}

// renderVerbDoc paints the markdown-lite page: a level-one title, level-two
// sections and indented blocks. Body lines keep their bytes, so examples stay
// copy-pasteable; only the headings carry colour.
func renderVerbDoc(page string) {
	lines := strings.Split(strings.TrimRight(page, "\n"), "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			if i > 0 {
				fmt.Println()
			}
			fmt.Println(ui.Cyan(strings.TrimSpace(line[2:])))
		case strings.HasPrefix(line, "## "):
			fmt.Println()
			fmt.Println(ui.Cyan("  " + strings.TrimSpace(line[3:])))
		case line == "":
			fmt.Println()
		default:
			fmt.Println("  " + line)
		}
	}
	blank := ""
	if len(lines) > 0 {
		blank = lines[len(lines)-1]
	}
	if blank != "" {
		fmt.Println()
	}
	fmt.Println(ui.Dim("  gg help  every verb · gg help <verb>  one page"))
}

// renderFallback is the page a verb without a file gets: the registry facts,
// so no verb is ever silent.
func renderFallback(e verbs.Entry) {
	fmt.Println()
	fmt.Println(ui.Cyan("gg " + e.Display()))
	fmt.Println()
	fmt.Printf("  %s\n", e.Description)
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.Dim("category:"), e.Category)
	if len(e.Abbrevs) > 0 {
		fmt.Printf("  %s %s\n", ui.Dim("shorthand:"), strings.Join(e.Abbrevs, ", "))
	}
	if len(e.Aliases) > 0 {
		fmt.Printf("  %s %s\n", ui.Dim("alias:"), strings.Join(e.Aliases, ", "))
	}
	if cmd := e.Command(); cmd != "" {
		fmt.Printf("  %s %s\n", ui.Dim("runs:"), cmd)
	}
	if e.Hint != "" {
		fmt.Printf("  %s %s\n", ui.Dim("parameters:"), e.Hint)
	}
	if verbs.IsMutating(e.Name) {
		fmt.Printf("  %s %s\n", ui.Dim("mutating:"), "recorded in the oplog, undoable with gg undo")
	}
	fmt.Println()
	fmt.Println(ui.Dim("  no long page yet: " + verbDocPath(e)))
	fmt.Println(ui.Dim("  gg help  every verb · gg help <verb>  one page"))
}
