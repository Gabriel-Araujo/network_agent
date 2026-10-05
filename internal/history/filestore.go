package history

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/google/uuid"
)

var log = logger.Named("HISTORY")

// Nomes e permissões usados no disco. O diretório de dados do app
// (".network_agent") é o mesmo de internal/config/constants.go: o histórico vive
// ao lado do config.json, em ~/.network_agent/history.
const (
	appDirName = ".network_agent"
	historyDir = "history"
	fileExt    = ".json"
	tmpSuffix  = ".tmp"
	dirPerm    = 0o700
	filePerm   = 0o600
)

// Parâmetros do retry do rename. O Windows falha com "Acesso negado" quando o
// destino tem um handle aberto — mesmo só de leitura (indexador, antivírus,
// outro processo lendo o arquivo). Algumas tentativas curtas cobrem a janela;
// no pior caso custam 100ms, que é o preço de não perder a escrita.
const (
	renameAttempts = 5
	renameDelay    = 20 * time.Millisecond
)

// DefaultDir devolve o diretório padrão do histórico:
//
//	Windows:     C:\Users\{usuario}\.network_agent\history
//	Linux:       /home/{usuario}/.network_agent/history
//	macOS:       /Users/{usuario}/.network_agent/history
//
// Existe para que nenhum front-end reimplemente a resolução do caminho: a UI e
// o CLI chamam NewFileStore(DefaultDir()). os.UserHomeDir() resolve os três
// sistemas (USERPROFILE no Windows, HOME nos demais); se o home não puder ser
// determinado, cai no diretório temporário, como config.defaultFileLocation.
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "network_agent", historyDir)
	}
	return filepath.Join(home, appDirName, historyDir)
}

// FileStore guarda cada conversa em um arquivo próprio: <dir>/<id>.json, com o
// JSON direto do struct Conversation e sem envelope de versão.
//
// O mutex protege as operações dentro do processo. Entre processos (a UI e o CLI
// abertos ao mesmo tempo) vale last-write-wins, aceito e documentado: cada
// conversa é um arquivo próprio, então dois processos só colidem na mesma
// conversa — e aí a última gravação vence.
type FileStore struct {
	dir string
	mux sync.RWMutex
}

var _ Store = (*FileStore)(nil)

// NewFileStore devolve um Store em arquivo sobre o diretório injetado. Nada é
// criado no disco antes da primeira escrita. O diretório é injetado para o teste
// usar t.TempDir() e para cada front-end decidir o seu; o padrão está em
// DefaultDir().
func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

// List lê todos os arquivos de conversa do diretório, sem índice separado, e
// ordena por updatedAt desc. Diretório que ainda não existe é lista vazia, não
// erro: no primeiro uso o histórico está legitimamente vazio.
//
// Um arquivo ilegível ou corrompido derruba a listagem com ErrInvalid em vez de
// sumir em silêncio: a conversa que desaparece do sidebar sem explicação é
// exatamente a falha silenciosa que este pacote não pode ter.
func (s *FileStore) List() ([]Conversation, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Conversation{}, nil
		}
		return nil, fmt.Errorf("history: falha ao ler o diretório %s: %w", s.dir, err)
	}

	out := make([]Conversation, 0, len(entries))
	for _, e := range entries {
		// Só <id>.json: diretórios, arquivos temporários e intrusos (.DS_Store,
		// rascunho.txt) ficam de fora.
		if e.IsDir() || filepath.Ext(e.Name()) != fileExt {
			continue
		}
		c, err := s.readFile(e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}

	slices.SortFunc(out, byUpdatedAt)
	return out, nil
}

// Get devolve a conversa com o transcript.
func (s *FileStore) Get(id string) (Conversation, error) {
	if err := validID(id); err != nil {
		return Conversation{}, err
	}

	s.mux.RLock()
	defer s.mux.RUnlock()

	return s.readFile(id + fileExt)
}

// Create cunha id, título e timestamps e grava a conversa vazia.
func (s *FileStore) Create(nc NewConversation) (Conversation, error) {
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

	if err := s.write(&c); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

// Append acrescenta mensagens ao fim do transcript. Conversa inexistente devolve
// ErrNotFound e não cria nada — a criação é explícita, via Create.
//
// O Store não valida o conteúdo das mensagens; a única regra é o guardrail do
// status transiente, aplicado na escrita.
func (s *FileStore) Append(id string, msgs ...Message) (Conversation, error) {
	if err := validID(id); err != nil {
		return Conversation{}, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	c, err := s.readFile(id + fileExt)
	if err != nil {
		return Conversation{}, err
	}
	if len(msgs) == 0 {
		return c, nil
	}

	c.Messages = append(c.Messages, msgs...)
	c.UpdatedAt = nowStamp()
	if err := s.write(&c); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

// Rename troca o título. O updatedAt acompanha a mudança: a lista ordena por ele,
// então a conversa renomeada sobe para o topo.
func (s *FileStore) Rename(id, title string) (Conversation, error) {
	if err := validID(id); err != nil {
		return Conversation{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Conversation{}, fmt.Errorf("%w: o título da conversa é obrigatório", ErrInvalid)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	c, err := s.readFile(id + fileExt)
	if err != nil {
		return Conversation{}, err
	}

	c.Title = title
	c.UpdatedAt = nowStamp()
	if err := s.write(&c); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

// Delete apaga o arquivo da conversa. Id inexistente devolve ErrNotFound.
func (s *FileStore) Delete(id string) error {
	if err := validID(id); err != nil {
		return err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	path := s.path(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: conversa %s", ErrNotFound, id)
		}
		return fmt.Errorf("history: falha ao apagar %s: %w", path, err)
	}
	return nil
}

// ---------- disco ----------

// path devolve o arquivo de uma conversa dentro do diretório do Store.
func (s *FileStore) path(id string) string {
	return filepath.Join(s.dir, id+fileExt)
}

// readFile lê e normaliza uma conversa pelo nome do arquivo. O chamador segura o
// mutex.
func (s *FileStore) readFile(nome string) (Conversation, error) {
	path := filepath.Join(s.dir, nome)

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			id := strings.TrimSuffix(nome, fileExt)
			return Conversation{}, fmt.Errorf("%w: conversa %s", ErrNotFound, id)
		}
		return Conversation{}, fmt.Errorf("history: falha ao ler %s: %w", path, err)
	}

	var c Conversation
	if err := json.Unmarshal(raw, &c); err != nil {
		return Conversation{}, fmt.Errorf("%w: %s: %s", ErrInvalid, path, err)
	}

	normalize(&c)
	return c, nil
}

// write grava a conversa de forma atômica (tmp + rename) e normaliza o valor
// recebido, que é devolvido ao chamador exatamente como foi para o disco. A
// normalização mora aqui porque este é o único ponto de escrita: nenhum caminho
// novo esquece o guardrail. O chamador segura o mutex.
func (s *FileStore) write(c *Conversation) error {
	normalize(c)

	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("history: falha ao serializar a conversa %s: %w", c.ID, err)
	}

	// O diretório pode não existir ainda: o histórico nasce na primeira gravação.
	if err := os.MkdirAll(s.dir, dirPerm); err != nil {
		return fmt.Errorf("history: falha ao criar o diretório %s: %w", s.dir, err)
	}

	// O tmp leva o id no nome para que duas conversas gravadas em paralelo (ou
	// por processos diferentes) nunca disputem o mesmo arquivo temporário.
	path := s.path(c.ID)
	tmp := path + tmpSuffix

	if err := os.WriteFile(tmp, raw, filePerm); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("history: falha ao escrever %s: %w", tmp, err)
	}
	if err := renameWithRetry(tmp, path); err != nil {
		// Sem isto um tmp órfão ficaria no diretório do usuário a cada falha.
		os.Remove(tmp)
		return fmt.Errorf("history: falha ao gravar %s: %w", path, err)
	}

	log.Debugf("Conversa %s gravada em %s", c.ID, path)
	return nil
}

// renameWithRetry renomeia o tmp sobre o destino, insistindo algumas vezes. O
// Windows recusa o rename enquanto o destino tiver um handle aberto — antivírus,
// indexador ou outro processo lendo o arquivo bastam —, e desistir na primeira
// negativa seria a falha de escrita silenciosa que o critério do projeto proíbe.
func renameWithRetry(tmp, dst string) error {
	var err error
	for i := 0; i < renameAttempts; i++ {
		if err = os.Rename(tmp, dst); err == nil {
			return nil
		}
		if i < renameAttempts-1 {
			log.Warnf("rename de %s para %s falhou (tentativa %d/%d): %v", tmp, dst, i+1, renameAttempts, err)
			time.Sleep(renameDelay)
		}
	}
	return err
}

// byUpdatedAt ordena da conversa mais recente para a mais antiga. O id desempata
// quando os timestamps são iguais (duas escritas no mesmo milissegundo), para que
// a lista não mude de ordem entre duas chamadas. Um timestamp fora do formato
// esperado cai na comparação textual em vez de perder a ordenação.
func byUpdatedAt(a, b Conversation) int {
	ta, okA := parseStamp(a.UpdatedAt)
	tb, okB := parseStamp(b.UpdatedAt)
	if !okA || !okB {
		return strings.Compare(b.UpdatedAt, a.UpdatedAt)
	}
	if c := tb.Compare(ta); c != 0 {
		return c
	}
	return strings.Compare(b.ID, a.ID)
}
