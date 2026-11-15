package turn

import (
	"os"
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
	"sessioni09.txt": false, // note inside a flowchart — outside the three rules
	"sessioni10.txt": false, // trailing comment with a single % — outside the three rules
	"sessiont11.txt": true,
	"sessiont12.txt": false, // state id with a hyphen — outside the three rules
	"sessiont13.txt": false, // parens in a bracketed subgraph title
	"sessiont14.txt": true,
	"sessiont15.txt": false, // state id with a hyphen — outside the three rules
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
	t.Logf("corpus of %d blocks: %d of the 12 failures caught, %d outside the three rules by design",
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

// TestScanLineRules is the mutation control of the three rules: each case
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

		// Rule 4, ticket 27's class A: a `-- text` link label that picked
		// up a `|label|` and no arrow ever closed. Every line below was
		// measured in the pele 1.13.0
		// (.agent/ui/assets/27-rotulo-de-traco/probe-oraculo.txt), and the
		// first one is the line that breaks block 042 of the corpus. The
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
		// and `A -- B` does not — only three dashes or an arrow close.
		{"two dashes close nothing", `    A -- t |l| -- B`, violationDashLabelNoArrow},

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
		// The pipe is what makes rule 4 safe: classDiagram's bare
		// association has the same `-- text` shape with no pipe anywhere,
		// and it renders.
		{"classDiagram association", `    Animal -- Dog`, ""},
		{"classDiagram association with cardinality", `    Animal "1" -- "many" Dog`, ""},
		// Measured misses, written down as misses: a run of three with
		// text cannot be told from `A --- B` without counting words, and
		// the pipe-less form cannot be told from the association above.
		{"three dashes with text and a pipe (uncovered)", `    A --- t |l| B`, ""},
		{"dash label with no pipe at all (uncovered)", `    A -- CRITICAL LINK B`, ""},

		// Rule 5, ticket 27's class B: the line ended inside an edge
		// label. The last case is the real emission of block 032 of the
		// corpus, where an unclosed quote swallows the closing pipe.
		// Mutant: drop rule 5.
		{"pipe label never closed", `    A --> |Failure Detected B`, violationUnclosedPipeLabel},
		{"pipe label never closed at end of line", `    A --> |Failure Detected`, violationUnclosedPipeLabel},
		{"three pipes on one line", `    A --> |a | b| B`, violationUnclosedPipeLabel},
		{"the line of block 032", `    R3 ---|"eBGP Session|-- R4;`, violationUnclosedPipeLabel},

		// Rule 5 must not fire. The pipe is vocabulary in two other
		// diagram types, and all of these render: without the
		// relationPipe guard rule 5 accuses every one of them, and rule 1
		// already accused the `{` of `||--|{` before this ticket.
		// Mutant: drop the guard.
		{"classDiagram inheritance", `    Dog --|> Animal`, ""},
		{"classDiagram realization", `    Dog ..|> Animal`, ""},
		{"classDiagram dependency", `    Animal <|.. Dog`, ""},
		{"erDiagram zero or more", `    CUSTOMER ||--o{ ORDER : places`, ""},
		{"erDiagram one or more", `    ORDER ||--|{ LINE_ITEM : contains`, ""},
		{"erDiagram one to exactly one", `    LINE_ITEM }|--|| PRODUCT : is`, ""},
		{"erDiagram zero or one both ends", `    A |o--o| B : rel`, ""},
		{"erDiagram many to zero or one", `    A }o--o| B : rel`, ""},
		{"pipe label closed", `    A --> |Failure Detected| B`, ""},
		{"pipe label closed without spaces", `    A-->|Failure|B`, ""},
		// Measured misses: the empty `||` is erDiagram's "exactly one"
		// with no local difference from an empty flowchart label, and a
		// pipe inside a bracket label never reaches rule 5.
		{"empty pipe label (uncovered)", `    A --> || B`, ""},
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

		// Measured misses, written down as misses. A trailing comment
		// breaks a flowchart and parses in a sequenceDiagram, so telling
		// them apart needs each type's vocabulary — out of scope by
		// decision 1, and silence here is a miss, never a promise.
		{"single % trailing a statement (uncovered)", `    B1 --> C3; % link them`, ""},
		{"double %% trailing a statement (uncovered)", `    B1 --> C3; %% link them`, ""},
	}
	for _, c := range cases {
		if got := scanLine(c.line); got != c.want {
			t.Errorf("%s: scanLine(%q) = %q, want %q", c.name, c.line, got, c.want)
		}
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
// The three rules run over each block, so there is nothing to accuse — zero
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
