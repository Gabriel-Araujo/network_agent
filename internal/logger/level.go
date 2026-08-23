package logger

import "log/slog"

// Níveis expostos pelo logger.
//
// LevelTrace ocupa um valor abaixo de LevelDebug por conveniência de
// roteamento, e isso engana: TRACE é um canal de destino, não uma gravidade.
// Mensagens TRACE ignoram o threshold configurado, vão sempre para o arquivo e
// nunca para o console. Ver docs/adr/0001-trace-e-um-canal-nao-um-nivel.md.
const (
	LevelTrace = slog.LevelDebug - 4
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// levelName devolve o rótulo do nível. slog.Level.String() renderiza TRACE como
// "DEBUG-4", que não serve nem para o console nem para o JSON.
func levelName(l slog.Level) string {
	switch {
	case l <= LevelTrace:
		return "TRACE"
	case l < LevelInfo:
		return "DEBUG"
	case l < LevelWarn:
		return "INFO"
	case l < LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}
