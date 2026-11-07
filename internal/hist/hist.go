// Package hist renders the commit graph inside gg, with no external script.
package hist

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/ui"
)

// DefaultLimit is how many commits log shows when no count is given.
const DefaultLimit = 6

// record is the git format: one field separator per column.
const record = "%H%x1f%h%x1f%ct%x1f%d%x1f%s%x1f%an%x1f%G?"

const pushedLimit = "5000"

// renderer holds everything a line needs to be drawn.
type renderer struct {
	idx    *ids.Index
	long   bool
	pushed map[string]bool
}

// Run prints the graph: a leading number sets the limit, -l/--long shows the
// full sha.
func Run(args []string) int {
	limits, extras, rc := parseArgs(args)
	if rc != 0 {
		return rc
	}

	r := newRenderer(limits.long)
	gitArgs := limits.logArgs(extras)

	if !ui.IsTTY(os.Stdout) {
		return stream(append(gitArgs, "--color=never"), r)
	}
	if _, err := exec.LookPath("less"); err != nil {
		return stream(append(gitArgs, "--color=always"), r)
	}
	return page(append(gitArgs, "--color=always"), r)
}

// Block returns the same lines Run prints, for callers that compose them.
func Block(n int, long bool, extras ...string) []string {
	r := newRenderer(long)
	gitArgs := []string{"log", "--graph", "--decorate=short", "--color=always",
		"--format=" + record, "-n", strconv.Itoa(n)}
	gitArgs = append(gitArgs, extras...)

	cmd := exec.Command("git", gitArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil
	}
	return r.render(&out)
}

type limits struct {
	n    int
	long bool
}

func parseArgs(args []string) (limits, []string, int) {
	l := limits{n: DefaultLimit}
	kept := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "-l", "--long":
			l.long = true
		default:
			kept = append(kept, a)
		}
	}
	if len(kept) > 0 {
		if n, err := strconv.Atoi(kept[0]); err == nil && n > 0 {
			l.n = n
			kept = kept[1:]
		}
	}
	return l, kept, 0
}

func (l limits) logArgs(extras []string) []string {
	args := []string{"log", "--graph", "--decorate=short", "--format=" + record, "-n", strconv.Itoa(l.n)}
	return append(args, extras...)
}

func newRenderer(long bool) *renderer {
	idx, err := ids.Load()
	if err != nil {
		idx = &ids.Index{}
	}
	return &renderer{idx: idx, long: long, pushed: PushedSet()}
}

// PushedSet collects every commit reachable from a remote-tracking branch.
func PushedSet() map[string]bool {
	out := make(map[string]bool)
	lines, err := gitx.OutputAll("rev-list", "--remotes", "--max-count="+pushedLimit)
	if err != nil {
		return out
	}
	for _, sha := range lines {
		out[sha] = true
	}
	return out
}

func stream(gitArgs []string, r *renderer) int {
	cmd := exec.Command("git", gitArgs...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return 1
	}
	r.draw(pipe, os.Stdout)
	return wait(cmd)
}

func page(gitArgs []string, r *renderer) int {
	cmd := exec.Command("git", gitArgs...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return 1
	}

	pager := exec.Command("less", "-FRSX")
	stdin, err := pager.StdinPipe()
	if err != nil {
		return 1
	}
	pager.Stdout = os.Stdout
	pager.Stderr = os.Stderr
	if err := pager.Start(); err != nil {
		r.draw(pipe, os.Stdout)
		return wait(cmd)
	}

	r.draw(pipe, stdin)
	_ = stdin.Close()
	_ = pager.Wait()
	return wait(cmd)
}

func (r *renderer) draw(src io.Reader, dst io.Writer) {
	w := bufio.NewWriter(dst)
	defer w.Flush()
	for _, line := range r.render(src) {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return
		}
	}
}

func (r *renderer) render(src io.Reader) []string {
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []string
	for scanner.Scan() {
		out = append(out, r.line(scanner.Text()))
	}
	return out
}

// ansi matches SGR escape sequences. git runs with --color=always on a TTY,
// so its graph arrives pre-coloured; gg strips that and paints every line
// itself, otherwise graph() would re-wrap git's ESC bytes and corrupt them.
var ansi = regexp.MustCompile("\\x1b\\[[0-9;]*m")

// line turns one git line into a gg line:
//
//	<graph> <handle> <sha> <age> <subject> <decorations> [author]
func (r *renderer) line(raw string) string {
	raw = strings.TrimRight(raw, "\r")
	raw = ansi.ReplaceAllString(raw, "")
	start := shaStart(raw)
	if start < 0 {
		return graph(raw, "")
	}

	fields := strings.Split(raw[start:], "\x1f")
	if len(fields) < 7 {
		return raw
	}
	sha, short, epoch, deco, subject, author, sig := fields[0], fields[1], fields[2], fields[3], fields[4], fields[5], fields[6]

	ball := ui.BrightMagenta("●")
	if r.pushed[sha] {
		ball = ui.BrightGreen("●")
	}

	shown := short
	if r.long {
		shown = sha
	}

	var b strings.Builder
	b.WriteString(strings.TrimRight(graph(raw[:start], ball), " "))
	b.WriteString(" ")
	b.WriteString(ui.Red(r.idx.Handle(sha)))
	b.WriteString(" ")
	b.WriteString(ui.Dim(shown))
	b.WriteString(" ")
	b.WriteString(ui.Dim(age(epoch)))
	b.WriteString(" ")
	b.WriteString(subject)
	if deco != "" {
		b.WriteString(" ")
		b.WriteString(ui.Yellow(deco))
	}
	b.WriteString(" ")
	b.WriteString(ui.Blue("[" + author + "]"))
	if sig = strings.TrimSpace(sig); sig != "" {
		b.WriteString(" ")
		b.WriteString(ui.Red("(" + sig + ")"))
	}
	return b.String()
}

// age renders a compact relative time: 5m, 2h, 10d, 3w.
func age(epoch string) string {
	secs, err := strconv.ParseInt(strings.TrimSpace(epoch), 10, 64)
	if err != nil {
		return ""
	}
	d := time.Since(time.Unix(secs, 0))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/24/7))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	}
}

// shaStart finds where a 40 character sha begins, ignoring hex inside words.
func shaStart(line string) int {
	for i := 0; i+40 <= len(line); i++ {
		if !isHex(line[i]) || (i > 0 && isHex(line[i-1])) {
			continue
		}
		if isHexRun(line[i:i+40]) && (i+40 == len(line) || !isHex(line[i+40])) {
			return i
		}
	}
	return -1
}

func isHexRun(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isHex(s[i]) {
			return false
		}
	}
	return true
}

func isHex(c byte) bool { return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' }

// graph redraws git's ASCII lanes with box characters. A non-empty ball
// replaces every commit dot, coloured by push state.
func graph(s, ball string) string {
	if s == "" {
		return s
	}
	plain := strings.NewReplacer("*", "●", "|", "│", "/", "╱", "\\", "╲", "_", " ").Replace(s)
	if !ui.Enabled() {
		return plain
	}
	palette := []func(string) string{ui.Cyan, ui.Magenta, ui.Blue, ui.Green, ui.Yellow}
	var b strings.Builder
	lane := 0
	for _, r := range plain {
		switch r {
		case ' ':
			b.WriteRune(' ')
		case '●':
			if ball != "" {
				b.WriteString(ball)
			} else {
				b.WriteString(palette[lane%len(palette)]("●"))
			}
		default:
			b.WriteString(palette[lane%len(palette)](string(r)))
		}
		lane++
	}
	return b.String()
}

func wait(cmd *exec.Cmd) int {
	if err := cmd.Wait(); err != nil {
		return 1
	}
	return 0
}
