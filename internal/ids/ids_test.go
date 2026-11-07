package ids

import (
	"strings"
	"testing"
)

const sha = "641682b7a6c27c7273086ea06574b451742d5792"

func TestRawIsStable(t *testing.T) {
	first := Raw("Change-Id: I1234")
	if first != Raw("Change-Id: I1234") {
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
	if Raw("Change-Id: I1234") == Raw("Change-Id: I5678") {
		t.Fatal("different identities produced the same raw handle")
	}
}

func TestShaped(t *testing.T) {
	good := []string{"ab", "abc", "abcd"}
	bad := []string{"a", "abcde", "AB", "a-b", "@@", "a1"}
	for _, g := range good {
		if !Shaped(g) {
			t.Errorf("%q should be shaped like a handle", g)
		}
	}
	for _, b := range bad {
		if Shaped(b) {
			t.Errorf("%q should not be shaped like a handle", b)
		}
	}
}

func TestHandleFallsBackToSHA(t *testing.T) {
	ix := &Index{}
	got := ix.Handle(sha)
	if len(got) < MinLen || len(got) > MaxLen {
		t.Fatalf("handle = %q, want %d..%d chars", got, MinLen, MaxLen)
	}
	if !strings.Contains(ix.Short(sha), sha[:7]) {
		t.Errorf("Short = %q, want the abbreviated sha", ix.Short(sha))
	}
}
