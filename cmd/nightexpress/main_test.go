package main

import (
	"os"
	"reflect"
	"testing"

	"nightexpress/engine"
)

// quiet runs fn with stdout and stderr sent to /dev/null.
func quiet(t *testing.T, fn func()) {
	t.Helper()
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = null, null
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	fn()
}

func TestExitCodes(t *testing.T) {
	t.Setenv(engine.HomeEnv, t.TempDir())
	cases := []struct {
		args []string
		want int
	}{
		{[]string{"--lang", "ru", "--script", "3,1,2,1,1,5,2,7,2,1,2,6,1"}, 0}, // E7
		{[]string{"--script", "1,1,1"}, 1},                                     // runs out before an ending
		{[]string{"--script", "9"}, 1},                                         // invalid choice
		{[]string{"--lang", "fr"}, 2},
		{[]string{"--no-such-flag"}, 2},
	}
	for _, c := range cases {
		var got int
		quiet(t, func() { got = run(c.args) })
		if got != c.want {
			t.Errorf("run(%q) = %d, want %d", c.args, got, c.want)
		}
	}
}

func TestSplitScript(t *testing.T) {
	got := splitScript(" 1, 2,j ,, lang ru ,3")
	want := []string{"1", "2", "j", "lang ru", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitScript = %q, want %q", got, want)
	}
}
