package turn

import (
	"fmt"
	"strings"
)

// MermaidBlock is one ```mermaid fence of a reply, with the verdict of the
// turn's narrow scanner on it.
//
// Violation empty means **not faulted**, never "valid": the scanner is three
// rules over the measured failure modes (6 of the 7 the mounted batch showed),
// not a Mermaid parser, so a block it does not accuse may still fail to render.
// The distinction is in CONTEXT.md, under Bloco de diagrama
// (decision 8: .agent/roteamento/issues/23-validar-os-blocos-mermaid-no-turno.md).
type MermaidBlock struct {
	// Index is the 1-based position of the fence in the reply.
	Index int

	// Violation names the rule the block broke ("" = not faulted by these
	// three rules).
	Violation string

	// Line is the 1-based line of the violation, counted in the reply, and
	// Snippet is that line trimmed. Both are zero/empty when not faulted.
	// They are what the re-ask points at.
	Line    int
	Snippet string
}

// the three rules' violation texts, constant and model-agnostic like the file
// tools' notices (internal/tools/file/prompts.go).
const (
	violationSpecialInLabel = "unquoted ( ) { } [ ] inside a label delimiter or a subgraph title — Mermaid needs double quotes around text with punctuation"
	violationSinglePercent  = "comment with a single % — Mermaid comments need %%"
	violationUnclosedFence  = "the ```mermaid fence was opened and never closed — the reply was truncated"
)

// scanMermaid finds every ```mermaid block of the text and applies the three
// rules to each, in the order they appear. The reply of any Subagente goes
// through the same scan: the check is of the text, not of who wrote it
// (decision 7 of ticket 23).
//
// No block returns nil — a reply with no diagram breaks none of the rules,
// which is deliberate: zero blocks is a different question, and it is not this
// scanner's (decision 3 of
// .agent/roteamento/issues/32-illustrator-responde-o-plano-sem-diagrama.md).
func scanMermaid(text string) []MermaidBlock {
	var blocks []MermaidBlock
	var body []bodyLine
	open, openLine := false, 0

	n := 0
	for raw := range strings.SplitSeq(text, "\n") {
		n++
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(line)

		if !open {
			// The open fence may carry a prefix: in a CLI transcript it
			// reads `Agent: ```mermaid`.
			if strings.Contains(line, "```mermaid") {
				open, openLine, body = true, n, nil
			}
			continue
		}
		if trimmed == "```" {
			blocks = append(blocks, verdict(len(blocks)+1, body))
			open = false
			continue
		}
		body = append(body, bodyLine{n: n, text: line})
	}

	if open {
		blocks = append(blocks, MermaidBlock{
			Index:     len(blocks) + 1,
			Violation: violationUnclosedFence,
			Line:      openLine,
			Snippet:   "```mermaid",
		})
	}
	return blocks
}

// bodyLine is a line of a block's body with its line number in the whole reply,
// so a violation points at the reply the user would have seen.
type bodyLine struct {
	n    int
	text string
}

// verdict applies rules 1 and 2 to a block's body and returns the first
// violation found — one record per block, so the re-ask points at one line
// instead of a list the model has to triage.
func verdict(index int, body []bodyLine) MermaidBlock {
	for _, l := range body {
		if v := scanLine(l.text); v != "" {
			return MermaidBlock{Index: index, Violation: v, Line: l.n, Snippet: strings.TrimSpace(l.text)}
		}
	}
	return MermaidBlock{Index: index}
}

// scanLine applies rules 1 and 2 to one line of a block's body in a single
// pass, tracking double quotes and the label delimiter it is inside.
//
// Rule 2 is deliberately only the **full-line** comment, because that is the
// part that is true of every diagram type. Measured against mermaid 11.17.2:
// a trailing `%%` breaks a flowchart but is fine in a sequenceDiagram, and a
// bare % in text renders everywhere (`A -- 50% --> B`, `A -->|% of load| B`,
// `A->>B: 50% done` all parse). Accusing a mid-line % would therefore trade
// one more catch for a class of false positives, and telling the valid case
// from the invalid one needs each diagram type's vocabulary — the parser
// decision 1 refused. The trailing-comment modes are left uncovered and
// written down as such.
//
// A %% that is not at the start of the line stops the scan instead: a
// comment's text is not label text, and 9 of the 14 comment lines in the
// calibration corpus carry parentheses.
func scanLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Rule 2: a comment is only ever %% at the start of its own line.
	if after, found := strings.CutPrefix(trimmed, "%"); found {
		if strings.HasPrefix(after, "%") {
			return ""
		}
		return violationSinglePercent
	}

	// A subgraph title is the one place where punctuation outside any
	// bracket still breaks the diagram: `subgraph Site A (AS 64512)`.
	subgraph := strings.HasPrefix(trimmed, "subgraph ")

	var closer byte // 0 = outside a delimiter, else the char that closes it
	quoted := false

	// Mermaid's node shapes double the delimiter — [[ ]], [( )], ([ ]), (( )),
	// ((( ))) and {{ }}, all six measured RENDER in the pele 1.13.0 — so the
	// doubling characters are the shape, not label text. Rule 1 skips the run
	// right after the opener and, only in a delimiter that opened with such a
	// run, the mate right before the closer: without the second half
	// `A([Stadium])` is accused, and keying it on shape keeps
	// `A --> |list 0]| B` accused.
	run, shape := false, false

	for i := range len(line) {
		c := line[i]
		switch {
		case c == '"':
			quoted, run = !quoted, false
		case quoted:
			// Quoted text is exactly what Mermaid accepts.
		case closer != 0:
			switch {
			case c == closer:
				closer, run, shape = 0, false, false
			case !special(c):
				run = false
			case run:
				// Still in the opening run: `[(`, `((`, `(((`.
				shape = true
			case shape && i+1 < len(line) && line[i+1] == closer:
				// The shape's mate: the `)` of `[(Database)]`.
			default:
				return violationSpecialInLabel
			}
		case c == '%':
			// A trailing %% is a comment: its text is not label text,
			// so the scan of this line ends here. A single % mid-line
			// is ordinary text that renders.
			if i+1 < len(line) && line[i+1] == '%' {
				return ""
			}
		case subgraph && undelimited(c):
			return violationSpecialInLabel
		case c == '[':
			closer, run = ']', true
		case c == '(':
			closer, run = ')', true
		case c == '{':
			closer, run = '}', true
		case c == '|':
			// The edge label's pipe opens and closes with the same
			// character, unlike the other three, so an odd count leaves
			// the scan inside it to the end of the line. It gets no
			// shape run: the doubled delimiters are node shapes, and an
			// edge label has none — `A --> |[0]| B` is a defect.
			closer = '|'
		}
	}
	return ""
}

// special are the characters rule 1 accuses inside a delimiter: the parentheses
// (4 of the 7 measured modes), the braces (1) and the brackets (six forms
// measured in the pele 1.13.0, ticket 26). Kept to the measured set on
// purpose — `<br>` and `:` and `/` ride inside labels that render.
func special(c byte) bool {
	return undelimited(c) || c == '[' || c == ']'
}

// undelimited is the narrower set rule 1 accuses outside any delimiter, on a
// subgraph line. The bracket is not in it: there it is the title's own
// delimiter and not text, and `subgraph S [AS 65001]` renders — 16 of the 44
// blocks of the calibration corpus carry one.
func undelimited(c byte) bool {
	return c == '(' || c == ')' || c == '{' || c == '}'
}

// faulted reports whether any block broke a rule.
func faulted(blocks []MermaidBlock) bool {
	for _, b := range blocks {
		if b.Violation != "" {
			return true
		}
	}
	return false
}

// shippable is the re-ask's non-regression gate: the retry's reply ships only
// if it carries at least one block and **all** of them pass. No block, still
// broken, or degraded prose all keep the original reply (decision 5 of
// ticket 23).
func shippable(blocks []MermaidBlock) bool {
	return len(blocks) > 0 && !faulted(blocks)
}

// mermaidNotice is the re-ask's message: the violation of each faulted block
// with its line and the line itself. Pointed text, not a general rule — the
// rule was already in the prompt and ticket 28 measured that it is not the
// bottleneck.
func mermaidNotice(blocks []MermaidBlock) string {
	var b strings.Builder
	b.WriteString("Your reply has a diagram block that will not render. Fix it and reply again with the whole answer, keeping the same content:\n")
	for _, blk := range blocks {
		if blk.Violation == "" {
			continue
		}
		fmt.Fprintf(&b, "\n- block %d, line %d: %s\n  %s\n", blk.Index, blk.Line, blk.Violation, blk.Snippet)
	}
	return b.String()
}
