package logger

import (
	"context"
	"log/slog"
)

// placeKey é a chave reservada do lugar (o subsistema que emitiu a linha).
// Viaja como atributo do slog.Record; cada sink decide como renderizá-la.
const placeKey = "place"

// router faz fan-out de um registro para os sinks.
//
// A política de nível não mora aqui: cada sink carrega o seu próprio Enabled,
// porque as duas regras divergem justamente no TRACE (ADR 0001). O router só
// pergunta e reparte.
type router struct {
	sinks []slog.Handler
}

func (r *router) Enabled(ctx context.Context, level slog.Level) bool {
	for _, s := range r.sinks {
		if s.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (r *router) Handle(ctx context.Context, rec slog.Record) error {
	var first error
	for _, s := range r.sinks {
		if !s.Enabled(ctx, rec.Level) {
			continue
		}
		// Clone porque os sinks consomem os atributos do registro.
		if err := s.Handle(ctx, rec.Clone()); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (r *router) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(r.sinks))
	for i, s := range r.sinks {
		out[i] = s.WithAttrs(attrs)
	}
	return &router{sinks: out}
}

func (r *router) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(r.sinks))
	for i, s := range r.sinks {
		out[i] = s.WithGroup(name)
	}
	return &router{sinks: out}
}

// newRouter monta os sinks ligados pela config. Config.File vazio desliga o
// arquivo; newFileHandler devolve nil quando não consegue abri-lo, e nesse caso
// o console segue sozinho.
func newRouter(cfg Config) slog.Handler {
	var sinks []slog.Handler
	if cfg.Console {
		sinks = append(sinks, newConsoleHandler(cfg))
	}
	if cfg.File != "" {
		if h := newFileHandler(cfg); h != nil {
			sinks = append(sinks, h)
		}
	}
	return &router{sinks: sinks}
}
