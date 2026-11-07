package hist

import (
	"strconv"
	"testing"
	"time"
)

func TestMergeParents(t *testing.T) {
	cases := []struct{ field, want string }{
		{"", ""},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ""},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "{aaaaaaa bbbbbbb}"},
		{"ccccccc ddddddd", "{ccccccc ddddddd}"},
	}
	for _, c := range cases {
		if got := mergeParents(c.field); got != c.want {
			t.Errorf("mergeParents(%q) = %q, want %q", c.field, got, c.want)
		}
	}
}

func TestAgeBuckets(t *testing.T) {
	now := time.Now()
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{5 * time.Minute, "5m"},
		{2 * time.Hour, "2h"},
		{3 * 24 * time.Hour, "3d"},
		{2 * 7 * 24 * time.Hour, "2w"},
		{90 * 24 * time.Hour, "3mo"},
	}
	for _, c := range cases {
		epoch := strconv.FormatInt(now.Add(-c.ago).Unix(), 10)
		if got := age(epoch); got != c.want {
			t.Errorf("age(%v ago) = %q, want %q", c.ago, got, c.want)
		}
	}
}

func TestAgeBadEpoch(t *testing.T) {
	if got := age("not-a-number"); got != "" {
		t.Errorf("age(garbage) = %q, want empty", got)
	}
}
