package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHandleVersionArgs(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-version"} {
		t.Run(arg, func(t *testing.T) {
			var out bytes.Buffer
			if !handleVersionArgs([]string{arg}, &out) {
				t.Fatal("expected version argument to be handled")
			}
			if got := strings.TrimSpace(out.String()); !strings.HasPrefix(got, "zuelose version ") {
				t.Fatalf("unexpected version output %q", got)
			}
		})
	}
}

func TestHandleVersionArgsLeavesApplicationArgumentsAlone(t *testing.T) {
	for _, args := range [][]string{nil, {"update"}, {"version", "extra"}, {"--unknown"}} {
		var out bytes.Buffer
		if handleVersionArgs(args, &out) {
			t.Fatalf("expected arguments %q not to be handled", args)
		}
		if out.Len() != 0 {
			t.Fatalf("unexpected output for arguments %q: %q", args, out.String())
		}
	}
}
