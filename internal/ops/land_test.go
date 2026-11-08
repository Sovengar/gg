package ops

import "testing"

func TestLandArgs(t *testing.T) {
	cases := []struct {
		args   []string
		branch string
		target string
		yes    bool
		bad    bool
	}{
		{nil, "", "", false, false},
		{[]string{"side"}, "side", "", false, false},
		{[]string{"side", "-y"}, "side", "", true, false},
		{[]string{"--yes", "side"}, "side", "", true, false},
		{[]string{"side", "-b", "main"}, "side", "main", false, false},
		{[]string{"-b", "release", "-y", "side"}, "side", "release", true, false},
		{[]string{"--onto", "main", "side"}, "side", "main", false, false},
		{[]string{"side", "other"}, "", "", false, true},
		{[]string{"side", "-b"}, "", "", false, true},
		{[]string{"side", "-b", "a", "--onto", "b"}, "", "", false, true},
		{[]string{"--nope"}, "", "", false, true},
	}
	for _, c := range cases {
		branch, target, yes, err := landArgs(c.args)
		if c.bad {
			if err == nil {
				t.Errorf("landArgs(%q) = %q, %q, %v; want an error", c.args, branch, target, yes)
			}
			continue
		}
		if err != nil {
			t.Errorf("landArgs(%q) failed: %v", c.args, err)
			continue
		}
		if branch != c.branch || target != c.target || yes != c.yes {
			t.Errorf("landArgs(%q) = %q, %q, %v; want %q, %q, %v",
				c.args, branch, target, yes, c.branch, c.target, c.yes)
		}
	}
}
