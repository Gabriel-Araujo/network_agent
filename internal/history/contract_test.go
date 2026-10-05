package history

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestContract roda a mesma bateria contra toda implementação de Store. Hoje são
// duas: o FileStore em um diretório temporário e o memStore em memória.
//
// É o entregável que prova a genericidade do contrato — é este mesmo teste que o
// CLI vai rodar contra a sua plugagem, sem reescrever nada. Se ele passar com o
// FileStore e falhar com a implementação em memória, o contrato vazou para
// dentro da implementação em arquivo.
func TestContract(t *testing.T) {
	impls := []struct {
		nome     string
		newStore func(t *testing.T) Store
	}{
		{
			nome:     "FileStore",
			newStore: func(t *testing.T) Store { return NewFileStore(t.TempDir()) },
		},
		{
			nome:     "MemStore",
			newStore: func(t *testing.T) Store { return newMemStore() },
		},
	}

	for _, impl := range impls {
		t.Run(impl.nome, func(t *testing.T) {
			testStoreContract(t, impl.newStore)
		})
	}
}

// testStoreContract é a bateria. Cada caso pede um Store novo, para não herdar
// estado do caso anterior.
func testStoreContract(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("CreateCunhaIDETimestamps", func(t *testing.T) {
		s := newStore(t)

		got, err := s.Create(NewConversation{
			Title:    "  Interface do agente de IA  ",
			Provider: "OpenAI Compatible",
			Model:    "qwen3",
		})
		if err != nil {
			t.Fatalf("Create() erro = %v", err)
		}

		if _, err := uuid.Parse(got.ID); err != nil {
			t.Errorf("Create() id = %q, want UUID v4: %v", got.ID, err)
		}
		if got.Title != "Interface do agente de IA" {
			t.Errorf("Create() title = %q, want sem espaços nas pontas", got.Title)
		}
		if got.Provider != "OpenAI Compatible" || got.Model != "qwen3" {
			t.Errorf("Create() provider/model = %q/%q, want %q/%q", got.Provider, got.Model, "OpenAI Compatible", "qwen3")
		}
		if _, err := time.Parse(time.RFC3339, got.CreatedAt); err != nil {
			t.Errorf("Create() createdAt = %q, não é RFC3339: %v", got.CreatedAt, err)
		}
		if _, err := time.Parse(time.RFC3339, got.UpdatedAt); err != nil {
			t.Errorf("Create() updatedAt = %q, não é RFC3339: %v", got.UpdatedAt, err)
		}
		if got.UpdatedAt < got.CreatedAt {
			t.Errorf("Create() updatedAt = %q, antes de createdAt = %q", got.UpdatedAt, got.CreatedAt)
		}
		if got.Messages == nil {
			t.Error("Create() messages = nil, want lista vazia (o JSON do Wails vira null no TS)")
		}
		if len(got.Messages) != 0 {
			t.Errorf("Create() messages = %v, want vazia", got.Messages)
		}

		// Get devolve o mesmo registro.
		back, err := s.Get(got.ID)
		if err != nil {
			t.Fatalf("Get() erro = %v", err)
		}
		if !reflect.DeepEqual(got, back) {
			t.Errorf("Get() = %+v, want %+v", back, got)
		}

		// Conversas distintas não compartilham id.
		outra := mustCreate(t, s, "Outra conversa")
		if outra.ID == got.ID {
			t.Errorf("Create() repetiu o id %q", outra.ID)
		}
	})

	t.Run("CreateRejeitaTituloVazio", func(t *testing.T) {
		s := newStore(t)

		for _, title := range []string{"", "   ", "\t\n"} {
			t.Run(fmt.Sprintf("%q", title), func(t *testing.T) {
				if _, err := s.Create(NewConversation{Title: title}); !errors.Is(err, ErrInvalid) {
					t.Errorf("Create(title=%q) erro = %v, want ErrInvalid", title, err)
				}
			})
		}
	})

	t.Run("ListaVazia", func(t *testing.T) {
		s := newStore(t)

		got, err := s.List()
		if err != nil {
			t.Fatalf("List() erro = %v", err)
		}
		if got == nil {
			t.Error("List() = nil, want lista vazia")
		}
		if len(got) != 0 {
			t.Errorf("List() = %v, want vazia", got)
		}
	})

	t.Run("ListOrdenaPorUpdatedAtDesc", func(t *testing.T) {
		s := newStore(t)

		primeira := mustCreate(t, s, "Primeira")
		sleepStamp()
		segunda := mustCreate(t, s, "Segunda")
		sleepStamp()
		terceira := mustCreate(t, s, "Terceira")

		assertOrder(t, s, terceira.ID, segunda.ID, primeira.ID)

		// A ordenação é por updatedAt, não por createdAt: um append na mais
		// antiga a traz para o topo.
		sleepStamp()
		if _, err := s.Append(primeira.ID, userMessage("Voltei")); err != nil {
			t.Fatalf("Append() erro = %v", err)
		}
		assertOrder(t, s, primeira.ID, terceira.ID, segunda.ID)
	})

	t.Run("GetInexistente", func(t *testing.T) {
		s := newStore(t)

		_, err := s.Get(uuid.NewString())
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() erro = %v, want ErrNotFound", err)
		}
	})

	t.Run("AppendAcrescentaMensagens", func(t *testing.T) {
		s := newStore(t)
		c := mustCreate(t, s, "Transcript")

		msgs := []Message{userMessage("Olá"), userMessage("Tudo bem?")}
		got, err := s.Append(c.ID, msgs...)
		if err != nil {
			t.Fatalf("Append() erro = %v", err)
		}
		if len(got.Messages) != len(msgs) {
			t.Fatalf("Append() messages = %d, want %d", len(got.Messages), len(msgs))
		}
		for i, want := range msgs {
			if !reflect.DeepEqual(got.Messages[i], want) {
				t.Errorf("Append() messages[%d] = %+v, want %+v", i, got.Messages[i], want)
			}
		}

		// Um segundo append acumula, e List devolve a conversa com o transcript
		// junto (não existe índice separado).
		if _, err := s.Append(c.ID, userMessage("Ainda aqui")); err != nil {
			t.Fatalf("Append() erro = %v", err)
		}
		list, err := s.List()
		if err != nil {
			t.Fatalf("List() erro = %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("List() = %d conversas, want 1", len(list))
		}
		if len(list[0].Messages) != 3 {
			t.Errorf("List() messages = %d, want 3", len(list[0].Messages))
		}
	})

	t.Run("AppendSemMensagensNaoMudaNada", func(t *testing.T) {
		s := newStore(t)
		c := mustCreate(t, s, "Nada a acrescentar")

		got, err := s.Append(c.ID)
		if err != nil {
			t.Fatalf("Append() erro = %v", err)
		}
		if !reflect.DeepEqual(c, got) {
			t.Errorf("Append() sem mensagens = %+v, want a conversa intacta %+v", got, c)
		}
	})

	t.Run("AppendEmConversaInexistente", func(t *testing.T) {
		s := newStore(t)
		existente := mustCreate(t, s, "Existe")

		id := uuid.NewString()
		_, err := s.Append(id, userMessage("Órfã"))
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Append() erro = %v, want ErrNotFound", err)
		}

		// E não criou nada: a criação é explícita, via Create.
		if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() após Append órfão erro = %v, want ErrNotFound", err)
		}
		list, err := s.List()
		if err != nil {
			t.Fatalf("List() erro = %v", err)
		}
		if len(list) != 1 || list[0].ID != existente.ID {
			t.Errorf("List() = %+v, want só a conversa existente", list)
		}
	})

	t.Run("Rename", func(t *testing.T) {
		s := newStore(t)
		c := mustCreate(t, s, "Título antigo")

		got, err := s.Rename(c.ID, "  Título novo  ")
		if err != nil {
			t.Fatalf("Rename() erro = %v", err)
		}
		if got.Title != "Título novo" {
			t.Errorf("Rename() title = %q, want %q", got.Title, "Título novo")
		}
		if got.CreatedAt != c.CreatedAt {
			t.Errorf("Rename() createdAt = %q, want %q (renomear não recunha a criação)", got.CreatedAt, c.CreatedAt)
		}

		back, err := s.Get(c.ID)
		if err != nil {
			t.Fatalf("Get() erro = %v", err)
		}
		if back.Title != "Título novo" {
			t.Errorf("Get() após Rename title = %q, want %q", back.Title, "Título novo")
		}

		if _, err := s.Rename(uuid.NewString(), "Fantasma"); !errors.Is(err, ErrNotFound) {
			t.Errorf("Rename() de id inexistente erro = %v, want ErrNotFound", err)
		}
		if _, err := s.Rename(c.ID, "   "); !errors.Is(err, ErrInvalid) {
			t.Errorf("Rename() com título vazio erro = %v, want ErrInvalid", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		s := newStore(t)
		ficando := mustCreate(t, s, "Fica")
		saindo := mustCreate(t, s, "Sai")

		if err := s.Delete(saindo.ID); err != nil {
			t.Fatalf("Delete() erro = %v", err)
		}
		if _, err := s.Get(saindo.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() após Delete erro = %v, want ErrNotFound", err)
		}

		list, err := s.List()
		if err != nil {
			t.Fatalf("List() erro = %v", err)
		}
		if len(list) != 1 || list[0].ID != ficando.ID {
			t.Errorf("List() = %+v, want só %s", list, ficando.ID)
		}

		if err := s.Delete(saindo.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("Delete() repetido erro = %v, want ErrNotFound", err)
		}
	})

	t.Run("GuardrailStatusTransiente", func(t *testing.T) {
		s := newStore(t)
		c := mustCreate(t, s, "Turno interrompido")

		// O front-end manda a mensagem ainda como streaming; o disco não pode
		// recebê-la assim, senão o turno interrompido volta como spinner eterno.
		streaming := Message{
			ID:      uuid.NewString(),
			Role:    RoleAgent,
			Content: "Configurando o FRR pela meta",
			Status:  StatusStreaming,
		}
		got, err := s.Append(c.ID, streaming)
		if err != nil {
			t.Fatalf("Append() erro = %v", err)
		}
		if got.Messages[0].Status != StatusDone {
			t.Errorf("Append() status = %q, want %q (guardrail do status transiente)", got.Messages[0].Status, StatusDone)
		}

		back, err := s.Get(c.ID)
		if err != nil {
			t.Fatalf("Get() erro = %v", err)
		}
		if back.Messages[0].Status != StatusDone {
			t.Errorf("Get() status = %q, want %q (nada de streaming vindo do disco)", back.Messages[0].Status, StatusDone)
		}

		// Os status terminais continuam como o front-end mandou.
		for _, st := range []Status{StatusDone, StatusError} {
			if _, err := s.Append(c.ID, Message{ID: uuid.NewString(), Role: RoleAgent, Content: "x", Status: st}); err != nil {
				t.Fatalf("Append() erro = %v", err)
			}
		}
		back, err = s.Get(c.ID)
		if err != nil {
			t.Fatalf("Get() erro = %v", err)
		}
		if back.Messages[1].Status != StatusDone || back.Messages[2].Status != StatusError {
			t.Errorf("status terminais viraram %q/%q, want done/error", back.Messages[1].Status, back.Messages[2].Status)
		}
	})

	t.Run("IdaEVoltaCompletaDoMessage", func(t *testing.T) {
		s := newStore(t)
		c := mustCreate(t, s, "Round-trip")

		// Uma mensagem com todos os campos preenchidos: nada pode ficar pelo
		// caminho entre o front-end e o disco.
		sucesso := Message{
			ID:        uuid.NewString(),
			Role:      RoleAgent,
			Content:   "Configurei o FRR e validei as sessões BGP.",
			Time:      "2026-09-21T14:32:05.417Z",
			Status:    StatusDone,
			ElapsedMs: 8421,
			Tokens:    512,
			Usage: &TokenUsage{
				Input:        400,
				Output:       112,
				Total:        512,
				ContextUsed:  512,
				ContextTotal: 128000,
			},
			ToolCalls: []ToolCall{
				{ID: "call-1", Action: "run_command", Label: "vtysh -c 'show bgp summary'", Result: "BGP router identifier 10.0.0.1"},
				{ID: "call-2", Action: "read_file", Label: "frr.conf", Result: "router bgp 65000"},
			},
		}
		falha := Message{
			ID:        uuid.NewString(),
			Role:      RoleAgent,
			Content:   "",
			Time:      "2026-09-21T14:33:10.002Z",
			Status:    StatusError,
			ElapsedMs: 1500,
			Error: &AgentError{
				What: "Não consegui falar com o roteador",
				Why:  "A sessão SSH caiu durante a configuração",
				Next: "Verifique se o roteador está acessível e tente de novo",
			},
		}

		if _, err := s.Append(c.ID, sucesso, falha); err != nil {
			t.Fatalf("Append() erro = %v", err)
		}

		got, err := s.Get(c.ID)
		if err != nil {
			t.Fatalf("Get() erro = %v", err)
		}
		if len(got.Messages) != 2 {
			t.Fatalf("Get() messages = %d, want 2", len(got.Messages))
		}
		if !reflect.DeepEqual(got.Messages[0], sucesso) {
			t.Errorf("mensagem com usage/toolCalls voltou diferente:\n got = %+v\nwant = %+v", got.Messages[0], sucesso)
		}
		if !reflect.DeepEqual(got.Messages[1], falha) {
			t.Errorf("mensagem com error voltou diferente:\n got = %+v\nwant = %+v", got.Messages[1], falha)
		}
	})
}

// ---------- implementação em memória ----------

// memStore é a segunda implementação de Store: um mapa em memória, escrito só
// para o teste de contrato. Ele existe para provar que o contrato não depende do
// disco — se ele não passar na bateria, o contrato está vazado.
type memStore struct {
	mux  sync.RWMutex
	data map[string]Conversation
}

var _ Store = (*memStore)(nil)

func newMemStore() *memStore {
	return &memStore{data: map[string]Conversation{}}
}

func (s *memStore) List() ([]Conversation, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	out := make([]Conversation, 0, len(s.data))
	for _, c := range s.data {
		out = append(out, cloneConversation(c))
	}
	slices.SortFunc(out, func(a, b Conversation) int {
		return strings.Compare(b.UpdatedAt, a.UpdatedAt)
	})
	return out, nil
}

func (s *memStore) Create(nc NewConversation) (Conversation, error) {
	title := strings.TrimSpace(nc.Title)
	if title == "" {
		return Conversation{}, fmt.Errorf("%w: o título da conversa é obrigatório", ErrInvalid)
	}

	now := nowStamp()
	c := Conversation{
		ID:        uuid.NewString(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
		Provider:  nc.Provider,
		Model:     nc.Model,
		Messages:  []Message{},
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.save(&c)
	return cloneConversation(c), nil
}

func (s *memStore) Get(id string) (Conversation, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.get(id)
}

func (s *memStore) Append(id string, msgs ...Message) (Conversation, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	c, err := s.get(id)
	if err != nil {
		return Conversation{}, err
	}
	if len(msgs) == 0 {
		return c, nil
	}

	c.Messages = append(c.Messages, msgs...)
	c.UpdatedAt = nowStamp()
	s.save(&c)
	return cloneConversation(c), nil
}

func (s *memStore) Rename(id, title string) (Conversation, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Conversation{}, fmt.Errorf("%w: o título da conversa é obrigatório", ErrInvalid)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	c, err := s.get(id)
	if err != nil {
		return Conversation{}, err
	}

	c.Title = title
	c.UpdatedAt = nowStamp()
	s.save(&c)
	return cloneConversation(c), nil
}

func (s *memStore) Delete(id string) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.data[id]; !ok {
		return fmt.Errorf("%w: conversa %s", ErrNotFound, id)
	}
	delete(s.data, id)
	return nil
}

// get busca sem tocar o mutex. O chamador segura o lock.
func (s *memStore) get(id string) (Conversation, error) {
	c, ok := s.data[id]
	if !ok {
		return Conversation{}, fmt.Errorf("%w: conversa %s", ErrNotFound, id)
	}
	return cloneConversation(c), nil
}

// save normaliza e guarda uma cópia. O chamador segura o lock. A normalização é
// a mesma do FileStore, de propósito: o guardrail do status transiente é regra do
// contrato, não da implementação em arquivo.
func (s *memStore) save(c *Conversation) {
	normalize(c)
	s.data[c.ID] = cloneConversation(*c)
}

// cloneConversation devolve uma cópia profunda: sem ela, um Get seguido de
// mutação no chamador mexeria no estado guardado — o FileStore não tem esse
// problema porque cada leitura vem do disco.
func cloneConversation(c Conversation) Conversation {
	out := c
	out.Messages = make([]Message, len(c.Messages))
	for i, m := range c.Messages {
		out.Messages[i] = cloneMessage(m)
	}
	return out
}

func cloneMessage(m Message) Message {
	out := m
	if m.Usage != nil {
		usage := *m.Usage
		out.Usage = &usage
	}
	if m.Error != nil {
		agentErr := *m.Error
		out.Error = &agentErr
	}
	if m.ToolCalls != nil {
		out.ToolCalls = append([]ToolCall(nil), m.ToolCalls...)
	}
	return out
}

// ---------- utilidades do teste ----------

// mustCreate cria uma conversa e derruba o teste se o Store recusar.
func mustCreate(t *testing.T, s Store, title string) Conversation {
	t.Helper()

	c, err := s.Create(NewConversation{Title: title, Provider: "OpenAI Compatible", Model: "qwen3"})
	if err != nil {
		t.Fatalf("Create(%q) erro = %v", title, err)
	}
	return c
}

// userMessage monta uma mensagem simples do usuário.
func userMessage(content string) Message {
	return Message{
		ID:      uuid.NewString(),
		Role:    RoleUser,
		Content: content,
		Time:    nowStamp(),
		Status:  StatusDone,
	}
}

// sleepStamp separa dois timestamps: eles têm resolução de milissegundo, e a
// bateria de ordenação precisa de updatedAt distintos.
func sleepStamp() {
	time.Sleep(5 * time.Millisecond)
}

// assertOrder exige que List devolva exatamente estes ids, nesta ordem, com os
// updatedAt de fato decrescentes.
func assertOrder(t *testing.T, s Store, wantIDs ...string) {
	t.Helper()

	got, err := s.List()
	if err != nil {
		t.Fatalf("List() erro = %v", err)
	}

	ids := make([]string, 0, len(got))
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Errorf("List() ids = %v, want %v", ids, wantIDs)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].UpdatedAt < got[i].UpdatedAt {
			t.Errorf("List() fora de ordem: %s (%s) veio antes de %s (%s)",
				got[i-1].ID, got[i-1].UpdatedAt, got[i].ID, got[i].UpdatedAt)
		}
	}
}
