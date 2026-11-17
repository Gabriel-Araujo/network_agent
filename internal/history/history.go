// Package history é o núcleo genérico do histórico de conversas do aplicativo:
// os tipos trocados com o front-end, o contrato Store e a implementação em
// arquivo (FileStore).
//
// O pacote é genérico de propósito. A UI (cmd/ui) e o CLI (cmd/cli) consomem o
// mesmo Store sem reescrever nada, e quem sustenta isso é o teste de contrato
// em contract_test.go: a mesma bateria roda contra o FileStore e contra uma
// implementação em memória.
//
// Forma em disco: um arquivo por conversa, <dir>/<id>.json, com o JSON direto
// do struct Conversation — sem envelope de versão, por decisão do ticket 01. A
// escrita é atômica (tmp + rename) e o diretório é injetado no construtor
// (NewFileStore), com DefaultDir() resolvendo ~/.network_agent/history.
//
// Concorrência: dentro do processo, um sync.RWMutex no FileStore serializa as
// operações. Entre processos (a UI e o CLI abertos ao mesmo tempo) vale
// last-write-wins, aceito e documentado: cada conversa é um arquivo próprio, e
// dois processos só colidem quando escrevem a mesma conversa.
package history

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotFound indica que a conversa não existe no Store. Get, Append, Rename e
// Delete devolvem esta sentinela em vez de um zero value silencioso: "a conversa
// sumiu do disco" e "a conversa está vazia" precisam ser distinguíveis na UI.
var ErrNotFound = errors.New("history: conversa não encontrada")

// ErrInvalid indica que a conversa existe mas não pôde ser interpretada, ou que
// os dados recebidos não servem para gravar (id que não é nome de arquivo
// seguro, título vazio).
var ErrInvalid = errors.New("history: conversa inválida")

// ---------- tipos ----------
//
// Espelham cmd/ui/frontend/src/types.ts. As tags JSON são camelCase iguais aos
// nomes do TypeScript porque o binding do Wails gera o TS a partir destes
// structs e o arquivo em disco precisa bater com o que o front-end já usa.

// Role é quem produziu a mensagem.
type Role string

const (
	RoleUser   Role = "user"
	RoleAgent  Role = "agent"
	RoleRepair Role = "mermaid_repair"
)

// Status é o estado de uma mensagem. StatusStreaming é transiente e nunca chega
// ao disco: o Store o normaliza para StatusDone, tanto na escrita quanto na
// leitura, para que um turno interrompido não reapareça como spinner eterno ao
// reabrir a conversa.
type Status string

const (
	StatusStreaming Status = "streaming"
	StatusDone      Status = "done"
	StatusError     Status = "error"
)

// ToolCall é uma chamada de ferramenta que acompanha a mensagem do agente.
type ToolCall struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Label  string `json:"label"`
	Result string `json:"result"`
}

// AgentError é o erro de um turno já explicado para o usuário: o quê, por quê e
// o que fazer a seguir.
type AgentError struct {
	What string `json:"what"`
	Why  string `json:"why"`
	Next string `json:"next"`
}

// TokenUsage é o consumo de tokens do turno.
type TokenUsage struct {
	Input        int `json:"input"`
	Output       int `json:"output"`
	Total        int `json:"total"`
	ContextUsed  int `json:"contextUsed"`
	ContextTotal int `json:"contextTotal"`
	// ContextSource diz de onde veio a janela (ContextTotal): "provider" quando
	// foi lida do endpoint do provedor, "config" quando é o context_size
	// declarado na configuração. O medidor da UI exibe a marca — número
	// declarado não se passa por medido — e conversas gravadas antes desta
	// decisão vêm sem o campo: o número delas era o declarado.
	ContextSource string `json:"contextSource,omitempty"`
}

// Message é uma mensagem da conversa. Vai para o disco inteira — inclusive
// usage, toolCalls e error: nada do que a UI mostra é derivado na leitura.
type Message struct {
	ID        string      `json:"id"`
	Role      Role        `json:"role"`
	Content   string      `json:"content"`
	Time      string      `json:"time,omitempty"`
	Status    Status      `json:"status"`
	ElapsedMs int         `json:"elapsedMs,omitempty"`
	Tokens    int         `json:"tokens,omitempty"`
	Usage     *TokenUsage `json:"usage,omitempty"`
	ToolCalls []ToolCall  `json:"toolCalls,omitempty"`
	Error     *AgentError `json:"error,omitempty"`

	// Subagent é quem respondeu o turno, e é um dos quatro em toda resposta
	// gravada daqui para frente: o ponto de entrada não responde turno
	// nenhum, então classificação vazia, que falhou, ou que nomeou
	// Subagente cuja configuração não carregou, respondem com o default e
	// gravam o nome dele (turn.Outcome.Subagent). É quem DE FATO respondeu,
	// não quem a classificação nomeou. Conversa gravada antes desta decisão
	// vem sem o campo — o turno dela foi do genérico, e o vazio legado fica
	// como está.
	Subagent string `json:"subagent,omitempty"`

	// Warnings são os avisos de degradação do turno, na ordem em que
	// saíram (turn.Outcome.Warnings): classificação que falhou, retrieval
	// que não veio, Subagente que não carregou. Ficam na mensagem porque o
	// Store é append-only — aviso não gravado junto da resposta se perde —
	// e porque é a resposta degradada que eles explicam.
	Warnings []string `json:"warnings,omitempty"`

	MermaidRepair *MermaidRepair `json:"mermaidRepair,omitempty"`
}

// Conversation é uma conversa completa, com o transcript.
//
// CreatedAt e UpdatedAt são strings RFC3339 em UTC, não time.Time: no binding do
// Wails v2.15 um time.Time vira any no TypeScript, e o front-end perderia o
// tipo. O formato gravado está em timeLayout.
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
}

// NewConversation são os dados que o front-end conhece ao criar uma conversa. O
// id e os timestamps são cunhados pelo Store.
//
// Title é obrigatório: um título só de espaços devolve ErrInvalid, para que a
// conversa sem nome não chegue em silêncio à lista do sidebar.
type NewConversation struct {
	Title    string
	Provider string
	Model    string
}

// Store é o contrato do histórico. Toda implementação — o FileStore em arquivo,
// a implementação em memória do teste de contrato, uma futura para o CLI —
// responde a estes seis métodos e nada mais.
type Store interface {
	// List devolve todas as conversas, da mais recente para a mais antiga
	// (updatedAt desc), com o transcript junto: não existe índice separado, o
	// Store lê todos os arquivos do diretório. Sem teto de conversas, e nunca
	// nil — lista vazia é []Conversation{}, para o JSON não virar null no TS.
	List() ([]Conversation, error)

	// Create cunha o id (UUID v4), o título e os timestamps, e grava a conversa
	// vazia. É o único caminho que cria: Append em conversa inexistente devolve
	// ErrNotFound.
	Create(nc NewConversation) (Conversation, error)

	// Get devolve a conversa com o transcript. Id inexistente devolve
	// ErrNotFound; arquivo ilegível devolve ErrInvalid.
	Get(id string) (Conversation, error)

	// Append acrescenta mensagens ao fim do transcript e devolve a conversa
	// atualizada. Conversa inexistente devolve ErrNotFound e não cria nada.
	Append(id string, msgs ...Message) (Conversation, error)

	// Rename troca o título e devolve a conversa atualizada.
	Rename(id, title string) (Conversation, error)

	// Delete apaga a conversa. Id inexistente devolve ErrNotFound.
	Delete(id string) error
}

// ---------- formato dos timestamps ----------

// timeLayout é o formato dos timestamps gravados no disco: RFC3339 em UTC com
// milissegundos. Largura fixa e sempre em UTC fazem a ordem lexicográfica bater
// com a cronológica, e o JavaScript (new Date(...)) interpreta o formato sem
// ressalva. Timestamps de arquivos antigos ou editados à mão são aceitos também
// na forma sem fração (time.RFC3339) — ver parseStamp.
const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// nowStamp devolve o instante atual no formato gravado.
func nowStamp() string {
	return time.Now().UTC().Format(timeLayout)
}

// NowStamp devolve o instante atual no formato gravado (RFC3339 UTC com
// milissegundos), para quem cunha mensagens por fora do Store — a UI, ao gravar
// um turno. É o mesmo formato de Message.Time, CreatedAt e UpdatedAt; o layout
// não deve ser duplicado lá fora.
func NowStamp() string {
	return nowStamp()
}

// parseStamp interpreta um timestamp gravado, tolerando a forma sem fração de
// segundo. O segundo retorno diz se a interpretação deu certo.
func parseStamp(s string) (time.Time, bool) {
	if t, err := time.Parse(timeLayout, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// ---------- guardrails compartilhados ----------

// normalize aplica o guardrail do status transiente e garante slices não-nulos.
// Roda em toda leitura e em toda escrita: é o que impede um turno interrompido
// de voltar do disco como spinner eterno.
func normalize(c *Conversation) {
	if c.Messages == nil {
		c.Messages = []Message{}
	}
	for i := range c.Messages {
		normalizeMessage(&c.Messages[i])
	}
}

// normalizeMessage troca o status transiente por um terminal. O disco só vê
// estado terminal: uma mensagem interrompida mostra o que chegou do stream, e
// não um "error" que ninguém produziu (o AgentError viria vazio).
func normalizeMessage(m *Message) {
	if m.Status == StatusStreaming || m.Status == "" {
		m.Status = StatusDone
	}
}

// validID recusa id que não seja nome de arquivo seguro: a conversa vira
// <dir>/<id>.json, então um id com separador de caminho ou ".." escreveria fora
// do diretório do histórico.
func validID(id string) error {
	switch {
	case id == "":
		return fmt.Errorf("%w: o id da conversa é obrigatório", ErrInvalid)
	case id == "." || id == "..":
		return fmt.Errorf("%w: id de conversa inseguro: %q", ErrInvalid, id)
	case strings.ContainsAny(id, `/\:*?"<>|`):
		return fmt.Errorf("%w: id de conversa inseguro: %q", ErrInvalid, id)
	case strings.HasSuffix(id, ".") || strings.HasSuffix(id, " "):
		// O Windows descarta ponto e espaço no fim do nome, o que faria dois
		// ids distintos apontarem para o mesmo arquivo.
		return fmt.Errorf("%w: id de conversa inseguro: %q", ErrInvalid, id)
	}
	return nil
}
