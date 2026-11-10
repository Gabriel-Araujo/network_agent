package main

import (
	"fmt"
	"strings"
	"testing"
)

const esc = "\x1b"

// TestLogRingStripsANSI fixa a única transformação que a gaveta aplica. O
// escape de cor chega ou não conforme o os.Stderr ser TTY — o que varia com a
// forma de iniciar o app —, e a gaveta colore pelo token [INFO ], não por ANSI.
func TestLogRingStripsANSI(t *testing.T) {
	r := &logRing{}
	line := "2026-01-01 00:00:00.000 [" + esc + "[32mINFO" + esc + "[0m ] UI - o app subiu\n"

	n, err := r.Write([]byte(line))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// O io.MultiWriter do console trata n < len(p) como ErrShortWrite e
	// derrubaria a escrita no stderr junto.
	if n != len(line) {
		t.Fatalf("Write devolveu %d, esperado %d", n, len(line))
	}

	got := r.Lines()
	want := "2026-01-01 00:00:00.000 [INFO ] UI - o app subiu"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("linhas = %q, esperado [%q]", got, want)
	}
	if strings.Contains(got[0], esc) {
		t.Fatalf("sobrou escape ANSI na linha: %q", got[0])
	}
}

// TestLogRingSplitsAndCaps cobre as duas outras coisas que o anel faz: quebrar
// em linhas (o valor de um atributo pode ter \n dentro, como um erro cru de
// driver) e guardar só as últimas maxLogLines.
func TestLogRingSplitsAndCaps(t *testing.T) {
	r := &logRing{}
	if _, err := r.Write([]byte("primeira\nsegunda\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := r.Lines(); len(got) != 2 {
		t.Fatalf("um Write com duas linhas rendeu %d: %q", len(got), got)
	}

	for i := range maxLogLines {
		if _, err := r.Write([]byte(fmt.Sprintf("linha %d\n", i))); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	got := r.Lines()
	if len(got) != maxLogLines {
		t.Fatalf("o anel guardou %d linhas, esperado o teto de %d", len(got), maxLogLines)
	}
	if got[len(got)-1] != fmt.Sprintf("linha %d", maxLogLines-1) {
		t.Fatalf("a última linha não é a mais nova: %q", got[len(got)-1])
	}
	if got[0] == "primeira" {
		t.Fatal("o anel não descartou as linhas mais antigas")
	}
}
