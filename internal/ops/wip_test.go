package ops

import "testing"

func TestWipArgs(t *testing.T) {
	cases := []struct {
		args  []string
		msg   string
		force bool
		bad   bool
	}{
		{nil, "wip", false, false},
		{[]string{"-m", "park"}, "park", false, false},
		{[]string{"--message", "two words"}, "two words", false, false},
		{[]string{"--message=inline"}, "inline", false, false},
		{[]string{"-f", "-m", "again"}, "again", true, false},
		{[]string{"--force"}, "wip", true, false},
		{[]string{"-m"}, "", false, true},
		{[]string{"--help"}, "", false, true},
		{[]string{"stray"}, "", false, true},
	}
	for _, c := range cases {
		msg, force, err := wipArgs(c.args)
		if c.bad {
			if err == nil {
				t.Errorf("wipArgs(%q) = %q, %v; want an error", c.args, msg, force)
			}
			continue
		}
		if err != nil {
			t.Errorf("wipArgs(%q) failed: %v", c.args, err)
			continue
		}
		if msg != c.msg || force != c.force {
			t.Errorf("wipArgs(%q) = %q, %v; want %q, %v", c.args, msg, force, c.msg, c.force)
		}
	}
}
