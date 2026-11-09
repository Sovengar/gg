package ops

import (
	"os"
	"path/filepath"
	"strings"

	"gg/internal/ui"
)

// shellWrapper is the fish function that turns `gg go` into a directory change.
// Every other verb and every `--` command runs untouched, and the wrapper only
// cds into a directory the binary actually printed.
const shellWrapper = `function gg --description 'gg, with a worktree-aware go'
    if test "$argv[1]" != go; or contains -- -- $argv
        command gg $argv
        return $status
    end
    set -l dir (command gg $argv)
    set -l rc $status
    if test $rc -eq 0 -a -d "$dir"
        cd $dir
    end
    return $rc
end
`

// Shell installs or removes the fish wrapper behind `gg go`. A program cannot
// change the shell that started it, so the wrapper is the only way `gg go` can
// leave the terminal in the worktree.
func Shell(args []string) int {
	path, err := shellPath()
	if err != nil {
		ui.Errf("gg shell: cannot find the fish config directory: %v", err)
		return 1
	}
	switch {
	case len(args) == 0, args[0] == "show":
		if _, err := os.Stat(path); err == nil {
			ui.Errf("gg shell: installed at %s", path)
			return 0
		}
		ui.Errf("gg shell: not installed. Run gg shell install")
		return 0
	case args[0] == "install":
		force := len(args) > 1 && (args[1] == "-f" || args[1] == "--force")
		if data, err := os.ReadFile(path); err == nil && !force {
			if string(data) != shellWrapper {
				ui.Errf("gg shell: %s exists and is not gg's wrapper. Use --force to replace it.", path)
				return 1
			}
			ui.Errf("gg shell: already installed at %s", path)
			return 0
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			ui.Errf("gg shell: %v", err)
			return 1
		}
		if err := os.WriteFile(path, []byte(shellWrapper), 0o644); err != nil {
			ui.Errf("gg shell: %v", err)
			return 1
		}
		ui.Errf("gg shell: installed %s", path)
		ui.Errf("gg shell: open a new fish session, then gg go <branch> changes directory")
		return 0
	case args[0] == "uninstall":
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			ui.Errf("gg shell: %v", err)
			return 1
		}
		ui.Errf("gg shell: removed %s", path)
		return 0
	default:
		ui.Errf("gg shell: %s. Use install, uninstall or show.", strings.Join(args, " "))
		return 2
	}
}

// shellPath is where fish keeps one function.
func shellPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fish", "functions", "gg.fish"), nil
}
