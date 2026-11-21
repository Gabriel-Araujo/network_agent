package llm

import (
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The truth criterion of a Tech pack of the frr-file-builder
// (.agent/frr-tech-packs/issues/04-como-provar-o-criterio-de-verdade.md):
// every example passes vtysh --dryrun on the FRR image of record, every line
// of the Shape and of the examples is a clicmd of the corpus, and every rule
// of doctrine.md and checklist.md cites a section that exists in it.

const (
	packsDir  = "resources/agents/skills/frr-file-builder/references"
	corpusDir = "resources/frrouting/docs"
	frrImage  = "quay.io/frrouting/frr:10.5.0"
)

// retrofitPending holds the packs born before the corpus mold, out of the
// clicmd and citation checkers: bgp writes vrrp, pbr-map and GRE, which have
// no clicmd, and has no Source column. The Retrofit of the bgp pack decides
// whether it joins them and deletes the entry
// (.agent/frr-tech-packs/issues/15-retrofit-do-pack-bgp-ao-molde.md).
var retrofitPending = map[string]bool{"bgp": true}

// bodyForms are the lines the skill body writes that no clicmd documents
// (.agent/frr-tech-packs/issues/03-objetos-transversais.md): one form per
// entry, in clicmd grammar, with the section whose example block shows it.
// TestBodyFormsAreShownInTheirSection holds each to that section.
var bodyForms = []struct{ form, source string }{
	{"key chain NAME", "ospfd › Interfaces"},
	{"key (0-2147483647)", "ospfd › Interfaces"},
	{"cryptographic-algorithm hmac-sha-256", "ospfd › Interfaces"},
	{"route-map NAME (permit|deny) (1-65535)", "bgp › Example Configuration"},
	{"match ip address prefix-list NAME", "bgp › Example Configuration"},
	{"ip address A.B.C.D/M", "ospfd › Sample Configuration"},
	{"interface NAME vrf NAME", "isisd › ISIS Vrf Configuration Examples"},
}

// TestPackExamplesPassDryRun feeds the first fenced block of every example of
// every pack to vtysh --dryrun, one container per pack. It skips without
// podman or the image, which a stopped Podman machine also looks like: a
// production ticket reads PASS here, never SKIP.
func TestPackExamplesPassDryRun(t *testing.T) {
	t.Chdir(repoRoot(t))
	podman, err := exec.LookPath("podman")
	if err != nil {
		t.Skip("no podman in PATH")
	}
	if exec.Command(podman, "image", "exists", frrImage).Run() != nil {
		t.Skipf("%s is not local, or the Podman machine is down", frrImage)
	}

	for _, slug := range packSlugs(t) {
		t.Run(slug, func(t *testing.T) {
			dir := t.TempDir()
			want := map[string]bool{}
			for _, path := range packExamples(t, slug) {
				block, ok := firstFence(readText(t, path))
				if !ok {
					t.Errorf("%s has no fenced block", path)
					continue
				}
				name := strings.TrimSuffix(filepath.Base(path), ".md") + ".conf"
				if err := os.WriteFile(filepath.Join(dir, name), []byte(block), 0o644); err != nil {
					t.Fatal(err)
				}
				want[name] = true
			}

			script := `for f in /w/*.conf; do echo "== ${f##*/}"; vtysh --dryrun -f "$f" 2>&1; echo "rc=$?"; done`
			out, err := exec.Command(podman, "run", "--rm", "--network", "none", "-v", dir+":/w:ro",
				"--entrypoint", "/bin/sh", frrImage, "-c", script).CombinedOutput()
			if err != nil {
				t.Fatalf("podman run: %v\n%s", err, out)
			}

			var name string
			var failed []string
			for line := range strings.Lines(string(out)) {
				line = strings.TrimSpace(line)
				switch {
				case strings.HasPrefix(line, "== "):
					name, failed = strings.TrimPrefix(line, "== "), nil
				case strings.HasPrefix(line, "line "):
					failed = append(failed, line)
				case strings.HasPrefix(line, "rc="):
					if line != "rc=0" {
						t.Errorf("%s: vtysh --dryrun %s\n%s", name, line, strings.Join(failed, "\n"))
					}
					delete(want, name)
				}
			}
			for name := range want {
				t.Errorf("%s: no vtysh result in the container output\n%s", name, out)
			}
		})
	}
}

// TestPackLinesAreCorpusCommands matches every line of the Shape of
// syntax.md and of every example against the clicmd grammar of the whole
// corpus plus bodyForms. A leading `no ` is implicit; comments, exit*, end
// and blank lines are skipped; in the Shape a `<...>` stands for one token.
// The ceiling: it proves the command exists somewhere in the corpus, not in
// the node the line sits in. The dry-run proves the node.
func TestPackLinesAreCorpusCommands(t *testing.T) {
	t.Chdir(repoRoot(t))
	grammars := corpusGrammars(t)

	for _, slug := range packSlugs(t) {
		if retrofitPending[slug] {
			continue
		}
		t.Run(slug, func(t *testing.T) {
			files := append([]string{filepath.Join(packsDir, slug, "syntax.md")}, packExamples(t, slug)...)
			for _, path := range files {
				block, ok := firstFence(readText(t, path))
				if !ok {
					t.Errorf("%s has no fenced block", path)
					continue
				}
				for i, line := range strings.Split(block, "\n") {
					if !isCorpusCommand(grammars, line) {
						t.Errorf("%s, block line %d: no clicmd of the corpus matches %q", path, i+1, strings.TrimSpace(line))
					}
				}
			}
		})
	}
}

// TestPackRulesCiteTheCorpus holds doctrine.md and checklist.md to their
// Source: every table carries a Source column and every row fills it; every
// section without a table closes with a [...] citation. Every citation found
// in the pack, there, in the rules of syntax.md and in the Sources line every
// example carries, names a `.rst` that exists and a section title in it.
func TestPackRulesCiteTheCorpus(t *testing.T) {
	t.Chdir(repoRoot(t))
	titles := corpusTitles(t)

	for _, slug := range packSlugs(t) {
		if retrofitPending[slug] {
			continue
		}
		t.Run(slug, func(t *testing.T) {
			check := func(where, citations string) {
				t.Helper()
				for _, c := range strings.Split(strings.Join(strings.Fields(citations), " "), ";") {
					if err := citationError(titles, strings.TrimSpace(c)); err != "" {
						t.Errorf("%s: %s", where, err)
					}
				}
			}
			bracket := regexp.MustCompile(`\[([^\[\]]*›[^\[\]]*)\]`)

			for _, name := range []string{"doctrine.md", "checklist.md"} {
				path := filepath.Join(packsDir, slug, name)
				for _, sec := range sections(readText(t, path)) {
					where := path + " › " + sec.title
					tables := tablesOf(sec.body)
					if len(tables) == 0 {
						body := strings.TrimSpace(sec.body)
						open := strings.LastIndex(body, "[")
						if !strings.HasSuffix(body, "]") || open < 0 {
							t.Errorf("%s: a section with no table closes with a [<rst> › <section>] citation", where)
							continue
						}
						check(where, body[open+1:len(body)-1])
						continue
					}
					for _, table := range tables {
						col := slices.Index(table[0], "Source")
						if col < 0 {
							t.Errorf("%s: table %v has no Source column", where, table[0])
							continue
						}
						for _, row := range table[1:] {
							if col >= len(row) || row[col] == "" {
								t.Errorf("%s: row %v has no Source", where, row)
								continue
							}
							check(where, row[col])
						}
					}
				}
			}

			syntax := filepath.Join(packsDir, slug, "syntax.md")
			for _, m := range bracket.FindAllStringSubmatch(readText(t, syntax), -1) {
				check(syntax, m[1])
			}
			for _, path := range packExamples(t, slug) {
				_, sources, ok := strings.Cut(readText(t, path), "\nSources:")
				if !ok {
					t.Errorf("%s has no Sources line", path)
					continue
				}
				sources, _, _ = strings.Cut(sources, "\n\n")
				check(path, strings.TrimSuffix(strings.TrimSpace(sources), "."))
			}
		})
	}
}

// TestBodyFormsAreShownInTheirSection keeps bodyForms honest: each form
// matches a line of the example blocks of the section it cites.
func TestBodyFormsAreShownInTheirSection(t *testing.T) {
	t.Chdir(repoRoot(t))
	titles := corpusTitles(t)
	prompt := regexp.MustCompile(`^[\w.-]+(\([\w-]+\))?#`)
	for _, bf := range bodyForms {
		if err := citationError(titles, bf.source); err != "" {
			t.Errorf("%q: %s", bf.form, err)
			continue
		}
		rst, title, _ := strings.Cut(bf.source, " › ")
		g := parseClicmd(bf.form)
		shown := false
		for line := range strings.Lines(sectionText(t, rst, title)) {
			line = prompt.ReplaceAllString(strings.TrimSpace(line), "")
			if g.matches(strings.Fields(line)) {
				shown = true
				break
			}
		}
		if !shown {
			t.Errorf("%q is shown in no line of %s", bf.form, bf.source)
		}
	}
}

// TestPackCheckersCatchTheirFailures is the negative control of the two
// offline checkers: each case is a mistake a pack has made or would make.
func TestPackCheckersCatchTheirFailures(t *testing.T) {
	t.Chdir(repoRoot(t))
	grammars := corpusGrammars(t)
	for line, want := range map[string]bool{
		" ip ospf network point-to-point":     true,
		" no ip ospf passive":                 true,
		" ip ospf area 0.0.0.10":              true,
		" area 0.0.0.10 stub no-summary":      true,
		"interface <transit interface>":       true,
		" ospf router-id <router-id>":         true,
		" ! WARNING: anything at all":         true,
		" ip ospf network p2p":                false,
		" ospf router-id 10.0.0.300":          false,
		" ip ospf cost 70000":                 false,
		" area 0.0.0.10 totally-stubby":       false,
		" ospf router-id 10.0.0.1   ! a note": false,
	} {
		if got := isCorpusCommand(grammars, line); got != want {
			t.Errorf("isCorpusCommand(%q) = %v, want %v", line, got, want)
		}
	}

	titles := corpusTitles(t)
	for citation, valid := range map[string]bool{
		"ospfd › Interfaces":                     true,
		"ospf_fundamentals › The Hello Protocol": true,
		"ospf-fundamentals › The Hello Protocol": false,
		"ospfd › Interface":                      false,
		"ospfd Interfaces":                       false,
	} {
		if got := citationError(titles, citation) == ""; got != valid {
			t.Errorf("citation %q valid = %v, want %v", citation, got, valid)
		}
	}
}

// packSlugs lists the pack folders of the skill.
func packSlugs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		t.Fatal(err)
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			slugs = append(slugs, e.Name())
		}
	}
	return slugs
}

// packExamples lists the examples of one pack.
func packExamples(t *testing.T, slug string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(packsDir, slug, "examples", "*.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("pack %s has no examples/*.md", slug)
	}
	return files
}

// readText reads a file with its line endings normalized to \n.
func readText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// firstFence returns the content of the first ``` block of a markdown text.
func firstFence(text string) (string, bool) {
	_, rest, ok := strings.Cut(text, "```")
	if !ok {
		return "", false
	}
	_, rest, _ = strings.Cut(rest, "\n")
	block, _, ok := strings.Cut(rest, "\n```")
	return block + "\n", ok
}

type section struct{ title, body string }

// sections splits a markdown text on its `## ` headings; the text before the
// first one is not a section.
func sections(text string) []section {
	var out []section
	for _, part := range strings.Split(text, "\n## ")[1:] {
		title, body, _ := strings.Cut(part, "\n")
		out = append(out, section{title, body})
	}
	return out
}

// tablesOf returns the markdown tables of a section, each as its rows of
// trimmed cells: the header first, the |---| line left out.
func tablesOf(body string) [][][]string {
	var tables [][][]string
	var cur [][]string
	for line := range strings.Lines(body + "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			if cur != nil {
				tables, cur = append(tables, cur), nil
			}
			continue
		}
		if strings.HasPrefix(line, "|-") || strings.HasPrefix(line, "| -") {
			continue
		}
		var row []string
		for cell := range strings.SplitSeq(strings.Trim(line, "|"), "|") {
			row = append(row, strings.TrimSpace(cell))
		}
		cur = append(cur, row)
	}
	return tables
}

// isUnderline reports whether s is a run of one `.rst` adornment character.
func isUnderline(s string) bool {
	return len(s) > 1 && strings.ContainsAny(s[:1], `=-~^*"+#`) && strings.Count(s, s[:1]) == len(s)
}

// corpusTitles maps every `.rst` of the corpus, by name without extension, to
// its section titles.
func corpusTitles(t *testing.T) map[string]map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(corpusDir, "*.rst"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no .rst under %s", corpusDir)
	}
	titles := map[string]map[string]bool{}
	for _, f := range files {
		set := map[string]bool{}
		lines := strings.Split(readText(t, f), "\n")
		for i := range len(lines) - 1 {
			if title, ok := titleAt(lines, i); ok {
				set[title] = true
			}
		}
		titles[strings.TrimSuffix(filepath.Base(f), ".rst")] = set
	}
	return titles
}

// titleAt reports whether line i is a section title: text underlined by a
// run of one punctuation character at least as long.
func titleAt(lines []string, i int) (string, bool) {
	title, under := strings.TrimSpace(lines[i]), strings.TrimSpace(lines[i+1])
	ok := title != "" && !isUnderline(title) && isUnderline(under) && len(under) >= len(title)
	return title, ok
}

// sectionText returns the text of one section of a `.rst`, up to the next
// title of any level.
func sectionText(t *testing.T, rst, title string) string {
	t.Helper()
	lines := strings.Split(readText(t, filepath.Join(corpusDir, rst+".rst")), "\n")
	var out []string
	in := false
	for i := range len(lines) - 1 {
		if got, ok := titleAt(lines, i); ok {
			if in {
				break
			}
			in = got == title
			continue
		}
		if in {
			out = append(out, lines[i])
		}
	}
	return strings.Join(out, "\n")
}

// citationError says what is wrong with one `<rst> › <section>` citation, or
// "" when the `.rst` exists and has that section.
func citationError(titles map[string]map[string]bool, citation string) string {
	rst, title, ok := strings.Cut(citation, " › ")
	switch {
	case !ok:
		return "citation " + strconv.Quote(citation) + " is not <rst> › <section>"
	case titles[rst] == nil:
		return "citation " + strconv.Quote(citation) + ": no " + rst + ".rst in the corpus"
	case !titles[rst][title]:
		return "citation " + strconv.Quote(citation) + ": " + rst + ".rst has no section " + strconv.Quote(title)
	}
	return ""
}

// corpusGrammars parses every `.. clicmd::` of the corpus, plus bodyForms.
func corpusGrammars(t *testing.T) []*gram {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(corpusDir, "*.rst"))
	directive := regexp.MustCompile(`(?m)^\s*\.\. clicmd:: (.+)$`)
	var out []*gram
	for _, f := range files {
		for _, m := range directive.FindAllStringSubmatch(readText(t, f), -1) {
			out = append(out, parseClicmd(m[1]))
		}
	}
	for _, bf := range bodyForms {
		out = append(out, parseClicmd(bf.form))
	}
	return out
}

// placeholder is the token a Shape `<...>` becomes: it matches any one slot.
const placeholder = "\x00"

var shapeSlot = regexp.MustCompile(`<[^>]*>`)

// isCorpusCommand reports whether one block line is a corpus command, or a
// line the checker skips.
func isCorpusCommand(grammars []*gram, line string) bool {
	toks := strings.Fields(shapeSlot.ReplaceAllString(line, placeholder))
	if len(toks) == 0 || strings.HasPrefix(toks[0], "!") || strings.HasPrefix(toks[0], "exit") || toks[0] == "end" {
		return true
	}
	for _, g := range grammars {
		if g.matches(toks) || (toks[0] == "no" && g.matches(toks[1:])) {
			return true
		}
	}
	return false
}

// gram is one node of a clicmd grammar: a word (literal, variable or range)
// or a group of alternatives, each a sequence of nodes. Brackets make a group
// optional; braces let it repeat.
type gram struct {
	word     string
	alts     [][]*gram
	optional bool
	repeat   bool
}

var clicmdToken = regexp.MustCompile(`[\[\]()<>{}|]|[^\s\[\]()<>{}|]+`)

// parseClicmd parses a clicmd syntax: `[opt]`, `(a|b)`, `<a|b>`, `{a|b}`,
// ranges `(1-65535)`, typed variables (`A.B.C.D`, `A.B.C.D/M`, `X:X::X:X`)
// and any other upper-case word as a variable.
func parseClicmd(syntax string) *gram {
	toks := clicmdToken.FindAllString(syntax, -1)
	g, _ := parseGroup(toks, 0, "")
	return g
}

func parseGroup(toks []string, i int, closer string) (*gram, int) {
	g := &gram{alts: [][]*gram{nil}}
	for i < len(toks) {
		tok := toks[i]
		i++
		switch tok {
		case closer:
			return g, i
		case "|":
			g.alts = append(g.alts, nil)
			continue
		case "[", "(", "<", "{":
			var kid *gram
			kid, i = parseGroup(toks, i, map[string]string{"[": "]", "(": ")", "<": ">", "{": "}"}[tok])
			kid.optional, kid.repeat = tok == "[", tok == "{"
			g.alts[len(g.alts)-1] = append(g.alts[len(g.alts)-1], kid)
			continue
		}
		last := len(g.alts) - 1
		g.alts[last] = append(g.alts[last], &gram{word: tok})
	}
	return g, i
}

// matches reports whether the grammar consumes exactly toks.
func (g *gram) matches(toks []string) bool {
	return slices.Contains(g.ends(toks, 0), len(toks))
}

// ends returns every position the node can stop at, starting from i.
func (g *gram) ends(toks []string, i int) []int {
	if g.alts == nil {
		if i < len(toks) && wordMatches(g.word, toks[i]) {
			return []int{i + 1}
		}
		return nil
	}
	var out []int
	if g.optional {
		out = append(out, i)
	}
	frontier := []int{i}
	for len(frontier) > 0 {
		var next []int
		for _, alt := range g.alts {
			for _, start := range frontier {
				pos := []int{start}
				for _, kid := range alt {
					var step []int
					for _, p := range pos {
						step = append(step, kid.ends(toks, p)...)
					}
					pos = step
				}
				next = append(next, pos...)
			}
		}
		// Only positions not reached before go on: a repeat stops when a
		// round adds none.
		next = slices.DeleteFunc(slices.Compact(slices.Sorted(slices.Values(next))), func(p int) bool {
			return slices.Contains(out, p)
		})
		out = append(out, next...)
		if !g.repeat {
			break
		}
		frontier = next
	}
	return out
}

var rangeWord = regexp.MustCompile(`^(\d+)-(\d+)$`)

// wordMatches reports whether one line token fills one grammar word.
func wordMatches(word, tok string) bool {
	if tok == placeholder {
		return true
	}
	if m := rangeWord.FindStringSubmatch(word); m != nil {
		n, err := strconv.ParseInt(tok, 10, 64)
		lo, _ := strconv.ParseInt(m[1], 10, 64)
		hi, _ := strconv.ParseInt(m[2], 10, 64)
		return err == nil && lo <= n && n <= hi
	}
	switch word {
	case "A.B.C.D":
		a, err := netip.ParseAddr(tok)
		return err == nil && a.Is4()
	case "A.B.C.D/M":
		p, err := netip.ParsePrefix(tok)
		return err == nil && p.Addr().Is4()
	case "X:X::X:X":
		a, err := netip.ParseAddr(tok)
		return err == nil && a.Is6()
	case "X:X::X:X/M":
		p, err := netip.ParsePrefix(tok)
		return err == nil && p.Addr().Is6()
	}
	if strings.ToUpper(word) == word && strings.ToLower(word) != word {
		return true
	}
	return word == tok
}
