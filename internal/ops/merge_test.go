package ops

import "testing"

func TestHasMergeFlag(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"side"}, false},
		{[]string{"--no-ff", "side"}, true},
		{[]string{"side", "--ff=only"}, true},
		{[]string{"--ff-only", "side"}, true},
		{[]string{"--squash", "side"}, true},
		{[]string{"--abort"}, true},
		{[]string{"-m", "msg", "side"}, true},
		{[]string{"--strategy=recursive", "side"}, true},
	}
	for _, c := range cases {
		if got := hasMergeFlag(c.args); got != c.want {
			t.Errorf("hasMergeFlag(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestMergeBranch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"side"}, "side"},
		{[]string{"--no-ff", "side"}, "side"},
		{[]string{"--abort"}, "<branch>"},
		{nil, "<branch>"},
	}
	for _, c := range cases {
		if got := mergeBranch(c.args); got != c.want {
			t.Errorf("mergeBranch(%q) = %q, want %q", c.args, got, c.want)
		}
	}
}
