package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

// timeLayout é o timestamp do console: milissegundos, sem timezone. O offset é
// ruído em console local; quem precisa dele lê o RFC3339 do arquivo.
const timeLayout = "2006-01-02 15:04:05.000"

// levelWidth alinha o rótulo em 5 chars ("[INFO ]", "[ERROR]") para que a
// coluna do lugar não dance de uma linha para a outra.
const levelWidth = 5

const ansiReset = "\033[0m"

// levelColor colore só o nível. O resto da linha competiria com a mensagem.
var levelColor = map[string]string{
	"DEBUG": "\033[36m", // ciano
	"INFO":  "\033[32m", // verde
	"WARN":  "\033[33m", // amarelo
	"ERROR": "\033[31m", // vermelho
}

// consoleHandler é o sink de console, em texto:
//
//	2026-08-23 14:20:31.482 [INFO ] MAIN - intent salvo path=/x/y.md
//
// Não embute um slog.Handler: a formatação é toda daqui, então os atributos de
// WithAttrs também são, e ficam guardados já renderizados em preformatted.
//
// O source (arquivo:linha) não entra: no console o lugar basta, e a linha só
// polui. Quem quer a linha lê o arquivo.
type consoleHandler struct {
	w  io.Writer
	mu *sync.Mutex // ponteiro: as cópias de WithAttrs/WithGroup escrevem no mesmo w.

	level    slog.Level
	color    bool
	maxValue int

	preformatted []byte   // atributos herdados de WithAttrs, já renderizados.
	groups       []string // prefixo de chave herdado de WithGroup.
}

func newConsoleHandler(cfg Config) slog.Handler {
	return &consoleHandler{
		w:        os.Stderr,
		mu:       &sync.Mutex{},
		level:    cfg.Level,
		color:    useColor(cfg.Color, os.Stderr),
		maxValue: cfg.MaxValue,
	}
}

// useColor resolve o tri-state uma vez, na construção. auto sonda o TTY para
// não cuspir escape ANSI quando a saída é redirecionada com >.
func useColor(c Color, f *os.File) bool {
	switch c {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		fi, err := f.Stat()
		return err == nil && fi.Mode()&os.ModeCharDevice != 0
	}
}

// Enabled aplica o threshold e barra o TRACE incondicionalmente — TRACE é um
// canal só-arquivo, e nenhuma configuração o traz para cá (ADR 0001).
func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	if level <= LevelTrace {
		return false
	}
	return level >= h.level
}

func (h *consoleHandler) Handle(_ context.Context, rec slog.Record) error {
	// O lugar viaja como atributo reservado, mas na linha ele é coluna: sai
	// daqui antes do resto, senão apareceria duas vezes.
	place := ""
	attrs := make([]slog.Attr, 0, rec.NumAttrs())
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == placeKey {
			place = a.Value.String()
			return true
		}
		attrs = append(attrs, a)
		return true
	})

	buf := make([]byte, 0, 256)
	buf = rec.Time.AppendFormat(buf, timeLayout)
	buf = append(buf, ' ')
	buf = h.appendLevel(buf, rec.Level)
	buf = append(buf, ' ')
	if place != "" {
		buf = append(buf, place...)
		buf = append(buf, " - "...)
	}
	buf = append(buf, rec.Message...)
	buf = append(buf, h.preformatted...)
	buf = h.appendAttrs(buf, attrs)
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	c := *h
	c.preformatted = c.appendAttrs(slices.Clip(h.preformatted), attrs)
	return &c
}

func (h *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.groups = append(slices.Clip(h.groups), name)
	return &c
}

// appendLevel escreve "[INFO ]", com o padding fora do escape de cor para que
// o alinhamento não dependa de o terminal engolir o escape.
func (h *consoleHandler) appendLevel(buf []byte, l slog.Level) []byte {
	name := levelName(l)
	buf = append(buf, '[')
	if h.color {
		buf = append(buf, levelColor[name]...)
		buf = append(buf, name...)
		buf = append(buf, ansiReset...)
	} else {
		buf = append(buf, name...)
	}
	for i := len(name); i < levelWidth; i++ {
		buf = append(buf, ' ')
	}
	return append(buf, ']')
}

func (h *consoleHandler) appendAttrs(buf []byte, attrs []slog.Attr) []byte {
	prefix := ""
	if len(h.groups) > 0 {
		prefix = strings.Join(h.groups, ".") + "."
	}
	for _, a := range attrs {
		buf = h.appendAttr(buf, prefix, a)
	}
	return buf
}

func (h *consoleHandler) appendAttr(buf []byte, prefix string, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf // o contrato do slog manda ignorar atributo vazio.
	}

	if a.Value.Kind() == slog.KindGroup {
		group := a.Value.Group()
		if len(group) == 0 {
			return buf
		}
		if a.Key != "" {
			prefix += a.Key + "."
		}
		for _, g := range group {
			buf = h.appendAttr(buf, prefix, g)
		}
		return buf
	}

	buf = append(buf, ' ')
	buf = append(buf, prefix...)
	buf = append(buf, a.Key...)
	buf = append(buf, '=')
	return append(buf, truncate(a.Value.String(), h.maxValue)...)
}

// truncate corta o valor em max runes. É a única perda de informação do
// logger, e existe só no console: o arquivo JSON guarda o valor inteiro.
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max || utf8.RuneCountInString(s) <= max {
		return s
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i] + "…"
		}
		n++
	}
	return s
}
