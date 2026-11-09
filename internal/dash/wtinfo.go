package dash

import (
	"fmt"
	"strings"

	"gg/internal/ops"
	"gg/internal/ui"
)

// wtNotes maps a branch to the drift Worktrunk reports for its worktree:
// ↑ commits the main branch lacks, ↓ commits it already has, ⚑ changes in the
// tree. A branch with nothing to report gets no entry, so the header stays as
// git draws it. Current worktree dirty state is skipped: the uncommitted lane
// already lists it.
func wtNotes() map[string]string {
	out := map[string]string{}
	for _, w := range ops.WtTrees() {
		if w.Branch == "" {
			continue
		}
		var parts []string
		if w.Main.Ahead > 0 {
			parts = append(parts, ui.Cyan(fmt.Sprintf("↑%d", w.Main.Ahead)))
		}
		if w.Main.Behind > 0 {
			parts = append(parts, ui.Yellow(fmt.Sprintf("↓%d", w.Main.Behind)))
		}
		if w.Dirty() && !w.IsCurrent {
			parts = append(parts, ui.Yellow("⚑"))
		}
		if len(parts) > 0 {
			out[w.Branch] = strings.Join(parts, " ")
		}
	}
	return out
}
