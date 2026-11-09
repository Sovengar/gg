// Package gitx is a thin wrapper over the git binary.
package gitx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Run executes git with inherited stdio and returns its exit code.
func Run(args ...string) int {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return exitCode(cmd.Run())
}

// RunBin executes an external binary with inherited stdio.
func RunBin(bin string, args ...string) int {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return exitCode(cmd.Run())
}

// Output runs git and returns trimmed stdout.
func Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// OutputAll runs git and returns a line slice.
func OutputAll(args ...string) ([]string, error) {
	out, err := Output(args...)
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// OutputBin runs an external binary and returns trimmed stdout.
func OutputBin(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

// InRepo reports whether the current directory is inside a work tree.
func InRepo() bool {
	_, err := Output("rev-parse", "--git-dir")
	return err == nil
}

// Head returns the current commit hash.
func Head() (string, bool) {
	sha, err := Output("rev-parse", "--verify", "--quiet", "HEAD")
	if err != nil || sha == "" {
		return "", false
	}
	return sha, true
}

// ShortHead returns the abbreviated current commit hash.
func ShortHead() (string, bool) {
	sha, err := Output("rev-parse", "--short", "HEAD")
	if err != nil || sha == "" {
		return "", false
	}
	return sha, true
}

// WipRef is the old shared marker: one wip for the whole repository, so two
// worktrees fought over it. WipSha still reads it, MarkWip drops it.
const WipRef = "refs/gg/wip"

// wipFile is the per-worktree marker, so every checkout keeps its own wip.
func wipFile() (string, bool) {
	path, err := Output("rev-parse", "--git-path", "gg/wip")
	if err != nil || path == "" {
		return "", false
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs, true
	}
	return path, true
}

// WipSha returns the wip commit when the marker exists.
func WipSha() (string, bool) {
	if path, ok := wipFile(); ok {
		if b, err := os.ReadFile(path); err == nil {
			if sha := strings.TrimSpace(string(b)); sha != "" {
				return sha, true
			}
		}
	}
	sha, err := Output("rev-parse", "--verify", "--quiet", WipRef)
	if err != nil || sha == "" {
		return "", false
	}
	return sha, true
}

// InWip reports whether HEAD is the wip commit.
func InWip() bool {
	head, ok := Head()
	if !ok {
		return false
	}
	wip, ok := WipSha()
	return ok && wip == head
}

// MarkWip points the marker at HEAD.
func MarkWip() {
	sha, ok := Head()
	if !ok {
		return
	}
	path, ok := wipFile()
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(path, []byte(sha+"\n"), 0o644); err != nil {
		return
	}
	_, _ = Output("update-ref", "-d", WipRef) // the shared marker is retired
}

// DropWip deletes the marker.
func DropWip() {
	if path, ok := wipFile(); ok {
		_ = os.Remove(path)
	}
	_, _ = Output("update-ref", "-d", WipRef)
}

// ClearStaleWip drops the marker when HEAD is no longer the wip commit.
func ClearStaleWip() {
	sha, ok := WipSha()
	if !ok {
		return
	}
	if head, ok := Head(); !ok || head != sha {
		DropWip()
	}
}

// Unmerged reports whether the index holds a conflicted path. Committing then
// writes the conflict markers into the commit, so gg new must refuse.
func Unmerged() bool {
	out, err := Output("ls-files", "-u")
	return err == nil && out != ""
}

// Dirty reports whether the working tree has changes.
func Dirty() bool {
	out, err := Output("status", "--porcelain")
	return err == nil && out != ""
}

// Upstream returns the full upstream ref, e.g. origin/main.
func Upstream() (string, bool) {
	name, err := Output("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil || name == "" {
		return "", false
	}
	return name, true
}

// Count returns how many commits are in the range.
func Count(rng string) (int, bool) {
	out, err := Output("rev-list", "--count", rng)
	if err != nil || out == "" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, false
	}
	return n, true
}
