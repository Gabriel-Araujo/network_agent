package turn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/paths"
)

// mermaidAccepts is the ground truth of the calibration corpus: the literal
// verdict of the real parser (mermaid 11.17.2) on the one ```mermaid block of
// each of the 15 sessions of assets/16-lote/sessions/, as the versioned probe
// output assets/16-lote/sessions/probe-lote.txt recorded it.
//
// Three blocks render; twelve do not. The corpus is a fixture, not a sample:
// it never changes, so a scanner change that starts accusing a rendering block
// fails here with no model and no network.
var mermaidAccepts = map[string]bool{
	"sessioni01.txt": false, // comment with a single %
	"sessioni02.txt": false, // parens in a bracketed subgraph title, and a single %
	"sessioni03.txt": false, // parens in a node label
	"sessioni04.txt": false, // parens in a bare subgraph title
	"sessioni05.txt": false, // parens in a node label
	"sessioni06.txt": false, // parens in a bracketed subgraph title
	"sessioni07.txt": false, // parens in a node label
	"sessioni08.txt": true,
	"sessioni09.txt": false, // note inside a flowchart — outside the rules
	"sessioni10.txt": false, // trailing comment with a single % — outside the rules
	"sessiont11.txt": true,
	"sessiont12.txt": false, // state id with a hyphen — outside the rules
	"sessiont13.txt": false, // parens in a bracketed subgraph title
	"sessiont14.txt": true,
	"sessiont15.txt": false, // state id with a hyphen — outside the rules
}

// TestScanMermaidCalibrationHasNoFalsePositive is the acceptance criterion of
// the scanner: over the whole calibration corpus it never accuses a block the
// real parser accepts. A false positive is the only error that makes a good
// diagram worse — it spends a re-ask breaking a diagram that rendered — so it
// is the one the corpus is allowed to fail on.
//
// Recall is reported, not asserted: four of the corpus's twelve failures (a
// note in a flowchart, two hyphenated state ids, one trailing comment) are
// deliberately outside the rules.
func TestScanMermaidCalibrationHasNoFalsePositive(t *testing.T) {
	caught, missed := 0, 0
	for name, accepts := range mermaidAccepts {
		blocks := scanMermaid(sessionText(t, name))
		if len(blocks) != 1 {
			t.Fatalf("%s: found %d blocks, want exactly 1", name, len(blocks))
		}
		block := blocks[0]

		switch {
		case accepts && block.Violation != "":
			t.Errorf("FALSE POSITIVE — %s renders in mermaid 11.17.2 but the scanner accused it:\n  line %d: %s\n  %s",
				name, block.Line, block.Violation, block.Snippet)
		case !accepts && block.Violation != "":
			caught++
		case !accepts:
			missed++
		}
	}
	t.Logf("corpus of %d blocks: %d of the 12 failures caught, %d outside the rules by design",
		len(mermaidAccepts), caught, missed)

	// Mutation control: the corpus only has teeth if the scanner actually
	// accuses the modes it claims. A scanner that returns "" for everything
	// has zero false positives too.
	if caught == 0 {
		t.Error("the scanner accused nothing in the whole corpus — zero false positives is vacuous")
	}
}

// TestScanMermaidPinsTheVerdictLine: the accusation points at the line the
// real parser failed on, because that line is what the re-ask shows the model.
// The wanted numbers are the ones in probe-lote.txt's parse errors.
func TestScanMermaidPinsTheVerdictLine(t *testing.T) {
	// Line numbers count the whole transcript, and the probe's count the
	// block body, so these are the body lines the probe named, resolved
	// against each session's own fence position.
	cases := map[string]string{
		"sessioni02.txt": "subgraph AS65010 [AS 65010 (Router RA)]",
		"sessioni03.txt": "C2[Router C2 (ABR)]",
		"sessioni04.txt": "subgraph Site A (AS 64512)",
		"sessioni05.txt": "CE1[Router CE1 (AS 65100)]",
		"sessioni06.txt": "subgraph Area0 [Area 0 (Backbone)]",
		"sessioni07.txt": "B --> B2[BGP Daemon Config (bgpd)];",
		"sessiont13.txt": "subgraph AS_A [Autonomous System A (e.g., AS 64512)]",
	}
	for name, wantSnippet := range cases {
		text := sessionText(t, name)
		block := scanMermaid(text)[0]
		if block.Snippet != wantSnippet {
			t.Errorf("%s: accused snippet %q, want %q", name, block.Snippet, wantSnippet)
			continue
		}
		// The line number has to resolve to that same line of the reply.
		got := strings.Split(text, "\n")
		if block.Line < 1 || block.Line > len(got) {
			t.Errorf("%s: line %d is outside the reply's %d lines", name, block.Line, len(got))
			continue
		}
		if strings.TrimSpace(strings.TrimSuffix(got[block.Line-1], "\r")) != wantSnippet {
			t.Errorf("%s: line %d is %q, want the accused line %q",
				name, block.Line, got[block.Line-1], wantSnippet)
		}
	}
}

func TestLinkStyleHexSemicolon(t *testing.T) {
	cases := []struct {
		line string
		bad  bool
	}{
		{"linkStyle 0 stroke:#0A3;", true},
		{"linkStyle 0 stroke:#00AA33;", true},
		{"linkStyle 0 stroke:#0A3", false},
		{"linkStyle 0 stroke:#0A3,stroke-width:2px;", false},
		{"linkStyle 0 stroke-width:2px;", false},
		{"linkStyle 0 stroke:red;", false},
		{"style A stroke:#333;", false},
		{"classDef c stroke:#333;", false},
		{"linkStyle 0 stroke:#0A3 ;", false},
		// Not rule 6 — the `;` is not at the end of the line — but rule 7
		// takes it, and the pele agrees: `linkStyle 0 stroke:red; %% nota`
		// is FALHA while the same line without the `;` renders, which is
		// the measured asymmetry of ticket 33.
		{"linkStyle 0 stroke:#0A3; %% comment", true},
		{"linkStyle 0 stroke:#0A3 %% comment", false},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			got := scanLine(tc.line, true)
			if (got != "") != tc.bad {
				t.Errorf("scanLine(%q) = %q, want violation = %v", tc.line, got, tc.bad)
			}
		})
	}
}

func TestLinkStyleProductionFixture(t *testing.T) {
	raw, err := os.ReadFile("../../.agent/ui/assets/24-linkstyle/bloco-real.mmd")
	if err != nil {
		t.Fatal(err)
	}
	blocks := scanMermaid("```mermaid\n" + string(raw) + "\n```")
	if len(blocks) != 1 || blocks[0].Violation == "" || blocks[0].Snippet != "linkStyle 1 stroke-width:2px,stroke:#0A3;" {
		t.Fatalf("production fixture verdict = %+v", blocks)
	}
	notice := mermaidNotice(blocks)
	if !strings.Contains(notice, "semicolon") || !strings.Contains(notice, "linkStyle") {
		t.Errorf("notice lacks specific fix: %s", notice)
	}
}

func TestScannerAgainstPeleCorpus(t *testing.T) {
	root := "../../.agent/ui/assets/16-divergencia"
	oracle, err := os.ReadFile(filepath.Join(root, "pele-all.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := os.ReadFile("../../.agent/ui/assets/27-rotulo-de-traco/probe-corpus.txt")
	if err != nil {
		t.Fatal(err)
	}
	previous := make(map[string]bool)
	for line := range strings.SplitSeq(string(baseline), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 6 && len(fields[0]) == 3 && (fields[5] == "ACUSA" || fields[5] == "LIMPO") {
			previous[fields[0]] = fields[5] == "ACUSA"
		}
	}
	if len(previous) != 44 {
		t.Fatalf("expected 44 baseline verdicts, got %d", len(previous))
	}
	var caught, missed, clean, falsePositives int
	for line := range strings.SplitSeq(strings.TrimSpace(string(oracle)), "\n") {
		id, verdict, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || (verdict != "FALHA" && verdict != "RENDER") {
			t.Fatalf("invalid oracle row %q", line)
		}
		raw, err := os.ReadFile(filepath.Join(root, "blocos", fmt.Sprintf("%s.mmd", id)))
		if err != nil {
			t.Fatal(err)
		}
		accused := faulted(scanMermaid("```mermaid\n" + string(raw) + "\n```"))
		before, found := previous[id]
		if !found || (before && !accused) {
			t.Errorf("%s: scanner lost a previously detected failure", id)
		}
		switch {
		case verdict == "FALHA" && accused:
			caught++
		case verdict == "FALHA":
			missed++
		case accused:
			falsePositives++
			t.Errorf("%s renders in pele but was accused", id)
		default:
			clean++
		}
	}
	if caught+missed+clean+falsePositives != 44 {
		t.Fatal("expected 44 oracle verdicts")
	}
	t.Logf("pele failures: %d caught, %d missed; renders: %d clean, %d false positives", caught, missed, clean, falsePositives)
}

// TestScanLineRules is the mutation control of the line rules: each case
// below kills one obvious mutant of scanLine, named in the case. They are the
// distinctions the corpus depends on, pinned small enough to read.
func TestScanLineRules(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		// Rule 1 fires.
		{"parens inside a bracket label", `    C2[Router C2 (ABR)]`, violationSpecialInLabel},
		{"parens in a bare subgraph title", `  subgraph AS65001 (Router R1)`, violationSpecialInLabel},
		{"braces in a bracketed subgraph title", `  subgraph A [AS {65001}]`, violationSpecialInLabel},
		{"parens inside a round label", `  RA(Router RA (RR))`, violationSpecialInLabel},

		// Rule 1 must not fire — the mutants these kill.
		// Mutant: scan comment lines too. 9 of 14 corpus comments have parens.
		{"parens in a %% comment", `    %% Connections (e.g., via the ABR)`, ""},
		// Mutant: ignore double quotes.
		{"quoted parens in a subgraph title", `  subgraph AS65001 ["AS 65001 (R1)"]`, ""},
		{"quoted parens in a node label", `    RM["Route-Map (policy)"]`, ""},
		// Mutant: flag any paren anywhere. Sequence text takes no quotes.
		{"parens in sequence message text", `    R1->>R2: BGP OPEN (Negotiation of parameters)`, ""},
		{"parens in a sequence participant", `    participant R1 as Router 1 (Local)`, ""},
		{"parens in a Note over", `    Note over R1,R2: Step 1: TCP handshake (3-way)`, ""},
		// Mutant: widen `special` beyond the measured set.
		{"br tag and digits in a label", `    A[router bgp <br>AS 65001] --> B{neighbors}`, ""},
		{"slash and colon in a label", `    ActionA[Action A: Set / Modify Routes]`, ""},
		{"style line", `    style C2 fill:#f9f,stroke:#333,stroke-width:2px`, ""},

		// Rule 1 over the fourth delimiter: the edge label's pipe. The
		// four forms are measured in the pele 1.13.0 — unquoted ( ) { }
		// inside |...| fail, the quoted form and plain text render
		// (assets/16-divergencia/README.md). Mutant: track only [ ( {.
		{"parens inside a pipe label", `    A --> |Local Protection Activation (LFA)| R_ALT`, violationSpecialInLabel},
		{"quoted parens inside a pipe label", `    A --> |"Local Protection (LFA)"| R_ALT`, ""},
		{"percentage inside a pipe label", `    A --> |50% of load| B`, ""},
		{"plain text inside a pipe label", `    A --> |Failure Detected| B`, ""},

		// Rule 1 over the bracket, the fifth and sixth characters of the
		// set. The seven forms are measured in the pele 1.13.0 (ticket 26,
		// .agent/ui/assets/26-colchete/probe-colchete.txt): the bracket
		// inside any delimiter fails and the quoted form renders, so the
		// fix is the quotes the notice already teaches. Mutant: keep
		// special() to ( ) { }.
		{"bracket inside a bracket label", `    A[list [0]] --> B`, violationSpecialInLabel},
		{"bracket inside a round label", `    A(list [0]) --> B`, violationSpecialInLabel},
		{"bracket inside a brace label", `    A{list [0]} --> B`, violationSpecialInLabel},
		{"bracket inside a pipe label", `    A --> |list [0]| B`, violationSpecialInLabel},
		{"closing bracket alone inside a pipe label", `    A --> |list 0]| B`, violationSpecialInLabel},
		{"opening bracket alone inside a pipe label", `    A --> |list [0| B`, violationSpecialInLabel},
		{"quoted bracket inside a pipe label", `    A --> |"list [0]"| B`, ""},

		// Mermaid's doubled-delimiter node shapes, all six measured RENDER
		// in the same probe: the doubling is the shape, not label text.
		// Four of them (every one but [[ ]] and ([ ])) were accused before
		// the bracket entered the set — the guard pays an old debt too.
		// Mutant: drop the shape guard from scanLine.
		{"subroutine shape", `    A[[Subroutine]] --> B`, ""},
		{"database shape", `    A[(Database)] --> B`, ""},
		{"stadium shape", `    A([Stadium]) --> B`, ""},
		{"circle shape", `    A((Circle)) --> B`, ""},
		{"double circle shape", `    A(((Double))) --> B`, ""},
		{"hexagon shape", `    A{{Hexagon}} --> B`, ""},
		// The guard is adjacency to the delimiter, not a licence: a real
		// defect inside a shape is still accused, and the edge label's
		// pipe has no shape vocabulary at all. Both measured FALHA.
		// Mutant: skip every special char in a shaped delimiter.
		{"parens inside a stadium shape", `    A([Session (Down)]) --> B`, violationSpecialInLabel},
		{"bracket inside a pipe label is never a shape", `    A --> |[0]| B`, violationSpecialInLabel},

		// Mutant: accuse the bracket on a subgraph line too. There it is
		// the title's own delimiter and not text — 16 of the 44 blocks of
		// assets/16-divergencia/blocos carry a bracketed title.
		{"bracketed subgraph title", `    subgraph S [AS 65001]`, ""},

		// Rule 4, ticket 27's class A: a `-- text` link label that no
		// arrow ever closed. Every line below was measured in the pele
		// 1.13.0 (.agent/ui/assets/27-rotulo-de-traco/probe-oraculo.txt
		// and .agent/ui/assets/33-alargamento/probe-oraculo.txt), and the
		// second one is the line that breaks block 042 of the corpus. The
		// fix is not quotes — this is the first mode whose notice cannot
		// teach quoting, because the quoted form fails too.
		// Mutant: drop rule 4.
		{"dash label plus pipe label", `    A -- CRITICAL LINK |Failure Point| B`, violationDashLabelNoArrow},
		{"the line of block 042", `    B -- CRITICAL LINK |Failure Point| C`, violationDashLabelNoArrow},
		{"thick dash label plus pipe label", `    A == LINK |Point| B`, violationDashLabelNoArrow},
		{"dotted dash label plus pipe label", `    A -. LINK |Point| B`, violationDashLabelNoArrow},
		{"quoting does not fix the dash label", `    A -- CRITICAL LINK |"Failure Point"| B`, violationDashLabelNoArrow},
		{"dash label with a bracketed target", `    A -- t |l| B[x]`, violationDashLabelNoArrow},
		// Mutant: let any closing link close any text label. Measured:
		// `-- ... === B` fails and `== ... === B` renders, so the kind of
		// the closing link has to match the kind of the opening one.
		{"closing link of the wrong kind", `    A -- CRITICAL LINK |Failure Point| === B`, violationDashLabelNoArrow},
		// Mutant: treat a run of two as a closing link. `A --- B` renders
		// and `A -- B` does not — only three dashes or a link end close.
		{"two dashes close nothing", `    A -- t |l| -- B`, violationDashLabelNoArrow},

		// Rule 4 without the pipe (ticket 33, candidates 1 and 4): the
		// `|label|` stopped being part of the rule, because the type gate
		// now tells `A -- CRITICAL LINK B` from classDiagram's bare
		// association. All five measured FALHA in the pele 1.13.0.
		// Mutant: ask rule 4 for a closed `|label|` again.
		{"dash label with no pipe", `    A -- CRITICAL LINK B`, violationDashLabelNoArrow},
		{"thick dash label with no pipe", `    A == LINK B`, violationDashLabelNoArrow},
		{"dotted dash label with no pipe", `    A -. LINK B`, violationDashLabelNoArrow},
		{"a one-word dash label", `    A -- B`, violationDashLabelNoArrow},
		{"dash label with no pipe and a bracketed target", `    A -- CRITICAL LINK B[x]`, violationDashLabelNoArrow},

		// The circle and the cross close a link exactly like the arrow,
		// measured in the pele 1.13.0 (probe-oraculo-4.txt). Every line
		// below RENDERS, and the first five were **accused** before
		// ticket 33: rule 4's five pre-existing false positives, which is
		// the measured reason linkEnd exists.
		// Mutant: drop `o` and `x` from linkEnd.
		{"dash label closed by a circle", `    A -- t |l| --o B`, ""},
		{"dash label closed by a cross", `    A -- t |l| --x B`, ""},
		{"thick label closed by a thick circle", `    A == t |l| ==o B`, ""},
		{"circle arrowhead with a pipe label", `    A --o |l| B`, ""},
		{"cross arrowhead with a pipe label", `    A --x |l| B`, ""},
		{"bare circle arrowhead", `    A --o B`, ""},
		{"bare cross arrowhead", `    A --x B`, ""},
		{"circle on both ends", `    A o--o B`, ""},
		{"cross on both ends", `    A x--x B`, ""},
		{"dash label closed by a circle, no pipe", `    A -- texto --o B`, ""},
		{"invisible link", `    A ~~~ B`, ""},
		// The kind still has to match, with a circle as with an arrow.
		// Mutant: let a link end of any kind close any text label.
		{"circle of the wrong kind", `    A -- texto ==o B`, violationDashLabelNoArrow},
		{"circle of the wrong kind after a pipe", `    A -- t |l| ==o B`, violationDashLabelNoArrow},
		// Measured miss: adjacency is the only thing telling `--o` from a
		// text label that starts with an o, and `A --open the valve B` is
		// FALHA. A miss, never a false positive.
		{"dash label whose text starts with o (uncovered)", `    A --open the valve B`, ""},

		// Rule 4 must not fire. The valid form differs from the invalid
		// one only by the arrow **after** the pipe, which is the risk the
		// ticket named: accusing any of these would be the one error the
		// scanner's calibration has never made.
		{"the arrow after the pipe is the fix", `    A -- CRITICAL LINK |Failure Point| --> B`, ""},
		{"dash label closed by an arrow", `    A -- CRITICAL LINK --> B`, ""},
		{"dash label closed by three dashes", `    A -- t |l| --- B`, ""},
		{"thick label closed by three equals", `    A == t |l| === B`, ""},
		{"thick label closed by a thick arrow", `    A == LINK |Point| ==> B`, ""},
		{"dotted label closed by a dotted arrow", `    A -. t |l| .-> B`, ""},
		{"dotted label closed by a dotted link", `    A -. t |l| -.- B`, ""},
		{"three dashes open no text label", `    A --- |Router bgp 1| P20;`, ""},
		{"an arrow inside a pipe label is text", `    A --> |a --> b| B`, ""},
		{"two edges on one line", `    A --> |l| B -- texto --> C`, ""},
		{"quoted dash label closed by an arrow", `    P20[Peer R-AS 20] -- "eBGP" --> A;`, ""},
		{"percent dash label closed by an arrow", `    A -- 50% --> B`, ""},
		// Measured miss, written down as a miss: a run of three with text
		// cannot be told from `A --- B` without counting words. The
		// pipe-less form that used to sit here is covered as of ticket 33.
		{"three dashes with text and a pipe (uncovered)", `    A --- t |l| B`, ""},

		// Rule 5, ticket 27's class B: the line ended inside an edge
		// label. The last case is the real emission of block 032 of the
		// corpus, where an unclosed quote swallows the closing pipe.
		// Mutant: drop rule 5.
		{"pipe label never closed", `    A --> |Failure Detected B`, violationUnclosedPipeLabel},
		{"pipe label never closed at end of line", `    A --> |Failure Detected`, violationUnclosedPipeLabel},
		{"three pipes on one line", `    A --> |a | b| B`, violationUnclosedPipeLabel},
		{"the line of block 032", `    R3 ---|"eBGP Session|-- R4;`, violationUnclosedPipeLabel},

		// Rule 5 must not fire.
		{"pipe label closed", `    A --> |Failure Detected| B`, ""},
		{"pipe label closed without spaces", `    A-->|Failure|B`, ""},
		// Measured misses. The empty `||` is FALHA in the pele and the
		// type gate makes it reachable, but ticket 33 **ruled it out**:
		// zero cases in the 89 blocks of the three corpora, and the
		// literal rule accuses `A -- texto || --> B`, which renders. A
		// pipe inside a bracket label never reaches rule 5.
		{"empty pipe label (uncovered)", `    A --> || B`, ""},
		{"empty pipe label inside a dash label (renders)", `    A -- texto || --> B`, ""},
		{"pipe inside a bracket label (uncovered)", `    A[texto | pipe] --> B`, ""},

		// Rule 2 fires: only the full-line comment with a single %.
		{"single % comment line", `    % eBGP peering link`, violationSinglePercent},
		{"double %% is a real comment", `    %% eBGP peering link`, ""},

		// Rule 2 must not fire. Every case below was measured against
		// mermaid 11.17.2 with 16-probe-mermaid-blocks.mjs, and the
		// first four PARSE — accusing them would be a false positive.
		// Mutant: treat any % as a comment marker.
		{"percent in an unquoted link label", `    A -- 50% --> B`, ""},
		{"percent after a pipe", `    A -->|% of load| B`, ""},
		{"percent in a sequence message", `    A->>B: 50% done`, ""},
		{"percentage inside a label", `    A[50% utilization]`, ""},

		// Rule 7, ticket 33: the comment after a statement. A trailing
		// comment breaks a flowchart and parses in a sequenceDiagram, so
		// it took the type gate to become reachable at all; the `%` and
		// the `%%` are one rule because the pele treats them alike by
		// statement (probe-oraculo-6.txt). The first line is the measured
		// defect of block 010 of the corpus and of sessioni10.
		// Mutant: drop rule 7.
		{"single % trailing a statement", `    B1 --> C3; % link them`, violationTailComment},
		{"double %% trailing a statement", `    B1 --> C3; %% link them`, violationTailComment},
		{"the line of block 010", `    B1 -- Protocol Exchange --> C3; % Link interfaces`, violationTailComment},
		{"comment after an edge with no semicolon", `    A --> B %% nota`, violationTailComment},
		{"comment after a node", `    A[x] %% nota`, violationTailComment},
		{"comment after end", `    end %% nota`, violationTailComment},
		// The four statements that swallow a trailing comment as a value
		// or a title, all measured RENDER with a valid prelude.
		// Mutant: drop the swallows guard.
		{"comment after style", `    style A fill:#f9f %% nota`, ""},
		{"comment after style past a semicolon", `    style A fill:#f9f; %% nota`, ""},
		{"comment after classDef", `    classDef c fill:#f9f %% nota`, ""},
		{"comment after a subgraph title", `    subgraph S %% nota`, ""},
		{"comment after linkStyle with no semicolon", `    linkStyle 0 stroke:red %% nota`, ""},
		// linkStyle is the one that swallows only up to the `;`, which is
		// what block 024 of the corpus is made of.
		// Mutant: except every linkStyle, or no linkStyle.
		{"comment after linkStyle past a semicolon", `    linkStyle 0 stroke:red; %% nota`, violationTailComment},
		{"the line of block 024", `    linkStyle 2 stroke:#0a0,stroke-width:2px; %% Area Border Link`, violationTailComment},
		// Rule 7 must not fire on a % that is text. The first two are
		// inside an open `-- text` label, which is the state rule 4
		// already tracks; the rest never start a token.
		// Mutant: drop the openKind or the tokenStart guard.
		{"a lone % inside a dash label", `    A -- % --> B`, ""},
		{"a spaced % inside a dash label", `    A -- 50 % --> B`, ""},
		{"a percentage in CSS", `    style A fill:#f9f,width:50%`, ""},
		{"a percentage glued to a node id", `    A50% --> B`, ""},
		// Not a miss: measured with its `end`, a subgraph title swallows
		// the % as well as the comment, so all three of these render
		// (.agent/ui/assets/33-alargamento/probe-oraculo-7.txt). The
		// parentheses of rule 1 still break the same title, which is the
		// asymmetry the subgraph guard has to keep.
		{"a percentage in a bare subgraph title", `    subgraph 50% de carga`, ""},
		{"a comment after a bare subgraph title", `    subgraph S % nota`, ""},
		{"parentheses still break a subgraph title", `    subgraph Site A (AS 64512)`, violationSpecialInLabel},
	}
	for _, c := range cases {
		if got := scanLine(c.line, true); got != c.want {
			t.Errorf("%s: scanLine(%q) = %q, want %q", c.name, c.line, got, c.want)
		}
	}
}

// TestScanLineOutsideFlowchart is the mutation control of the type gate
// (ticket 30): the lines below are another diagram type's vocabulary, and the
// gate — not an adjacency heuristic — is the only thing that keeps the
// flowchart rules off them. Each one was measured RENDER in the pele 1.13.0.
//
// The first ten are what the deleted relationPipe guard used to protect by
// guessing. Measured with the gate forced open, 10 of the 14 lines below are
// accused — the three classDiagram arrows, three of the five erDiagram
// cardinalities (`||--|{`, `}|--||`, `}o--o|`) and the four gated-rule cases:
// that is the mutant they kill.
func TestScanLineOutsideFlowchart(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{"classDiagram inheritance", `    Dog --|> Animal`, ""},
		{"classDiagram realization", `    Dog ..|> Animal`, ""},
		{"classDiagram dependency", `    Animal <|.. Dog`, ""},
		{"erDiagram zero or more", `    CUSTOMER ||--o{ ORDER : places`, ""},
		{"erDiagram one or more", `    ORDER ||--|{ LINE_ITEM : contains`, ""},
		{"erDiagram one to exactly one", `    LINE_ITEM }|--|| PRODUCT : is`, ""},
		{"erDiagram zero or one both ends", `    A |o--o| B : rel`, ""},
		{"erDiagram many to zero or one", `    A }o--o| B : rel`, ""},
		{"classDiagram association", `    Animal -- Dog`, ""},
		{"classDiagram association with cardinality", `    Animal "1" -- "many" Dog`, ""},

		// Rules 1, 4, 5 and 6 are gated: the same lines the flowchart
		// table above accuses pass clean here. Mutant: gate nothing, or
		// gate on the wrong side.
		{"rule 1 is gated", `    C2[Router C2 (ABR)]`, ""},
		{"rule 4 is gated", `    A -- CRITICAL LINK |Failure Point| B`, ""},
		{"rule 4 without a pipe is gated", `    A -- CRITICAL LINK B`, ""},
		{"rule 5 is gated", `    A --> |Failure Detected B`, ""},
		{"rule 6 is gated", `linkStyle 1 stroke-width:2px,stroke:#0A3;`, ""},
		// Rule 7 is gated for a measured reason, not by symmetry: the same
		// trailing `%%` renders in a sequenceDiagram and in a
		// stateDiagram-v2 (probe-oraculo.txt).
		{"rule 7 is gated", `    A --> B %% nota`, ""},
		{"rule 7 is gated for a single %", `    A --> B % nota`, ""},

		// Rule 2 is not: the full-line comment with a single % is the one
		// line rule true of every diagram type, and gating it would make
		// sessioni01 of the calibration corpus — a `graph` — depend on the
		// gate for a catch that has nothing to do with it.
		// Mutant: gate rule 2 too.
		{"rule 2 still fires", `    % eBGP peering link`, violationSinglePercent},
		{"double %% is still a comment", `    %% eBGP peering link`, ""},
	}
	for _, c := range cases {
		if got := scanLine(c.line, false); got != c.want {
			t.Errorf("%s: scanLine(%q, false) = %q, want %q", c.name, c.line, got, c.want)
		}
	}
}

// TestVerdictGatesOnDiagramType pins the gate's input: the declaration on the
// block's first non-empty line, read as an allowlist of two keywords. The same
// body is accused under a flowchart and clean under every other header, which
// is the whole decision of ticket 30 in one table.
func TestVerdictGatesOnDiagramType(t *testing.T) {
	// A line rule 1 accuses, and nothing else.
	const defect = "    C2[Router C2 (ABR)]\n"

	for _, c := range []struct {
		name    string
		header  string
		accused bool
	}{
		{"flowchart", "flowchart LR\n", true},
		{"flowchart with no direction", "flowchart\n", true},
		{"graph is the old spelling of the same type", "graph TD\n", true},
		{"blank lines before the declaration", "\n\nflowchart LR\n", true},
		{"erDiagram", "erDiagram\n", false},
		{"classDiagram", "classDiagram\n", false},
		{"sequenceDiagram", "sequenceDiagram\n", false},
		{"stateDiagram-v2", "stateDiagram-v2\n", false},
		{"an unknown type runs only the text rules", "gantt\n", false},
		{"a type newer than this code", "radarChart\n", false},
		{"no declaration at all", defect, false},
	} {
		blocks := scanMermaid("```mermaid\n" + c.header + defect + "```\n")
		if len(blocks) != 1 {
			t.Fatalf("%s: found %d blocks, want 1", c.name, len(blocks))
		}
		if accused := blocks[0].Violation != ""; accused != c.accused {
			t.Errorf("%s: accused = %v, want %v (%q)", c.name, accused, c.accused, blocks[0].Violation)
		}
	}

	// Rule 2 does not consult the type, and rule 3 is about the reply.
	blocks := scanMermaid("```mermaid\nerDiagram\n    % nota\n```\n")
	if blocks[0].Violation != violationSinglePercent {
		t.Errorf("rule 2 outside a flowchart = %q, want the single-%% violation", blocks[0].Violation)
	}
}

// TestScanMermaidUnclosedFence: rule 3. A fence opened and never closed is
// truncation, not disobedience, and it is the one violation that is about the
// reply rather than the diagram.
func TestScanMermaidUnclosedFence(t *testing.T) {
	blocks := scanMermaid("here you go:\n```mermaid\nflowchart LR\n  A --> B\n")
	if len(blocks) != 1 {
		t.Fatalf("found %d blocks, want 1", len(blocks))
	}
	if blocks[0].Violation != violationUnclosedFence {
		t.Errorf("violation %q, want %q", blocks[0].Violation, violationUnclosedFence)
	}
	if blocks[0].Line != 2 {
		t.Errorf("line %d, want 2 (the open fence)", blocks[0].Line)
	}
}

// TestScanMermaidZeroBlocks: a reply with no diagram breaks none of the rules.
// The rules run over each block, so there is nothing to accuse — zero
// blocks is a different question and not this scanner's (ticket 32).
func TestScanMermaidZeroBlocks(t *testing.T) {
	if blocks := scanMermaid("Here is the plan, in prose. No diagram at all."); blocks != nil {
		t.Errorf("scanned %d blocks in a reply with none", len(blocks))
	}
	if faulted(nil) {
		t.Error("no block counts as faulted")
	}
	if shippable(nil) {
		t.Error("a retry with no block must not ship")
	}
}

// TestScanMermaidManyBlocks: the index is the fence's position in the reply,
// and every block gets its own record even when only one is faulted.
func TestScanMermaidManyBlocks(t *testing.T) {
	blocks := scanMermaid("one:\n```mermaid\nflowchart LR\n  A[ok] --> B\n```\ntwo:\n```mermaid\nflowchart LR\n  C[bad (x)] --> D\n```\n")
	if len(blocks) != 2 {
		t.Fatalf("found %d blocks, want 2", len(blocks))
	}
	if blocks[0].Index != 1 || blocks[0].Violation != "" {
		t.Errorf("block 1 = %+v, want index 1 not faulted", blocks[0])
	}
	if blocks[1].Index != 2 || blocks[1].Violation != violationSpecialInLabel {
		t.Errorf("block 2 = %+v, want index 2 faulted on rule 1", blocks[1])
	}
	if !faulted(blocks) || shippable(blocks) {
		t.Error("one faulted block faults the reply and must not ship")
	}
	// The notice names the faulted block only, with line and snippet.
	notice := mermaidNotice(blocks)
	if strings.Contains(notice, "block 1") {
		t.Error("the notice points at a block that passed")
	}
	for _, want := range []string{"block 2", "line 9", "C[bad (x)] --> D", violationSpecialInLabel} {
		if !strings.Contains(notice, want) {
			t.Errorf("the notice is missing %q:\n%s", want, notice)
		}
	}
}

// sessionText reads one transcript of the calibration corpus.
func sessionText(t *testing.T, name string) string {
	t.Helper()
	path, err := paths.RepoFile(".agent/roteamento/assets/16-lote/sessions/" + name)
	if err != nil {
		t.Fatalf("resolving %s: %v", name, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the calibration corpus %s: %v", name, err)
	}
	return string(data)
}
