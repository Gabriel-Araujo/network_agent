package validate

import (
	"strings"
	"testing"
)

// The wrapper must keep the snippet's line numbering recoverable: snippet
// line N must sit at wrapped line N+offset, so callers can subtract the
// returned offset from vtysh-reported lines.
func TestWrapSnippetLineMapping(t *testing.T) {
	// leading blank line on purpose: wrapping must not trim the snippet
	snippet := "\nneighbor 10.0.0.1 remote-as 65001\n  network 10.0.0.0/24\n"
	wrapped, offset := WrapSnippet("bgp", snippet)
	if offset != 1 {
		t.Fatalf("offset = %d, want 1", offset)
	}
	lines := strings.Split(wrapped, "\n")
	for i, want := range strings.Split(snippet, "\n") {
		got := lines[i+offset]
		if got != " "+want {
			t.Errorf("snippet line %d at wrapped[%d] = %q, want %q", i+1, i+offset, got, " "+want)
		}
	}
}

func TestWrapSnippetNoWrap(t *testing.T) {
	cases := []struct {
		name, protocol, snippet string
	}{
		{"already has context", "bgp", "router bgp 65000\n  neighbor 10.0.0.1 remote-as 65001\n"},
		{"interface context", "bgp", "interface eth0\n  ip address 10.0.0.1/24\n"},
		{"unknown protocol", "static", "ip route 10.0.0.0/24 10.0.0.1\n"},
		{"empty", "bgp", "  \n"},
	}
	for _, c := range cases {
		wrapped, offset := WrapSnippet(c.protocol, c.snippet)
		if offset != 0 {
			t.Errorf("%s: offset = %d, want 0", c.name, offset)
		}
		if wrapped != c.snippet {
			t.Errorf("%s: snippet was modified: %q", c.name, wrapped)
		}
	}
}
