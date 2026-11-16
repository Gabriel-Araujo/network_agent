package main

import (
	"fmt"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

type mermaidSource struct {
	Source string
	start  int
	end    int
	prefix string
	empty  bool
}

func mermaidSources(content string) ([]mermaidSource, error) {
	source := []byte(content)
	root := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(source))
	blocks := []mermaidSource{}
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		code, ok := node.(*ast.FencedCodeBlock)
		if !ok || string(code.Language(source)) != "mermaid" {
			return ast.WalkContinue, nil
		}
		value := strings.ReplaceAll(string(code.Lines().Value(source)), "\r\n", "\n")
		value = strings.TrimSuffix(value, "\n")
		block := mermaidSource{Source: value}
		if code.Lines().Len() == 0 {
			if code.Info == nil {
				return ast.WalkStop, fmt.Errorf("missing fence information")
			}
			infoEnd := code.Info.Segment.Stop
			endOfOpening := strings.IndexByte(content[infoEnd:], '\n')
			if endOfOpening < 0 {
				return ast.WalkStop, fmt.Errorf("unterminated empty fence")
			}
			block.start = infoEnd + endOfOpening + 1
			block.end = block.start
			openingStart := strings.LastIndexByte(content[:infoEnd], '\n') + 1
			opening := content[openingStart:infoEnd]
			fence := strings.IndexAny(opening, "`~")
			if fence < 0 {
				return ast.WalkStop, fmt.Errorf("missing opening fence")
			}
			block.prefix = opening[:fence]
			block.empty = true
		} else {
			first := code.Lines().At(0)
			last := code.Lines().At(code.Lines().Len() - 1)
			block.start, block.end = first.Start, last.Stop
			lineStart := strings.LastIndexByte(content[:block.start], '\n') + 1
			block.prefix = content[lineStart:block.start]
		}
		blocks = append(blocks, block)
		return ast.WalkContinue, nil
	})
	return blocks, err
}

func replaceMermaidSources(content string, replacements []history.MermaidReplacement) (string, error) {
	blocks, err := mermaidSources(content)
	if err != nil {
		return "", err
	}
	wanted := make(map[int]string, len(replacements))
	for _, replacement := range replacements {
		if replacement.Index < 1 || replacement.Index > len(blocks) {
			return "", fmt.Errorf("invalid Mermaid block index")
		}
		if _, exists := wanted[replacement.Index]; exists {
			return "", fmt.Errorf("duplicate Mermaid block index")
		}
		wanted[replacement.Index] = replacement.Source
	}
	var result strings.Builder
	last := 0
	for i, block := range blocks {
		replacement, found := wanted[i+1]
		if !found {
			continue
		}
		result.WriteString(content[last:block.start])
		if block.empty {
			result.WriteString(block.prefix)
		}
		result.WriteString(strings.ReplaceAll(replacement, "\n", "\n"+block.prefix))
		result.WriteByte('\n')
		last = block.end
	}
	result.WriteString(content[last:])
	updated := result.String()
	after, err := mermaidSources(updated)
	if err != nil || len(after) != len(blocks) {
		return "", fmt.Errorf("replacement changed Markdown structure")
	}
	for i, block := range after {
		expected := blocks[i].Source
		if replacement, found := wanted[i+1]; found {
			expected = replacement
		}
		if block.Source != expected {
			return "", fmt.Errorf("replacement changed Mermaid block %d", i+1)
		}
	}
	return updated, nil
}
