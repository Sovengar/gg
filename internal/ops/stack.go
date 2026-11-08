package ops

import (
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ids"
	"gg/internal/ui"
)

// stackNode is one branch and the branch its history sits on.
type stackNode struct {
	row    branchRow
	parent string
	ahead  int
}

// Stack prints the branches as stacks: each branch sits under the branch it
// was cut from, so a chain of branches reads top to bottom. The parent is
// derived, never stored: it is the branch whose tip shares the newest merge
// base with the child. gg therefore keeps no metadata of its own, and a stack
// made with plain git or with but shows up the same way.
func Stack() int {
	rows := branchRows()
	if len(rows) == 0 {
		ui.Errf("gg br -a: no branches yet")
		return 1
	}
	// a floor must not move with HEAD: the current-branch fallback would make
	// every branch the floor of its own stack. The "default" mark still reads
	// the full answer, so a repository with no conventional name keeps it
	floor := ConventionBranch()
	def := DefaultBranch()
	nodes := stackNodes(rows, floor)
	roots, kids := stackTree(rows, nodes)

	roots = defaultFirst(roots, def)
	bidx := branchIndexOrEmpty()
	width := stackWidth(roots, kids, nodes)

	fmt.Println(ui.Dim(fmt.Sprintf("  branch stacks (%d):", len(rows))))
	// A branch cut from a floor is parallel to its siblings, not stacked on
	// them, so it gets no indent: only a branch that sits on another branch
	// opens a level. guides carries one bit per open chain level: whether that
	// level still has a sibling below it.
	floorOf := make(map[string]bool, len(rows))
	for _, r := range rows {
		floorOf[r.name] = isStackFloor(r.name, floor)
	}
	var walk func(name string, guides []bool)
	walk = func(name string, guides []bool) {
		fmt.Println(stackLine(nodes[name], guides, width, bidx, def))
		children := kids[name]
		// a level opens only when the branch carries a single child: two
		// children are parallel branches, and indenting them would claim they
		// sit on each other
		opensLevel := !floorOf[name] && len(children) == 1
		for i, kid := range children {
			next := guides
			if opensLevel {
				next = append(append([]bool{}, guides...), i < len(children)-1)
			}
			walk(kid, next)
		}
	}
	for _, root := range roots {
		walk(root, nil)
	}
	return 0
}

// stackNodes derives the parent of every branch. The parent of b is the
// deepest branch tip that b already contains, so a stack is a chain of
// ancestor tips. Two branches that diverged from one point contain neither
// tip, so they stay siblings at the top of their own stack.
func stackNodes(rows []branchRow, floor string) map[string]*stackNode {
	nodes := make(map[string]*stackNode, len(rows))
	for _, r := range rows {
		nodes[r.name] = &stackNode{row: r}
	}
	for _, r := range rows {
		node := nodes[r.name]
		best, bestDepth := "", -1
		for _, other := range rows {
			if other.name == r.name || other.sha == r.sha {
				continue
			}
			// the default branch and but's own refs are floors, never children:
			// a merge can make the default contain any tip, and showing it
			// under that branch reads as if the stack grew from it
			if isStackFloor(r.name, floor) {
				continue
			}
			// only a tip the child contains can be its parent: two branches
			// cut from the same commit contain neither, and would otherwise
			// name each other and close a loop
			if !isAncestor(other.sha, r.sha) {
				continue
			}
			// the deepest tip wins: it is the branch this one was cut from.
			// two branches on one commit are equally deep, so the default
			// branch takes the tie and the line stays the same between runs
			depth, ok := gitx.Count(other.sha)
			if !ok || depth < bestDepth {
				continue
			}
			if depth == bestDepth && other.name != floor {
				continue
			}
			best, bestDepth = other.name, depth
		}
		node.parent = best
		if best == "" {
			continue
		}
		if ahead, ok := gitx.Count(best + ".." + r.sha); ok {
			node.ahead = ahead
		}
	}
	return nodes
}

// isStackFloor reports whether a branch is a floor a stack grows from: the
// default branch, and the refs but keeps for its target and its workspace.
func isStackFloor(name, floor string) bool {
	if floor != "" && name == floor {
		return true
	}
	return strings.HasPrefix(name, "gitbutler/")
}

// isAncestor reports whether the commit tip is reachable from sha.
func isAncestor(tip, sha string) bool {
	_, err := gitx.Output("merge-base", "--is-ancestor", tip, sha)
	return err == nil
}

// stackTree turns the parent links into a root list and a child list, both in
// the order branchRows already gives: newest committer date first.
func stackTree(rows []branchRow, nodes map[string]*stackNode) ([]string, map[string][]string) {
	roots := make([]string, 0, len(rows))
	kids := make(map[string][]string, len(rows))
	for _, r := range rows {
		parent := nodes[r.name].parent
		if parent == "" {
			roots = append(roots, r.name)
			continue
		}
		kids[parent] = append(kids[parent], r.name)
	}
	return roots, kids
}

// defaultFirst puts the default branch at the top of the root list, so the
// tree always opens on the branch every stack grows from.
func defaultFirst(roots []string, def string) []string {
	if def == "" {
		return roots
	}
	out := make([]string, 0, len(roots))
	for _, name := range roots {
		if name == def {
			out = append(out, name)
		}
	}
	for _, name := range roots {
		if name != def {
			out = append(out, name)
		}
	}
	return out
}

// stackWidth is the column the branch names end at: the widest name plus the
// indent its depth costs, so the mark that follows every name lines up.
func stackWidth(roots []string, kids map[string][]string, nodes map[string]*stackNode) int {
	width := 0
	var walk func(name string, depth int)
	walk = func(name string, depth int) {
		if w := len(name) + depth*2; w > width {
			width = w
		}
		for _, kid := range kids[name] {
			walk(kid, depth+1)
		}
	}

	for _, root := range roots {
		walk(root, 0)
	}
	return width
}

// stackLine renders one branch: its changeId, its name under a branch mark
// when it has a parent, then how many commits it holds above that parent.
func stackLine(n *stackNode, guides []bool, width int, bidx *ids.BranchIndex, def string) string {
	marker := " "
	if n.row.current {
		marker = "*"
	}
	// the name column is shared, so a child lines up under its parent whatever
	// its own name length is: the guides only move the box mark
	box := ""
	for i, more := range guides {
		switch {
		case i < len(guides)-1 && more:
			box += ui.Dim("│ ")
		case i < len(guides)-1:
			box += "  "
		case more:
			box += ui.Dim("├ ")
		default:
			box += ui.Dim("└ ")
		}
	}
	pad := width - len(n.row.name) - len(guides)*2
	if pad < 0 {
		pad = 0
	}
	line := fmt.Sprintf("  %s %s %s%s%s", ui.Green(marker), ui.Red(bidx.ChangeID(n.row.name)),
		box, n.row.name, strings.Repeat(" ", pad)+"  ")
	if n.parent == "" {
		line += ui.Dim("base")
	} else {
		line += ui.Dim(fmt.Sprintf("↑%d on %s", n.ahead, n.parent))
	}
	if n.row.upstream != "" {
		ahead, _ := gitx.Count(n.row.upstream + ".." + n.row.sha)
		behind, _ := gitx.Count(n.row.sha + ".." + n.row.upstream)
		line += "  " + ui.Dim(fmt.Sprintf("↑%d ↓%d", ahead, behind))
	}
	if def != "" && n.row.name == def {
		line += "  " + ui.Dim("default")
	}
	return line
}

// StackGroups groups branch names into stacks, leaf first: each group is one
// branch and the branches it is stacked on, up to the branch a stack grows
// from. Two branches cut from the same commit land in two groups, because each
// one is its own stack. A name with no branch behind it, such as a detached
// HEAD, gets a group of its own so nothing disappears from the tree.
func StackGroups(names []string) [][]string {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	rows := make([]branchRow, 0, len(names))
	for _, r := range branchRows() {
		if want[r.name] {
			rows = append(rows, r)
		}
	}
	nodes := stackNodes(rows, ConventionBranch())

	present := make(map[string]bool, len(rows))
	for _, r := range rows {
		present[r.name] = true
	}
	parent := make(map[string]string, len(rows))
	hasChild := make(map[string]bool, len(rows))
	for _, r := range rows {
		p := nodes[r.name].parent
		if p == "" || !present[p] {
			continue
		}
		parent[r.name] = p
		hasChild[p] = true
	}

	groups := make([][]string, 0, len(names))
	for _, n := range names {
		if !present[n] {
			groups = append(groups, []string{n})
			continue
		}
		if hasChild[n] {
			// a branch under another one is drawn inside that one's stack
			continue
		}
		chain := []string{n}
		seen := map[string]bool{n: true}
		for p := parent[n]; p != "" && !seen[p]; p = parent[p] {
			chain = append(chain, p)
			seen[p] = true
		}
		groups = append(groups, chain)
	}
	return groups
}
