package files

import (
	"strings"
	"testing"

	"gg/internal/ids"
)

// diffPayload is one path with two separate hunks, as git diff -U3 prints them.
const diffPayload = `diff --git a/f.txt b/f.txt
index 1111111..2222222 100644
--- a/f.txt
+++ b/f.txt
@@ -1,3 +1,3 @@
 l1
-l2
+L2
 l3
@@ -20,3 +20,3 @@
 l20
-l21
+L21
 l22
`

func TestHunksOfSplitsAndIDs(t *testing.T) {
	hunks := split(diffPayload)
	if len(hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(hunks))
	}
	for _, h := range hunks {
		if len(h.ID) < ids.MintLen() || len(h.ID) > 40 {
			t.Fatalf("hunk id = %q, want a hex prefix", h.ID)
		}
		if !strings.Contains(h.Patch, "@@ -") {
			t.Fatalf("patch lost its hunk header: %q", h.Patch)
		}
	}
	if hunks[0].ID == hunks[1].ID {
		t.Fatal("two hunks share an id")
	}
	if !strings.Contains(hunks[0].Patch, "l1") || strings.Contains(hunks[0].Patch, "l20") {
		t.Fatal("the first patch carries the second hunk")
	}
}

func TestHunksOfStableIDs(t *testing.T) {
	first := split(diffPayload)
	second := split(diffPayload)
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Fatalf("hunk %d id moved between reads", i)
		}
	}
}

func TestUntrackedHasNoHunks(t *testing.T) {
	if got := split(""); got != nil {
		t.Fatalf("an empty diff yielded %v", got)
	}
}
