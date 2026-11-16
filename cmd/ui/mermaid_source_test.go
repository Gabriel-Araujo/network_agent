package main

import (
	"reflect"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/history"
)

func TestMermaidSourceExtractionAndReplacement(t *testing.T) {
	cases := []struct {
		name, text, source, replacement, expected string
	}{
		{"plain", "Before\n\n```mermaid\nflowchart LR\nA-->B\n```\nAfter", "flowchart LR\nA-->B", "flowchart LR\nA-->C", "Before\n\n```mermaid\nflowchart LR\nA-->C\n```\nAfter"},
		{"blockquote", "> ```mermaid\n> graph LR\n> A-->B\n> ```\n\nAfter", "graph LR\nA-->B", "graph LR\nA-->C", "> ```mermaid\n> graph LR\n> A-->C\n> ```\n\nAfter"},
		{"empty", "```mermaid\n```\n", "", "graph LR\nA-->B", "```mermaid\ngraph LR\nA-->B\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blocks, err := mermaidSources(tc.text)
			if err != nil || len(blocks) != 1 || blocks[0].Source != tc.source {
				t.Fatalf("blocks=%+v err=%v", blocks, err)
			}
			got, err := replaceMermaidSources(tc.text, []history.MermaidReplacement{{Index: 1, Source: tc.replacement}})
			if err != nil || got != tc.expected {
				t.Fatalf("result=%q err=%v, want %q", got, err, tc.expected)
			}
		})
	}
}

func TestMermaidSourceIgnoresOtherCodeAndRejectsFenceInjection(t *testing.T) {
	text := "```text\n```mermaid\n```\n\n```mermaid\ngraph LR\nA-->B\n```\n\n```mermaid\ngraph LR\nB-->C\n```"
	blocks, err := mermaidSources(text)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(blocks))
	for i, b := range blocks {
		got[i] = b.Source
	}
	if !reflect.DeepEqual(got, []string{"graph LR\nA-->B", "graph LR\nB-->C"}) {
		t.Fatalf("sources=%q", got)
	}
	if _, err := replaceMermaidSources(text, []history.MermaidReplacement{{Index: 1, Source: "graph LR\n```\nchanged prose"}}); err == nil {
		t.Fatal("accepted replacement that escapes its fence")
	}
}
