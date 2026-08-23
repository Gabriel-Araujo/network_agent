package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/paths"
)

// nameLayout nomeia o arquivo da execução. A ordem YYMMDD-HHMMSS faz a ordem
// lexicográfica coincidir com a cronológica, e é disso que a poda depende para
// achar os mais recentes sem statar arquivo nenhum.
const nameLayout = "060102-150405"

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// fileHandler é o sink de arquivo: JSON, um arquivo por execução.
//
// Ao contrário do console, aqui o formato não é nosso — é o do slog, e o que
// ele escreve já é o que foi decidido (RFC3339 com ms no time, um registro por
// Write, sob mutex). Por isso o JSONHandler fica embutido em vez de
// reimplementado: quem não é dono do formato não precisa ser dono dos
// atributos.
//
// O que sobra para este tipo é a política de nível, que é o único ponto do
// pacote onde o TRACE escapa do threshold.
type fileHandler struct {
	slog.Handler
	level slog.Level
}

// Enabled deixa passar o threshold e, além dele, todo TRACE — TRACE é canal de
// destino, não gravidade (ADR 0001). Com LOG_LEVEL=info, o arquivo recebe Info+
// e todo TRACE, sem receber Debug: a lacuna parece bug e não é. Remover a
// cláusula do TRACE daqui quebra a decisão.
func (h *fileHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level <= LevelTrace || level >= h.level
}

// WithAttrs e WithGroup re-embrulham o resultado. Sem isso devolveriam o
// JSONHandler puro, e o Enabled acima — a regra do TRACE — sumiria na primeira
// chamada.

func (h *fileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &fileHandler{Handler: h.Handler.WithAttrs(attrs), level: h.level}
}

func (h *fileHandler) WithGroup(name string) slog.Handler {
	return &fileHandler{Handler: h.Handler.WithGroup(name), level: h.level}
}

// newFileHandler abre o arquivo desta execução e devolve o sink.
//
// Devolve nil quando não consegue: falha ao escrever log não pode derrubar o
// programa nem calar o console, então vira um aviso em stderr e o logger segue
// só com o console.
func newFileHandler(cfg Config) slog.Handler {
	f, err := openLogFile(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: seguindo sem arquivo de log (%v)\n", err)
		return nil
	}

	// Escrita direta no arquivo, sem bufio: Fatal sai por os.Exit e Panic por
	// panic, nenhum dos dois passa por um Close, e um buffer perderia
	// justamente a última linha — a que interessa.
	return &fileHandler{
		Handler: slog.NewJSONHandler(f, &slog.HandlerOptions{
			Level:       LevelTrace, // filtrar é do Enabled acima; aqui nunca.
			AddSource:   true,
			ReplaceAttr: replaceForFile(repoRootPrefix()),
		}),
		level: cfg.Level,
	}
}

// openLogFile resolve <File>/<binario>/<YYMMDD-HHMMSS>.log, cria a árvore, abre
// o arquivo e poda os antigos.
func openLogFile(cfg Config) (*os.File, error) {
	root := cfg.File
	if !filepath.IsAbs(root) {
		// Caminho relativo é relativo à raiz do módulo, não ao CWD: os
		// binários deste repo são rodados de qualquer pasta.
		resolved, err := paths.RepoFile(root)
		if err != nil {
			return nil, err
		}
		root = resolved
	}

	dir := filepath.Join(root, binaryName(cfg.Binary))
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, time.Now().Format(nameLayout)+".log")
	// O_APPEND, não O_TRUNC: duas execuções no mesmo segundo colidem de nome, e
	// nesse caso é melhor dividirem o arquivo do que uma apagar a da outra.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePerm)
	if err != nil {
		return nil, err
	}

	// Depois de abrir, para o arquivo desta execução contar no total mantido.
	prune(dir, cfg.MaxFiles)
	return f, nil
}

// prune mantém os keep arquivos mais recentes da pasta e apaga o resto.
//
// Por pasta de binário, para rodar o ingester 50 vezes não varrer o histórico
// do cli. A ordem é a do nome, o que só funciona porque nameLayout é ordenável.
// Erros são ignorados de propósito: poda é higiene, não é a função do logger.
func prune(dir string, keep int) {
	if keep <= 0 {
		return
	}
	entries, err := os.ReadDir(dir) // já vem ordenado por nome.
	if err != nil {
		return
	}

	var logs []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".log" {
			logs = append(logs, e.Name())
		}
	}
	for i := 0; i < len(logs)-keep; i++ {
		_ = os.Remove(filepath.Join(dir, logs[i]))
	}
}

// binaryName é a pasta desta execução: o binário sem .exe. O dev está em
// Windows, e sem o corte a árvore ganharia uma pasta "cli.exe".
func binaryName(override string) string {
	if override != "" {
		return override
	}
	name := filepath.Base(os.Args[0])
	if ext := filepath.Ext(name); strings.EqualFold(ext, ".exe") {
		name = strings.TrimSuffix(name, ext)
	}
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "app" // os.Args[0] vazio; a árvore precisa de algum nome.
	}
	return name
}

// replaceForFile ajusta os dois campos que o slog renderiza de um jeito que não
// serve aqui: o nível, que sairia "DEBUG-4" no TRACE, e o source, que sairia
// como objeto com o caminho absoluto da máquina onde se compilou.
func replaceForFile(trim string) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) > 0 {
			return a // só os campos de topo são nossos.
		}
		switch a.Key {
		case slog.LevelKey:
			if l, ok := a.Value.Any().(slog.Level); ok {
				a.Value = slog.StringValue(levelName(l))
			}
		case slog.SourceKey:
			if src, ok := a.Value.Any().(*slog.Source); ok {
				a.Value = slog.StringValue(sourceRef(src, trim))
			}
		}
		return a
	}
}

// sourceRef achata o source em "internal/rag/retriever.go:88": relativo à raiz
// do módulo quando o binário foi compilado aqui, absoluto quando não foi.
func sourceRef(src *slog.Source, trim string) string {
	file := filepath.ToSlash(src.File)
	if trim != "" {
		file = strings.TrimPrefix(file, trim)
	}
	return fmt.Sprintf("%s:%d", file, src.Line)
}

// repoRootPrefix é a raiz do módulo com barra final, para encurtar o source.
// Vazio quando não dá para descobrir — aí o caminho sai inteiro, que é feio mas
// não é errado.
func repoRootPrefix() string {
	root, err := paths.RepoRoot()
	if err != nil {
		return ""
	}
	return filepath.ToSlash(root) + "/"
}
