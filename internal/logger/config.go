package logger

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Color é o tri-state de coloração do console. Os valores são a grafia da env
// var, para que Config e LOG_COLOR digam literalmente a mesma coisa.
type Color string

const (
	ColorAuto   Color = "auto"
	ColorAlways Color = "true"
	ColorNever  Color = "false"
)

const (
	defaultMaxValue = 512
	defaultMaxFiles = 10
)

// Config descreve os dois sinks. Não existe formato configurável: o console é
// sempre texto e o arquivo é sempre JSON — são sinks distintos, com regras
// distintas, e não duas renderizações de uma mesma saída.
type Config struct {
	Level    slog.Level
	Console  bool
	Color    Color
	MaxValue int
	File     string
	MaxFiles int
	Binary   string
}

// ConfigFromEnv lê a config das env vars reais do processo.
//
// Deliberadamente não lê o .env do repo: o logger precisa existir antes do
// carregamento de config, que morre com panic quando falha. Valor inválido cai
// no default em silêncio — o logger nunca aborta o programa por causa da sua
// própria configuração.
func ConfigFromEnv() Config {
	cfg := Config{
		Level:    LevelInfo,
		Console:  true,
		Color:    ColorAuto,
		MaxValue: defaultMaxValue,
		MaxFiles: defaultMaxFiles,
	}

	cfg.Level = envLevel("LOG_LEVEL", cfg.Level)
	cfg.Console = envBool("LOG_CONSOLE", cfg.Console)
	cfg.Color = envColor("LOG_COLOR", cfg.Color)
	cfg.MaxValue = envPositiveInt("LOG_MAX_VALUE", cfg.MaxValue)
	cfg.MaxFiles = envPositiveInt("LOG_MAX_FILES", cfg.MaxFiles)
	cfg.File = os.Getenv("LOG_FILE")

	return cfg
}

func envLevel(key string, def slog.Level) slog.Level {
	switch strings.ToLower(os.Getenv(key)) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return def
	}
}

func envBool(key string, def bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

func envColor(key string, def Color) Color {
	switch c := Color(strings.ToLower(os.Getenv(key))); c {
	case ColorAuto, ColorAlways, ColorNever:
		return c
	default:
		return def
	}
}

func envPositiveInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil || v <= 0 {
		return def
	}
	return v
}
