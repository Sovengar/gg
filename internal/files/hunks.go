package files

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"gg/internal/gitx"
	"gg/internal/ids"
)

// Hunk is one region of a dirty tracked path. Its id lives in the file's
// namespace, so the full handle reads `fileId:hunkId`, and Patch reverts only
// this region.
type Hunk struct {
	ID    string
	Patch string
}

// HunksOf lists the hunks of one dirty tracked path, with the context lines the
// caller draws. It returns nothing for a clean or untracked path: a whole new
// file is one change, not a region of one.
func HunksOf(path string, context int, extra ...string) []Hunk {
	args := append([]string{"diff", "--no-color", "--no-ext-diff",
		fmt.Sprintf("--unified=%d", context)}, extra...)
	out, err := gitx.Output(append(args, "--", path)...)
	if err != nil || out == "" {
		return nil
	}
	return split(out)
}

// split reads one file's diff into hunks, each with its own patch.
func split(diff string) []Hunk {
	var header []string
	var hunks []Hunk
	var body []string
	flush := func() {
		if len(body) == 0 {
			return
		}
		patch := strings.Join(append(append([]string{}, header...), body...), "\n") + "\n"
		hunks = append(hunks, Hunk{ID: hunkID(patch), Patch: patch})
		body = nil
	}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "),
			strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "+++ "):
			header = append(header, line)
		case strings.HasPrefix(line, "@@ "):
			flush()
			body = []string{line}
		case body != nil:
			body = append(body, line)
		}
	}
	flush()
	return uniqueHunkIDs(hunks)
}

// hunkID is the sha1 of the hunk body, without its `@@` header, in hex: the
// scheme but uses for hunks. The short id is a prefix of it.
func hunkID(patch string) string {
	var body strings.Builder
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "@@") {
			continue
		}
		body.WriteString(line)
		body.WriteString("\n")
	}
	sum := sha1.Sum([]byte(body.String()))
	return hex.EncodeToString(sum[:])
}

// uniqueHunkIDs mints one short id per hunk: a hex prefix as wide as the ids
// around it allow, starting at the mint width. Two hunks with the same body get
// but's `#index-total` suffix.
func uniqueHunkIDs(hunks []Hunk) []Hunk {
	counts := map[string]int{}
	for _, h := range hunks {
		counts[h.ID]++
	}
	index := map[string]int{}
	seen := map[string]bool{}
	for i := range hunks {
		full := hunks[i].ID
		n := ids.MintLen()
		if n > len(full) {
			n = len(full)
		}
		id := full[:n]
		for seen[id] && n < len(full) {
			n++
			id = full[:n]
		}
		if counts[full] > 1 {
			id = fmt.Sprintf("%s#%d-%d", id, index[full], counts[full])
			index[full]++
		}
		seen[id] = true
		hunks[i].ID = id
	}
	return hunks
}

// HunkHandle is the handle a user types for one hunk: file changeId, colon,
// hunk id.
func HunkHandle(ix *Index, path, hunkID string) string {
	return ix.ChangeID(path) + ":" + hunkID
}
