package ids

import (
	"strings"
	"testing"
)

const sha = "641682b7a6c27c7273086ea06574b451742d5792"

func TestRawIsStable(t *testing.T) {
	first := RawChangeID("Change-Id: I1234")
	if first != RawChangeID("Change-Id: I1234") {
		t.Fatal("raw identity must be stable")
	}
	if len(first) != MaxLen {
		t.Fatalf("raw = %q, want %d chars", first, MaxLen)
	}
	for _, r := range first {
		if !strings.ContainsRune(alphabet, r) {
			t.Fatalf("raw %q contains %q outside the alphabet", first, r)
		}
	}
	if RawChangeID("Change-Id: I1234") == RawChangeID("Change-Id: I5678") {
		t.Fatal("different identities produced the same raw changeId")
	}
}

func TestShaped(t *testing.T) {
	good := []string{"k", "kl", "klm", "zzzz"}
	bad := []string{"", "a", "AB", "a-b", "@@", "k1", strings.Repeat("k", MaxLen+1)}
	for _, g := range good {
		if !Shaped(g) {
			t.Errorf("%q should be shaped like a changeId", g)
		}
	}
	for _, b := range bad {
		if Shaped(b) {
			t.Errorf("%q should not be shaped like a changeId", b)
		}
	}
}

func TestChangeIDFallsBackToSHA(t *testing.T) {
	ix := &Index{}
	got := ix.ChangeID(sha)
	if len(got) < MinLen || len(got) > MaxLen {
		t.Fatalf("changeId = %q, want %d..%d chars", got, MinLen, MaxLen)
	}
	if !strings.Contains(ix.Short(sha), sha[:7]) {
		t.Errorf("Short = %q, want the abbreviated sha", ix.Short(sha))
	}
}
