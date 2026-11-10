package main

import (
	"bytes"
	"regexp"
	"slices"
	"sync"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
)

// maxLogLines é o teto do anel da gaveta: ~46x o maior run já medido (43
// linhas), então na prática nunca descarta. O teto existe para um app aberto o
// dia inteiro não crescer sem cota, e é uma linha de código.
const maxLogLines = 2000

// logs é o anel da gaveta de Logs. Variável de pacote, e não campo do App,
// porque o logger.Init roda antes do NewApp() (main.go) e as quatro primeiras
// linhas de todo run — CONFIG e MODELS, onde "falha ao ler a configuração" e
// "falha ao conectar ao provedor" aparecem — nascem justamente ali.
var logs = &logRing{}

// logRing é o segundo destino do sink de console (logger.Config.ConsoleTee):
// guarda as últimas maxLogLines linhas que o console escreveu. A igualdade com
// o terminal é por construção — os dois destinos recebem os mesmos bytes do
// mesmo Handle —, e é daí que vêm as três propriedades visíveis da gaveta: sem
// source, valor truncado em 512, sem TRACE.
//
// Não conhece o Wails de propósito: não emite evento, não lê o ctx do App, não
// chama runtime.*. A gaveta puxa as linhas por um método ligado, e com isso o
// transporte não precisa acertar a corrida do ctx que o startup escreve — nem a
// guarda de ctx-ainda-nil, que mataria o processo nas linhas pré-startup.
type logRing struct {
	mu      sync.Mutex
	lines   []string
	pending []byte // sobra de um Write que não terminou em \n.
}

// Write é chamado sob o mutex do console, de qualquer goroutine que logue; a
// leitura vem da thread do método ligado. O mutex daqui é por isso, não por
// ordenação.
//
// Uma linha do console é um Write só, mas o valor de um atributo pode ter \n
// dentro (um erro cru de driver, por exemplo), então a quebra é feita aqui.
func (r *logRing) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pending = append(r.pending, ansiSGR.ReplaceAll(p, nil)...)
	for {
		line, rest, found := bytes.Cut(r.pending, []byte("\n"))
		if !found {
			break
		}
		r.lines = append(r.lines, string(line))
		r.pending = rest
	}
	// Recorte em vez de anel circular: o descarte é raro (o teto é 46x o maior
	// run medido) e o reslice não copia nada.
	r.lines = r.lines[max(0, len(r.lines)-maxLogLines):]

	// O contrato do io.Writer: o io.MultiWriter do console trata n < len(p)
	// como ErrShortWrite e derrubaria a escrita no stderr junto.
	return len(p), nil
}

// Lines devolve uma cópia das linhas do anel. Cópia porque o chamador é a ponte
// do Wails, que serializa fora do mutex enquanto o agente segue logando.
func (r *logRing) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.lines)
}

// ansiSGR casa o escape de cor que o console escreve no nível (levelColor e
// ansiReset, internal/logger/console.go:22-30). Se ele chega até aqui depende de
// o os.Stderr ser TTY — o que varia com a forma de iniciar o app, e daria ao
// mesmo código duas aparências. Então a gaveta tira o escape e colore pelo
// token [INFO ], que tem largura fixa em offset conhecido.
var ansiSGR = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// ---------- API exposta ao frontend ----------

// GetLogLines devolve as linhas que o sink de console escreveu nesta execução —
// as mesmas que o CLI mostra no terminal, da primeira linha do run em diante.
//
// Devolve o anel inteiro a cada chamada, e a gaveta chama a cada 250ms: o teto
// é 2000 linhas por tick, contra as 15 do run medido. Se um dia incomodar, o
// upgrade é um parâmetro de offset aqui — não um transporte novo.
func (a *App) GetLogLines() []string {
	return logs.Lines()
}

// GetLogPath devolve o caminho do arquivo desta execução, para o "copiar o
// caminho" da gaveta: é lá que estão o source, o TRACE e o valor inteiro que o
// console trunca. Vazio quando o sink de arquivo não subiu.
func (a *App) GetLogPath() string {
	return logger.FilePath()
}
