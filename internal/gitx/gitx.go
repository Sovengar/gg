// Package gitx is a thin wrapper over the git binary.
package gitx

import (
	"errors"
	"os"
	"os/exec"
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

// Snapshot stores HEAD in refs/gg/undo/prev so that undo can return here.
func Snapshot() {
	if sha, ok := Head(); ok {
		_, _ = Output("update-ref", "refs/gg/undo/prev", sha)
	}
}

// Prev returns the commit saved by the last Snapshot.
func Prev() (string, bool) {
	sha, err := Output("rev-parse", "--verify", "--quiet", "refs/gg/undo/prev")
	if err != nil || sha == "" {
		return "", false
	}
	return sha, true
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
