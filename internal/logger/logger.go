// Package logger é o logging do projeto: um console em texto para quem está
// olhando e um arquivo JSON por execução para quem vai investigar depois.
//
// Uso:
//
//	// no topo de cada arquivo
//	var log = logger.Named("RAG")
//
//	// uma vez, no main
//	logger.Init(logger.ConfigFromEnv())
//
// O handle devolvido por Named funciona antes de Init — nesse estado o logger
// é console/texto/Info.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

// current guarda o handler ativo. É trocado por baixo dos handles já
// existentes, e é isso que dá o late-binding: Named roda no init de pacote,
// antes do main, e não teria como enxergar a config passada ao Init.
var current atomic.Pointer[slog.Handler]

// Init instala a config. Deve ser chamado uma vez, no início do main.
//
// Não devolve erro de propósito: falha na configuração do logger não é motivo
// para derrubar o programa, e um erro aqui só convidaria os call sites a fazer
// exatamente isso. Falhas degradam para o console.
func Init(cfg Config) {
	h := newRouter(cfg)
	current.Store(&h)
}

// handler devolve o handler ativo, construindo o default preguiçosamente para
// que um log emitido antes do Init ainda saia.
func handler() slog.Handler {
	if h := current.Load(); h != nil {
		return *h
	}
	h := newRouter(Config{
		Level:    LevelInfo,
		Console:  true,
		Color:    ColorAuto,
		MaxValue: defaultMaxValue,
	})
	current.CompareAndSwap(nil, &h)
	return *current.Load()
}

// Logger é um handle nomeado por lugar — o subsistema que emite a linha.
//
// Não guarda config: só o nome. Toda a decisão de para onde a linha vai é
// resolvida na hora da chamada, contra o handler ativo.
type Logger struct {
	place string
}

// Named devolve o handle de um lugar (MAIN, RAG, TOOLS...). Seguro de chamar
// em init de pacote.
func Named(place string) *Logger { return &Logger{place: place} }

// As variantes simples recebem pares chave/valor no estilo do slog:
//
//	log.Info("intent salvo", "path", p)
//
// As variantes com f formatam a mensagem e não levam atributos.

func (l *Logger) Trace(msg string, args ...any) { l.log(LevelTrace, msg, args) }
func (l *Logger) Debug(msg string, args ...any) { l.log(LevelDebug, msg, args) }
func (l *Logger) Info(msg string, args ...any)  { l.log(LevelInfo, msg, args) }
func (l *Logger) Warn(msg string, args ...any)  { l.log(LevelWarn, msg, args) }
func (l *Logger) Error(msg string, args ...any) { l.log(LevelError, msg, args) }

func (l *Logger) Tracef(format string, a ...any) { l.log(LevelTrace, fmt.Sprintf(format, a...), nil) }
func (l *Logger) Debugf(format string, a ...any) { l.log(LevelDebug, fmt.Sprintf(format, a...), nil) }
func (l *Logger) Infof(format string, a ...any)  { l.log(LevelInfo, fmt.Sprintf(format, a...), nil) }
func (l *Logger) Warnf(format string, a ...any)  { l.log(LevelWarn, fmt.Sprintf(format, a...), nil) }
func (l *Logger) Errorf(format string, a ...any) { l.log(LevelError, fmt.Sprintf(format, a...), nil) }

// Fatal registra em Error e encerra o processo com status 1.
func (l *Logger) Fatal(msg string, args ...any) {
	l.log(LevelError, msg, args)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, a ...any) {
	l.log(LevelError, fmt.Sprintf(format, a...), nil)
	os.Exit(1)
}

// Panic registra em Error e entra em pânico com a mensagem.
func (l *Logger) Panic(msg string, args ...any) {
	l.log(LevelError, msg, args)
	panic(msg)
}

func (l *Logger) Panicf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	l.log(LevelError, msg, nil)
	panic(msg)
}

// log monta o registro e o entrega ao handler ativo.
//
// args vem como slice, e não variádico, para que todos os métodos públicos
// fiquem à mesma distância do call site — o pc capturado aqui é o source
// (arquivo:linha) que o sink de arquivo grava.
func (l *Logger) log(level slog.Level, msg string, args []any) {
	ctx := context.Background()
	h := handler()
	if !h.Enabled(ctx, level) {
		return
	}

	// 0: Callers, 1: log, 2: o método público, 3: quem chamou.
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	rec := slog.NewRecord(time.Now(), level, msg, pcs[0])
	rec.AddAttrs(slog.String(placeKey, l.place))
	rec.Add(args...)

	_ = h.Handle(ctx, rec)
}
