package ids

import "testing"

func TestShortestChangeIDIsUnique(t *testing.T) {
	raws := map[string]string{
		"main":  RawChangeID("branch:main"),
		"trunk": RawChangeID("branch:trunk"),
		"twin":  RawChangeID("branch:twin"),
	}
	owners := prefixOwners(raws)
	seen := make(map[string]string, len(raws))
	for name, raw := range raws {
		h := shortestChangeID(owners, raw)
		if len(h) < MinLen || len(h) > MaxLen {
			t.Fatalf("%s changeId = %q, want %d..%d chars", name, h, MinLen, MaxLen)
		}
		if other, dup := seen[h]; dup {
			t.Fatalf("%s and %s share the changeId %q", name, other, h)
		}
		seen[h] = name
	}
}

func TestBranchChangeIDIsStable(t *testing.T) {
	raw := RawChangeID("branch:topic")
	if shortestChangeID(prefixOwners(map[string]string{"topic": raw}), raw) != raw[:MinLen] {
		t.Fatal("a lone branch should take the shortest changeId")
	}
	if RawChangeID("branch:a") == RawChangeID("branch:b") {
		t.Fatal("different branch names produced the same raw changeId")
	}
}
