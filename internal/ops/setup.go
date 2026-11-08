package ops

import (
	"fmt"
	"os/exec"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ui"
)

// Init runs git init, then colocates a jj repo on top of it. --bare or
// --no-colocate leave the repo git only.
func Init(args []string) int {
	if rc := gitx.Run(append([]string{"init"}, stripGG(args)...)...); rc != 0 {
		return rc
	}
	dir := "."
	if pos := positionals(args, initValueFlags); len(pos) > 0 {
		dir = pos[0]
	}
	return colocate(dir, args)
}

// Clone runs git clone, then colocates a jj repo on top of the checkout. The
// first arg is the url, the second the target directory. --bare, --mirror or
// --no-colocate leave the clone git only.
func Clone(args []string) int {
	pos := positionals(args, cloneValueFlags)
	if len(pos) < 1 {
		ui.Errf("gg clone <url> [dir]: name the repository to clone")
		return 2
	}
	if rc := gitx.Run(append([]string{"clone"}, stripGG(args)...)...); rc != 0 {
		return rc
	}
	dest := cloneDest(pos, bare(args))
	return colocate(dest, args)
}

// cloneDest is where git puts a clone: the named directory, else the url's last
// path element. A bare clone keeps the .git suffix, a plain one drops it.
func cloneDest(pos []string, bare bool) string {
	if len(pos) > 1 {
		return pos[1]
	}
	name := pos[0]
	name = strings.TrimSuffix(name, "/")
	if i := strings.LastIndexAny(name, "/:"); i >= 0 {
		name = name[i+1:]
	}
	if !bare {
		name = strings.TrimSuffix(name, ".git")
	}
	return name
}

// colocate adds the jj repo unless the call says not to, or git built a bare
// repo that has no working copy to colocate.
func colocate(dir string, args []string) int {
	if isBare(dir) {
		note("bare: %s (git only)", dir)
		return 0
	}
	if !wantsColocate(args) {
		note("git only: %s (no jj)", dir)
		return 0
	}
	if _, err := exec.LookPath("jj"); err != nil {
		ui.Errf("gg: git is done, but jj is not installed; %s stays git only", dir)
		return 0
	}
	if rc := gitx.RunBin("jj", "git", "init", "--colocate", dir); rc != 0 {
		ui.Errf("gg: git is done, but jj could not colocate %s", dir)
		return 0
	}
	note("colocated: %s (git + jj)", dir)
	return 0
}

// wantsColocate reports whether jj goes on top. --no-colocate asks for plain
// git, and --git-repo shares an existing store instead of colocating.
func wantsColocate(args []string) bool {
	for _, a := range args {
		if a == "--no-colocate" || strings.HasPrefix(a, "--git-repo") {
			return false
		}
	}
	return true
}

// stripGG drops the flags only gg reads, so git never sees them.
func stripGG(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a != "--no-colocate" {
			out = append(out, a)
		}
	}
	return out
}

// bare reports whether the call builds a repo with no working copy.
func bare(args []string) bool {
	for _, a := range args {
		if a == "--bare" || a == "--mirror" {
			return true
		}
	}
	return false
}

// isBare trusts git over the flags: a repo with no working copy is bare.
func isBare(dir string) bool {
	out, err := gitx.Output("-C", dir, "rev-parse", "--is-bare-repository")
	return err == nil && out == "true"
}

// positionals returns the args git reads as paths: no flags, and no value of a
// flag that takes one.
func positionals(args []string, valueFlags map[string]bool) []string {
	var out []string
	skip := false
	for _, a := range args {
		if skip {
			skip = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			skip = valueFlags[a]
			continue
		}
		out = append(out, a)
	}
	return out
}

// note prints one dim line: the hint that says what the repo now holds.
func note(format string, args ...any) {
	fmt.Println(ui.Dim("  " + fmt.Sprintf(format, args...)))
}

// initValueFlags take a value on git init.
var initValueFlags = map[string]bool{
	"-b": true, "--initial-branch": true, "--template": true,
	"--separate-git-dir": true, "--object-format": true,
	"--ref-format": true, "--shared": true,
}

// cloneValueFlags take a value on git clone.
var cloneValueFlags = map[string]bool{
	"-b": true, "--branch": true, "-o": true, "--origin": true,
	"-u": true, "--upload-pack": true, "--depth": true,
	"--shallow-since": true, "--shallow-exclude": true, "--reference": true,
	"--reference-if-able": true, "--separate-git-dir": true,
	"-c": true, "--config": true, "--template": true, "-j": true,
	"--jobs": true, "--filter": true, "--server-option": true,
	"--revision": true, "--recurse-submodules": true, "--ref-format": true,
	"--object-format": true,
}
