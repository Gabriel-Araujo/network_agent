package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Gabriel-Araujo/network_agent/internal/rag"
)

// parseSections detecta headers RST e monta a árvore de breadcrumbs. O
// nível de cada caractere de sublinhado é definido pela ordem em que
// aparece pela primeira vez no documento (convenção do docutils).
func parseSections(lines []string) []rag.Section {
	levelOfChar := map[rune]int{}
	nextLevel := 1
	var headerLines []rag.HeaderLine

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		runes := []rune(line)
		if len(runes) >= 3 && allSameRune(runes) && strings.ContainsRune(headerUnderlineChars, runes[0]) {
			prev := lines[i-1]
			prevTrim := strings.TrimSpace(prev)
			if prevTrim != "" && absInt(len(runes)-utf8.RuneCountInString(prevTrim)) <= 2 {
				ch := runes[0]
				lvl, ok := levelOfChar[ch]
				if !ok {
					lvl = nextLevel
					levelOfChar[ch] = lvl
					nextLevel++
				}
				headerLines = append(headerLines, rag.HeaderLine{TitleLine: i - 1, Level: lvl, Title: prevTrim})
			}
		}
	}

	var sections []rag.Section

	var stack []rag.StackEntry
	for idx, hl := range headerLines {
		for len(stack) > 0 && stack[len(stack)-1].Level >= hl.Level {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, rag.StackEntry{Level: hl.Level, Title: hl.Title})
		path := make([]string, len(stack))
		for i, s := range stack {
			path[i] = s.Title
		}

		contentStart := hl.TitleLine + 2 // pula título + linha de sublinhado
		var contentEnd int
		if idx+1 < len(headerLines) {
			contentEnd = headerLines[idx+1].TitleLine
		} else {
			contentEnd = len(lines)
		}

		sections = append(sections, rag.Section{
			Level:     hl.Level,
			Title:     hl.Title,
			Path:      path,
			StartLine: contentStart,
			EndLine:   contentEnd,
		})
	}
	return sections
}

// extractClicmdsAndProse separa o conteúdo próprio de uma seção em blocos
// `.. clicmd::` e prosa. O corpo de um clicmd é toda linha seguinte em
// branco OU indentada; ele termina na primeira linha não-vazia com
// indentação zero — é essa regra que mantém comando+descrição+exemplo
// juntos como uma unidade atômica.
func extractClicmdsAndProse(lines []string, start, end int) ([]rag.ClicmdEntry, []string) {
	var clicmds []rag.ClicmdEntry
	var proseLines []string
	i := start
	for i < end {
		line := lines[i]
		m := clicmdRE.FindStringSubmatch(line)
		if m != nil {
			command := strings.TrimSpace(m[1])
			var body []string
			j := i + 1
			for j < end {
				l := lines[j]
				if strings.TrimSpace(l) == "" || strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
					body = append(body, l)
					j++
				} else {
					break
				}
			}
			clicmds = append(clicmds, rag.ClicmdEntry{Command: command, Body: body})
			i = j
		} else {
			proseLines = append(proseLines, line)
			i++
		}
	}
	return clicmds, proseLines
}

// splitProseIntoBlocks agrupa a prosa em blocos indivisíveis: um
// parágrafo, ou uma diretiva (code-block, note, option...) junto com todo
// o corpo indentado dela. Só corta num blank line quando o que vem depois
// NÃO é indentado — ou seja, nunca separa um code-block ao meio por causa
// de uma linha em branco interna a ele.
func splitProseIntoBlocks(proseLines []string) []string {
	var blocks []string
	var current []string
	n := len(proseLines)
	i := 0
	for i < n {
		line := proseLines[i]
		current = append(current, line)
		i++
		if strings.TrimSpace(line) == "" {
			if i < n && (strings.HasPrefix(proseLines[i], " ") || strings.HasPrefix(proseLines[i], "\t") || strings.TrimSpace(proseLines[i]) == "") {
				continue
			}
			if strings.TrimSpace(strings.Join(current, "")) != "" {
				blocks = append(blocks, strings.Trim(strings.Join(current, "\n"), "\n"))
			}
			current = nil
		}
	}
	if strings.TrimSpace(strings.Join(current, "")) != "" {
		blocks = append(blocks, strings.Trim(strings.Join(current, "\n"), "\n"))
	}

	var result []string
	for _, b := range blocks {
		if strings.TrimSpace(b) != "" {
			result = append(result, b)
		}
	}
	return result
}

var anchorRE = regexp.MustCompile(`^\.\. _[\w-]+:\s*$`)
var roleRE = regexp.MustCompile(":(\\w+):`([^`]+)`")

// stripAnchorLines remove âncoras de referência cruzada do RST
// (`.. _label:`) — são metadado interno de link, sem valor semântico pro
// embedding.
func stripAnchorLines(lines []string) []string {
	var result []string
	for _, l := range lines {
		if !anchorRE.MatchString(strings.TrimSpace(l)) {
			result = append(result, l)
		}
	}
	return result
}

// cleanRSTMarkup resolve roles do Sphinx pra texto plano, ficando mais
// natural pro embedding: :rfc:`4271` -> RFC 4271, :clicmd:`x` -> `x`,
// :abbr:`BGP` -> BGP.
func cleanRSTMarkup(text string) string {
	return roleRE.ReplaceAllStringFunc(text, func(match string) string {
		sub := roleRE.FindStringSubmatch(match)
		role, val := sub[1], sub[2]
		switch role {
		case "rfc":
			return "RFC " + val
		case "clicmd":
			return "`" + val + "`"
		default:
			return val // abbr, ref, etc. -> só o texto
		}
	})
}

// buildChunks monta os chunks de cada seção: um chunk "command_reference"
// atômico por `.. clicmd::`, nunca dividido, e a prosa restante agrupada
// em chunks "concept" até um teto de tokens.
func buildChunks(lines []string, sections []rag.Section, daemon, protocol, sourceURL string, maxProseTokens int) []rag.Chunk {
	chunks := []rag.Chunk{} // slice não-nil, pra serializar como "[]" e não "null" quando vazio
	order := 0

	for _, sec := range sections {
		start, end := sec.StartLine, sec.EndLine
		if start < 0 {
			start = 0
		}
		if start > len(lines) {
			start = len(lines)
		}
		if end > len(lines) {
			end = len(lines)
		}
		if end < start {
			end = start
		}

		sectionLines := stripAnchorLines(lines[start:end])
		clicmds, proseLines := extractClicmdsAndProse(sectionLines, 0, len(sectionLines))
		fullSectionText := cleanRSTMarkup(strings.TrimSpace(strings.Join(sectionLines, "\n")))
		sectionPathStr := strings.Join(sec.Path, " > ")

		// --- command_reference: um chunk atômico por comando, nunca dividido ---
		for _, entry := range clicmds {
			bodyText := strings.TrimSpace(dedent(strings.Join(entry.Body, "\n")))
			bodyText = cleanRSTMarkup(bodyText)
			var content string
			if bodyText != "" {
				content = entry.Command + "\n\n" + bodyText
			} else {
				content = entry.Command
			}
			cmd := entry.Command
			chunks = append(chunks, rag.Chunk{
				ChunkID:       fmt.Sprintf("%s:cmd:%s", protocol, slugify(entry.Command, 60)),
				ChunkType:     "command_reference",
				Daemon:        daemon,
				Protocol:      protocol,
				SectionPath:   sectionPathStr,
				Command:       &cmd,
				Content:       content,
				ParentContent: fullSectionText,
				SourceURL:     sourceURL,
				TokenCount:    countTokens(content),
				Order:         order,
			})
			order++
		}

		// --- concept: prosa restante, agrupada em blocos até um teto de tokens ---
		blocks := splitProseIntoBlocks(proseLines)
		var groups []string
		var current []string
		currentTokens := 0
		for _, b := range blocks {
			bt := countTokens(b)
			if len(current) > 0 && currentTokens+bt > maxProseTokens {
				groups = append(groups, strings.Join(current, "\n\n"))
				current = nil
				currentTokens = 0
			}
			current = append(current, b)
			currentTokens += bt
		}
		if len(current) > 0 {
			groups = append(groups, strings.Join(current, "\n\n"))
		}

		for gi, groupText := range groups {
			groupText = cleanRSTMarkup(groupText)
			chunks = append(chunks, rag.Chunk{
				ChunkID:       fmt.Sprintf("%s:concept:%s:%04d", protocol, slugify(sectionPathStr, 60), gi),
				ChunkType:     "concept",
				Daemon:        daemon,
				Protocol:      protocol,
				SectionPath:   sectionPathStr,
				Command:       nil,
				Content:       groupText,
				ParentContent: fullSectionText,
				SourceURL:     sourceURL,
				TokenCount:    countTokens(groupText),
				Order:         order,
			})
			order++
		}
	}

	return chunks
}

// chunkFile lê um arquivo .rst e devolve seus chunks.
func chunkFile(path, daemon, protocol, sourceURL string) ([]rag.Chunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// normaliza quebras de linha, equivalente ao modo texto universal do Python
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")

	sections := parseSections(lines)
	return buildChunks(lines, sections, daemon, protocol, sourceURL, 220), nil
}
