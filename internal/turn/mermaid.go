package turn

import (
	"fmt"
	"strings"
)

// MermaidBlock is one ```mermaid fence of a reply, with the verdict of the
// turn's narrow scanner on it.
//
// Violation empty means **not faulted**, never "valid": the scanner is seven
// rules over the measured failure modes (6 of the 7 the mounted batch showed,
// plus the two classes of ticket 27, linkStyle and the comment after a
// statement), not a Mermaid parser, so a block it does
// not accuse may still fail to render. The distinction is in CONTEXT.md, under
// Bloco de diagrama
// (decision 8: .agent/roteamento/issues/23-validar-os-blocos-mermaid-no-turno.md).
//
// Five of the seven are flowchart vocabulary and only run on a block that
// declares one, so outside a flowchart silence is of two rules and not of seven
// (.agent/ui/issues/30-o-scanner-conhece-o-tipo-do-diagrama.md).
type MermaidBlock struct {
	// Index is the 1-based position of the fence in the reply.
	Index int

	// Violation names the rule the block broke ("" = not faulted by these
	// seven rules).
	Violation string

	// Line is the 1-based line of the violation, counted in the reply, and
	// Snippet is that line trimmed. Both are zero/empty when not faulted.
	// They are what the re-ask points at.
	Line    int
	Snippet string
}

// the rules' violation texts, constant and model-agnostic like the file
// tools' notices (internal/tools/file/prompts.go).
//
// The last four carry no quoting advice on purpose: they are
// modes whose fix is not double quotes, measured in the pele 1.13.0 —
// `A -- CRITICAL LINK |"Failure Point"| B` fails quoted exactly as it fails
// bare (.agent/ui/assets/27-rotulo-de-traco/probe-oraculo.txt).
const (
	violationSpecialInLabel    = "unquoted ( ) { } [ ] inside a label delimiter or a subgraph title — Mermaid needs double quotes around text with punctuation"
	violationSinglePercent     = "comment with a single % — Mermaid comments need %%"
	violationUnclosedFence     = "the ```mermaid fence was opened and never closed — the reply was truncated"
	violationDashLabelNoArrow  = "an edge with a -- text label and no arrow closing the link — Mermaid needs the arrow after the text: A -- text --> B"
	violationUnclosedPipeLabel = "a |label| opened and never closed on its line — the edge label needs its second |"
	violationLinkStyleHex      = "linkStyle ends with a hexadecimal color followed by a semicolon — remove the semicolon or move the color before another property"
	violationTailComment       = "a comment after a statement on the same line — Mermaid only takes a %% comment at the start of its own line"
)

// scanMermaid finds every ```mermaid block of the text and applies the rules
// to each, in the order they appear. The reply of any Subagente goes
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

// verdict applies the per-line rules to a block's body and returns the first
// violation found — one record per block, so the re-ask points at one line
// instead of a list the model has to triage.
func verdict(index int, body []bodyLine) MermaidBlock {
	flow := flowchartBlock(body)
	for _, l := range body {
		if v := scanLine(l.text, flow); v != "" {
			return MermaidBlock{Index: index, Violation: v, Line: l.n, Snippet: strings.TrimSpace(l.text)}
		}
	}
	return MermaidBlock{Index: index}
}

// flowchartBlock reports whether the block declares a flowchart, which is what
// licenses the four rules that are flowchart vocabulary (ticket 30). The
// allowlist is deliberate: an unknown type — `gantt`, `mindmap`, a type newer
// than this code — runs only the two rules that hold for every diagram, so a
// vocabulary the rules never measured can no longer be accused by adjacency.
// It costs nothing measured: all 20 accusations over the three corpora of 89
// blocks (the 44 of the pele oracle, the 15 calibration sessions, 30 blocks of
// real history) are on a flowchart or a graph.
//
// The declaration is the first token of the block's first non-empty line, in
// 89 of 89 measured blocks: none opens with front matter or a %%{init}%%
// directive. A block that did would read as an unknown type and fall to the
// quiet side, which is a miss and never a false positive.
func flowchartBlock(body []bodyLine) bool {
	for _, l := range body {
		trimmed := strings.TrimSpace(l.text)
		if trimmed == "" {
			continue
		}
		for kind := range strings.FieldsSeq(trimmed) {
			return kind == "flowchart" || kind == "graph"
		}
	}
	return false
}

// scanLine applies the per-line rules to one line of a block's body in a
// single pass, tracking double quotes and the label delimiter it is inside.
//
// Rules 4 and 5 (ticket 27) ride on the same pass and fire at the end of the
// line, because both are about something the line never did: rule 4 is a
// `-- text` link label that no arrow ever closed, rule 5 a pipe label that
// opened and never closed. Both are new rules rather than a widening of rule 1,
// and both were measured in the pele 1.13.0 first —
// .agent/ui/assets/27-rotulo-de-traco/README.md.
//
// Rule 4 no longer asks for a `|label|` (ticket 33): the pipe used to be the
// rule's licence, because `A -- CRITICAL LINK B` could not be told from
// classDiagram's bare association `Animal -- Dog`, which renders. The type
// gate tells them apart, so the pipe is gone from the rule and `A -- t B`,
// `A == t B` and `A -. t B` — all measured FALHA — are accused too.
//
// Rule 7 is the comment after a statement, `%` or `%%` alike. It is this
// scanner's only rule where the **statement** decides, not the character:
// measured in the pele 1.13.0 with a valid prelude of four nodes and three
// edges (.agent/ui/assets/33-alargamento/probe-oraculo-6.txt), a trailing
// comment breaks an edge, a node and an `end`, and renders after `style`,
// `classDef`, `subgraph` and a `linkStyle` with no `;` — those four swallow it
// as part of a value or a title. It needs the type for the same reason rule 2
// does not have it: the same trailing `%%` renders in a sequenceDiagram and in
// a stateDiagram-v2.
//
// Rule 2 stays deliberately the **full-line** comment, because that is the part
// that is true of every diagram type, and it is one of the two rules that run
// outside a flowchart. A bare % in text renders everywhere (`A -- 50% --> B`,
// `A -->|% of load| B`, `A->>B: 50% done`, `style A width:50%`), which is why
// rule 7 fires only on a % that starts a token, outside every delimiter and
// outside an open `-- text` label: inside one, `A -- % --> B` renders.
//
// A %% that rule 7 does not accuse stops the scan instead: a comment's text is
// not label text, and 9 of the 14 comment lines in the calibration corpus carry
// parentheses.
func scanLine(line string, flow bool) string {
	trimmed := strings.TrimSpace(line)

	// Rule 2: a comment is only ever %% at the start of its own line.
	if after, found := strings.CutPrefix(trimmed, "%"); found {
		if strings.HasPrefix(after, "%") {
			return ""
		}
		return violationSinglePercent
	}

	// Everything below is flowchart vocabulary — the label delimiters, the
	// link and its text label, the edge's linkStyle — so outside a
	// flowchart it does not run (ticket 30). The line that used to guess
	// another type's relationship token from the characters around a pipe
	// (classDiagram's `--|>`, erDiagram's `||--|{`) is gone with it: the
	// type answers what adjacency was guessing.
	if !flow {
		return ""
	}

	if linkStyleHexSemicolon(trimmed) {
		return violationLinkStyleHex
	}

	// A subgraph title is the one place where punctuation outside any
	// bracket still breaks the diagram: `subgraph Site A (AS 64512)`.
	subgraph := strings.HasPrefix(trimmed, "subgraph ")

	// Rule 7's exceptions: the statements that swallow a trailing comment as
	// part of a value or a title, so one after them renders. `linkStyle` is
	// the odd one out — it swallows the comment only up to a `;`, which is
	// what breaks block 024 of the corpus (`linkStyle 2 stroke:#0a0,
	// stroke-width:2px; %% Area Border Link`, the block ticket 29 measured
	// its own rule would pass by) while `linkStyle 0 stroke:red %% nota`
	// renders. `style` and `classDef` swallow it past the `;` too.
	swallows := subgraph ||
		strings.HasPrefix(trimmed, "style ") ||
		strings.HasPrefix(trimmed, "classDef ")
	linkStyle := strings.HasPrefix(trimmed, "linkStyle ")

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

	// Rule 4's state: openKind is the kind of link whose text label is open
	// ('-' for `--` and `-.`, '=' for `==`, 0 for none). The kind matters:
	// only a closing link of the same kind closes the text, measured in the
	// pele — `A -- t |l| --- B` renders and `A -- t |l| === B` does not.
	//
	// Rule 7 reads it too: a % inside an open text label is text, not a
	// comment. And semi records a `;`, which is what exposes a comment on a
	// `linkStyle` line.
	var linkKind, openKind byte
	linkRun := 0
	semi := false

	for i := range len(line) {
		c := line[i]

		// Rule 4 counts the link's dashes only outside quotes and
		// delimiters: inside a label they are text, and
		// `A --> |a --> b| B` renders.
		if closer == 0 && !quoted && c != '"' {
			switch {
			case linkChar(c):
				if linkRun == 0 {
					linkKind = linkKindOf(c)
				}
				linkRun++
			case linkRun > 0:
				switch {
				case linkRun >= 3 || linkEnd(c):
					// A closing link: `---`, `-->`, `.->`,
					// `--o`, `--x`. It closes the text label
					// only if it is of the same kind.
					if linkKind == openKind {
						openKind = 0
					}
				case linkRun == 2:
					// `--`, `==` or `-.`: the link's text label
					// opens here, and from now on only a link
					// end can close it. A run of three is a
					// whole link (`A --- B` renders) and opens
					// nothing.
					openKind = linkKind
				}
				linkRun = 0
			}
		}

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
			// Rule 7: a comment after a statement, which the
			// statement only swallows in the four measured cases.
			if openKind == 0 && tokenStart(line, i) && !swallows && (!linkStyle || semi) {
				return violationTailComment
			}
			// A %% the rule let through is a comment: its text is not
			// label text, so the scan of this line ends here. A % that
			// does not start a token is ordinary text that renders.
			if i+1 < len(line) && line[i+1] == '%' {
				return ""
			}
		case c == ';':
			semi = true
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
			// the scan inside it to the end of the line — which is
			// rule 5, below. It gets no shape run: the doubled
			// delimiters are node shapes, and an edge label has none —
			// `A --> |[0]| B` is a defect.
			closer = '|'
		}
	}

	switch {
	case closer == '|':
		// Rule 5: the line ended inside an edge label.
		return violationUnclosedPipeLabel
	case openKind != 0:
		// Rule 4: a `-- text` label that no link end ever closed.
		return violationDashLabelNoArrow
	}
	return ""
}

// tokenStart reports whether the byte at i opens a token, which is what tells
// rule 7's comment from a percentage: `A50% --> B` and `style A width:50%`
// render, `A --> B % nota` does not. The `;` counts as a boundary because that
// is where the statement ends — the measured form of block 010 of the corpus
// is `C3; % Link interfaces`.
func tokenStart(line string, i int) bool {
	if i == 0 {
		return true
	}
	c := line[i-1]
	return c == ' ' || c == '\t' || c == ';'
}

// linkEnd are the characters that end a Mermaid link next to its dashes: the
// arrowhead, the circle and the cross. The circle and the cross were measured
// in the pele 1.13.0 only in ticket 33, and measuring them closed five false
// positives rule 4 had been making since ticket 27 — `A -- t |l| --o B`,
// `A -- t |l| --x B`, `A == t |l| ==o B`, `A --o |l| B` and `A --x |l| B` all
// render (.agent/ui/assets/33-alargamento/probe-oraculo-4.txt).
//
// Adjacency is the whole discriminator: `A --o B` renders and `A --open the
// valve B` does not, so a link end only counts right after the dashes. The
// second of those is a measured miss, written down as one.
func linkEnd(c byte) bool {
	return c == '>' || c == 'o' || c == 'x'
}

// linkStyleHexSemicolon recognizes the narrowly measured entity-encoding bug.
func linkStyleHexSemicolon(line string) bool {
	if !strings.HasPrefix(line, "linkStyle ") && !strings.HasPrefix(line, "linkStyle\t") {
		return false
	}
	if !strings.HasSuffix(line, ";") {
		return false
	}
	_, color, found := strings.CutLast(line, "#")
	if !found || len(color) <= 1 {
		return false
	}
	for i := 0; i < len(color)-1; i++ {
		c := color[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// linkChar are the characters Mermaid draws a link with: the dash, the equals
// of the thick link and the dot of the dotted one.
func linkChar(c byte) bool {
	return c == '-' || c == '=' || c == '.'
}

// linkKindOf folds the dotted link into the dash family, because the pele
// accepts `A -. t |l| .-> B` and `A -. t |l| -.- B` — the dot and the dash
// close each other. Only the thick link is a kind of its own: `A -- t ==> B`
// fails, and so does `A == t --> B`.
func linkKindOf(c byte) byte {
	if c == '=' {
		return '='
	}
	return '-'
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
