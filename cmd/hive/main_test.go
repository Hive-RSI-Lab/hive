package main

import "testing"

func TestKBArgsTakeLimitAnywhere(t *testing.T) {
	for _, args := range [][]string{
		{"login", "--limit", "all"},
		{"--limit", "all", "login"},
	} {
		if q, l := kbArgs(args); q != "login" || l != "all" {
			t.Errorf("kbArgs(%q) = %q, %q; want login, all", args, q, l)
		}
	}
}

func TestReleaseArgRefusesWhatItCannotRead(t *testing.T) {
	for _, c := range []struct {
		args []string
		tag  string
		ok   bool
	}{
		{nil, "", true},
		{[]string{"--release", "v0.1.2"}, "v0.1.2", true},
		{[]string{"--release=v0.1.2"}, "v0.1.2", true},
		{[]string{"--release"}, "", false},
		{[]string{"--release="}, "", false},
		{[]string{"v0.1.2"}, "", false},
		{[]string{"--relase", "v0.1.2"}, "", false},
	} {
		if tag, ok := releaseArg(c.args); tag != c.tag || ok != c.ok {
			t.Errorf("releaseArg(%q) = %q, %v; want %q, %v", c.args, tag, ok, c.tag, c.ok)
		}
	}
}

// approve and refuse never guess on an ask with one option (a "told" seen),
// and say what an answer means where its word doesn't.
func TestApproveRefusePick(t *testing.T) {
	if _, err := pick("refuse", []string{"seen"}); err == nil {
		t.Fatal("refuse answered a one-option ask")
	}
	if _, err := pick("approve", nil); err == nil {
		t.Fatal("approve answered an ask with no options")
	}
	if o, _ := pick("approve", []string{"done", "declined"}); o != "done" || meaning(o) == "" {
		t.Fatalf("approve picked %q, meaning %q", o, meaning(o))
	}
	if o, _ := pick("refuse", []string{"run", "declined"}); o != "declined" {
		t.Fatalf("refuse picked %q", o)
	}
}
