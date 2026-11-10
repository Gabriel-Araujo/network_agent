package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestConsoleTee fixa a ponte da gaveta de Logs da UI: com Config.ConsoleTee
// preenchido, a linha vai ao stderr E ao writer, byte a byte. É essa igualdade
// que faz a gaveta não poder divergir do terminal — se este teste quebrar, a
// gaveta deixou de mostrar "as mesmas mensagens que aparecem no cli".
func TestConsoleTee(t *testing.T) {
	var tee bytes.Buffer
	stderr := captureStderr(t)

	Init(Config{
		Level:      LevelInfo,
		Console:    true,
		Color:      ColorNever,
		MaxValue:   defaultMaxValue,
		ConsoleTee: &tee,
	})
	Named("TEST").Info("linha do tee", "k", "v")

	out := stderr()
	if !strings.Contains(out, "linha do tee") {
		t.Fatalf("o stderr não recebeu a linha: %q", out)
	}
	if tee.String() != out {
		t.Fatalf("tee e stderr divergiram:\n tee=%q\n stderr=%q", tee.String(), out)
	}
}

// TestConsoleWithoutTee é o outro lado: campo nil, console sozinho. É o que o
// cmd/cli, o ingester e o chucker fazem, e eles não mudaram em nada.
func TestConsoleWithoutTee(t *testing.T) {
	stderr := captureStderr(t)

	Init(Config{Level: LevelInfo, Console: true, Color: ColorNever, MaxValue: defaultMaxValue})
	Named("TEST").Info("linha sem tee")

	if out := stderr(); !strings.Contains(out, "linha sem tee") {
		t.Fatalf("o console sem tee perdeu a linha: %q", out)
	}
}

// captureStderr troca o os.Stderr por um pipe e devolve o leitor. A troca tem
// de acontecer antes do Init: o newConsoleHandler lê o os.Stderr na construção.
//
// O handler ativo volta a nil no fim, para um teste não herdar a config do
// outro — handler() reconstrói o default sozinho.
func captureStderr(t *testing.T) func() string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = orig
		current.Store(nil)
	})

	return func() string {
		os.Stderr = orig
		// Fecha a ponta de escrita antes de ler: o volume é de uma linha, bem
		// abaixo do buffer do pipe, então não há risco de bloqueio.
		if err := w.Close(); err != nil {
			t.Fatalf("fechando o pipe: %v", err)
		}
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("lendo o pipe: %v", err)
		}
		_ = r.Close()
		return string(b)
	}
}
