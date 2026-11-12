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
