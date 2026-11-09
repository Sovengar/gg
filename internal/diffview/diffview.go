// Package diffview draws a git diff as boxes: one box per change, with a
// gutter of old and new line numbers. It is gg's reading of but diff.
package diffview

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gg/internal/files"
	"gg/internal/gitx"
	"gg/internal/idmap"
	"gg/internal/ui"
)

// hunkLabel draws a hunk id: the unique prefix red, padded to two characters in
// grey so the handle keeps the file:hunk shape.
func hunkLabel(h files.Hunk) string {
	end := len(h.ID)
	if end < 2 {
		end = 2
	}
	if end > len(h.Full) {
		end = len(h.Full)
	}
	return ui.Red(h.ID) + ui.Dim(h.Full[len(h.ID):end])
}

// splitArgs separates the flags from the revisions, so a flag such as -w still
// describes the working tree while a revision names committed history.
func splitArgs(args []string) (flags, revisions []string) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			continue
		}
		revisions = append(revisions, a)
	}
	return flags, revisions
}

// contextLines is how many unchanged lines each hunk shows, and it must match
// the value files.HunksOf reads so the printed hunk id names the same region.
const contextLines = 3

// Run draws the diff of the working tree. When stdout is not a terminal it
// falls back to plain git diff, so pipes and scripts still parse.
func Run(args []string) int {
	if !ui.IsTTY(os.Stdout) {
		return gitx.Run(append([]string{"diff"}, args...)...)
	}
	out, err := gitx.OutputAll(append([]string{"diff", "--no-color", "--no-ext-diff"}, args...)...)
	if err != nil || len(out) == 0 {
		return 0
	}
	// a revision argument reads committed history, so its regions are not
	// hunks any verb can act on; flags such as -w still describe the tree
	flags, revisions := splitArgs(args)
	render(parse(out), len(revisions) == 0, flags...)
	return 0
}

type hunk struct {
	oldStart, oldCount int
	newStart, newCount int
	lines              []dline
}

type dline struct {
	kind     byte
	old, new int
	text     string
}

type fileDiff struct {
	path   string
	hunks  []*hunk
	binary bool
}

var hunkRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parse reads unified diff lines into files and hunks.
func parse(lines []string) []*fileDiff {
	var files []*fileDiff
	var cur *fileDiff
	var h *hunk
	oldNo, newNo := 0, 0

	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			cur = &fileDiff{path: diffPath(l)}
			files = append(files, cur)
			h = nil
		case cur == nil:
			continue
		case strings.HasPrefix(l, "Binary files "):
			cur.binary = true
			h = nil
		case strings.HasPrefix(l, "@@"):
			m := hunkRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			h = &hunk{
				oldStart: atoi(m[1]), oldCount: atoiOr(m[2], 1),
				newStart: atoi(m[3]), newCount: atoiOr(m[4], 1),
			}
			oldNo, newNo = h.oldStart, h.newStart
			cur.hunks = append(cur.hunks, h)
		case h != nil && l != "" && (l[0] == ' ' || l[0] == '-' || l[0] == '+'):
			d := dline{kind: l[0], text: l[1:]}
			switch l[0] {
			case ' ':
				d.old, d.new = oldNo, newNo
				oldNo++
				newNo++
			case '-':
				d.old = oldNo
				oldNo++
			case '+':
				d.new = newNo
				newNo++
			}
			h.lines = append(h.lines, d)
		}
	}
	return files
}

// diffPath takes the b/ path of a "diff --git" line.
func diffPath(line string) string {
	if i := strings.LastIndex(line, " b/"); i >= 0 {
		return strings.Trim(strings.TrimSpace(line[i+3:]), `"`)
	}
	return strings.TrimPrefix(line, "diff --git ")
}

func atoi(s string) int { return atoiOr(s, 0) }

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

// render draws every file as a stack of boxes.
func render(diffs []*fileDiff, handles bool, flags ...string) {
	fidx := idmap.Shared()
	for _, f := range diffs {
		handle := fidx.File(f.path)
		var hunks []files.Hunk
		if handles {
			hunks = files.HunksOf(f.path, contextLines, flags...)
		}

		if f.binary {
			box(strings.TrimSpace(handle+" "+f.path),
				strings.TrimSpace(ui.Red(handle)+" "+f.path),
				[]string{ui.Dim("   Binary file")})
			continue
		}
		for i, h := range f.hunks {
			if len(h.lines) == 0 {
				continue
			}
			// the handle names the region: file changeId, colon, hunk id
			plainID := handle
			paintedID := idmap.Shared().FileDisplay(f.path)
			if handles && i < len(hunks) {
				plainID = handle + ":" + hunks[i].ID
				paintedID = paintedID + ":" + hunkLabel(hunks[i])
			}
			plain := strings.TrimSpace(plainID + " " + f.path)
			painted := strings.TrimSpace(paintedID + " " + f.path)
			box(plain, painted, h.rows())
		}
	}
}

// box draws the border, the title and the rows of one change.
func box(plain, painted string, rows []string) {
	edge := ui.Dim(strings.Repeat("─", len(plain)))
	fmt.Println(edge + ui.Dim("╮"))
	fmt.Println(painted + ui.Dim("│"))
	fmt.Println(edge + ui.Dim("╯"))
	for _, r := range rows {
		fmt.Println(r)
	}
}

// rows renders one hunk with its old and new line numbers aligned.
func (h *hunk) rows() []string {
	width := digits(max(h.oldStart+h.oldCount-1, h.newStart+h.newCount-1))
	gutter := func(old, new string) string {
		return "   " + fmt.Sprintf("%*s %*s", width, old, width, new) + "│"
	}

	out := make([]string, 0, len(h.lines))
	for _, d := range h.lines {
		switch d.kind {
		case ' ':
			out = append(out, ui.Dim(gutter(strconv.Itoa(d.old), strconv.Itoa(d.new)))+" "+d.text)
		case '-':
			out = append(out, ui.Dim(gutter(strconv.Itoa(d.old), ""))+ui.Red("-"+d.text))
		case '+':
			out = append(out, ui.Dim(gutter("", strconv.Itoa(d.new)))+ui.Green("+"+d.text))
		}
	}
	return out
}

func digits(n int) int {
	if n < 10 {
		return 1
	}
	d := 0
	for n > 0 {
		n /= 10
		d++
	}
	return d
}
