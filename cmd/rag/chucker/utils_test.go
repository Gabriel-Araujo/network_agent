package main

import "testing"

func TestSourceURLFor(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{
			"C:/src/frr/doc/user/bgp.rst",
			"https://github.com/FRRouting/frr/blob/master/doc/user/bgp.rst",
		},
		{
			"doc/user/ospf.rst",
			"https://github.com/FRRouting/frr/blob/master/doc/user/ospf.rst",
		},
		{
			"/home/u/frr/doc/developer/workflow.rst",
			"https://github.com/FRRouting/frr/blob/master/doc/developer/workflow.rst",
		},
		{
			// fora de doc/ — assume o corpus padrão doc/user/
			"C:/tmp/isis.rst",
			"https://github.com/FRRouting/frr/blob/master/doc/user/isis.rst",
		},
	}
	for _, c := range cases {
		if got := sourceURLFor(c.path); got != c.want {
			t.Errorf("sourceURLFor(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
