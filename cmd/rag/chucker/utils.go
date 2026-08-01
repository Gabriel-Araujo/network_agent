package main

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var clicmdRE = regexp.MustCompile(`^\.\. clicmd::\s*(.+)$`)

const headerUnderlineChars = "=-~^\"'`#*+"

var slugifyRE = regexp.MustCompile(`[^a-z0-9]+`)

// countTokens é uma aproximação simples (runes/4), sem depender de download
// de tokenizer. Pra produção, troque pelo tokenizer real do seu modelo de
// embedding.
func countTokens(text string) int {
	n := utf8.RuneCountInString(text) / 4
	if n < 1 {
		return 1
	}
	return n
}

func allSameRune(runes []rune) bool {
	for _, r := range runes[1:] {
		if r != runes[0] {
			return false
		}
	}
	return true
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func slugify(text string, maxlen int) string {
	s := slugifyRE.ReplaceAllString(strings.ToLower(text), "-")
	s = strings.Trim(s, "-")
	if len(s) > maxlen {
		s = s[:maxlen]
	}
	return s
}

// dedent remove a indentação comum a todas as linhas,
// ignorando linhas em branco no cálculo da margem.
func dedent(text string) string {
	lines := strings.Split(text, "\n")
	var margin string
	marginSet := false
	for _, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" {
			continue
		}
		indent := line[:len(line)-len(stripped)]
		if !marginSet {
			margin = indent
			marginSet = true
		} else {
			margin = commonPrefix(margin, indent)
		}
	}
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" {
			lines[i] = ""
			continue
		}
		if margin != "" && strings.HasPrefix(line, margin) {
			lines[i] = line[len(margin):]
		}
	}
	return strings.Join(lines, "\n")
}

func commonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[:i]
}
