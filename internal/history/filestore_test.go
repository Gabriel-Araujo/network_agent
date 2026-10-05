package history

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// Os testes deste arquivo cobrem o que só existe no FileStore: a forma em disco,
// a atomicidade da escrita e os cuidados de caminho. O contrato em si — o que
// toda implementação de Store precisa cumprir — está em contract_test.go.

func TestFileStoreUmArquivoPorConversa(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	c := mustCreate(t, s, "Interface do agente de IA")

	path := filepath.Join(dir, c.ID+fileExt)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("esperava a conversa em %s: %v", path, err)
	}

	// JSON direto do struct: sem envelope de versão, com as chaves que o
	// front-end já usa (camelCase do types.ts).
	var campos map[string]any
	if err := json.Unmarshal(raw, &campos); err != nil {
		t.Fatalf("arquivo não é JSON: %v\n%s", err, raw)
	}
	for _, chave := range []string{"id", "title", "createdAt", "updatedAt", "provider", "model", "messages"} {
		if _, ok := campos[chave]; !ok {
			t.Errorf("faltou a chave %q no arquivo: %s", chave, raw)
		}
	}
	if _, ok := campos["version"]; ok {
		t.Errorf("arquivo tem envelope de versão, e o ticket 01 decidiu por JSON direto: %s", raw)
	}

	// Cada conversa tem o seu arquivo.
	mustCreate(t, s, "Segunda conversa")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() erro = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("diretório tem %d arquivos, want 2 (um por conversa)", len(entries))
	}
}

func TestFileStoreEscritaAtomica(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	c := mustCreate(t, s, "Atomicidade")

	for i := 0; i < 3; i++ {
		if _, err := s.Append(c.ID, userMessage("turno")); err != nil {
			t.Fatalf("Append() erro = %v", err)
		}
	}
	if _, err := s.Rename(c.ID, "Atomicidade renomeada"); err != nil {
		t.Fatalf("Rename() erro = %v", err)
	}
	if err := s.Delete(c.ID); err != nil {
		t.Fatalf("Delete() erro = %v", err)
	}

	assertSemTmp(t, dir)
}

func TestFileStoreLimpaTmpQuandoRenameFalha(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	id := uuid.NewString()

	// Um diretório com o nome do arquivo de destino faz o rename falhar: é a
	// falha de escrita mais fácil de provocar de propósito.
	if err := os.Mkdir(filepath.Join(dir, id+fileExt), dirPerm); err != nil {
		t.Fatalf("Mkdir() erro = %v", err)
	}

	c := Conversation{
		ID:        id,
		Title:     "Falha",
		CreatedAt: nowStamp(),
		UpdatedAt: nowStamp(),
		Messages:  []Message{},
	}
	if err := s.write(&c); err == nil {
		t.Fatal("write() = nil, want erro: o destino é um diretório")
	}

	// A falha não pode deixar lixo no diretório do usuário.
	assertSemTmp(t, dir)
}

func TestFileStoreNormalizaStatusLidoDoDisco(t *testing.T) {
	dir := t.TempDir()
	id := uuid.NewString()

	// Arquivo gravado por fora do Store (versão anterior do app, edição à mão)
	// com status transiente: a leitura não pode entregar "streaming" à UI.
	raw := []byte(`{"id":"` + id + `","title":"Antiga","createdAt":"2026-09-21T10:00:00.000Z",` +
		`"updatedAt":"2026-09-21T10:00:00.000Z","provider":"","model":"",` +
		`"messages":[{"id":"m1","role":"agent","content":"parcial","status":"streaming"}]}`)
	if err := os.WriteFile(filepath.Join(dir, id+fileExt), raw, filePerm); err != nil {
		t.Fatalf("WriteFile() erro = %v", err)
	}

	s := NewFileStore(dir)

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get() erro = %v", err)
	}
	if got.Messages[0].Status != StatusDone {
		t.Errorf("Get() status = %q, want %q (guardrail na leitura também)", got.Messages[0].Status, StatusDone)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List() erro = %v", err)
	}
	if len(list) != 1 || list[0].Messages[0].Status != StatusDone {
		t.Errorf("List() = %+v, want a mensagem com status %q", list, StatusDone)
	}
}

func TestFileStoreIgnoraArquivosQueNaoSaoDeConversa(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	c := mustCreate(t, s, "Única")

	// Intrusos que não são conversas: não podem aparecer na lista nem derrubar
	// a leitura do diretório.
	for _, nome := range []string{".DS_Store", "rascunho.txt", "meio" + fileExt + tmpSuffix} {
		if err := os.WriteFile(filepath.Join(dir, nome), []byte("lixo"), filePerm); err != nil {
			t.Fatalf("WriteFile(%s) erro = %v", nome, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), dirPerm); err != nil {
		t.Fatalf("Mkdir() erro = %v", err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List() erro = %v", err)
	}
	if len(list) != 1 || list[0].ID != c.ID {
		t.Errorf("List() = %+v, want só %s", list, c.ID)
	}
}

func TestFileStoreDiretorioInexistente(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ainda-nao-existe")
	s := NewFileStore(dir)

	// Histórico que ainda não nasceu é lista vazia, não erro.
	list, err := s.List()
	if err != nil {
		t.Fatalf("List() erro = %v, want lista vazia", err)
	}
	if len(list) != 0 {
		t.Errorf("List() = %+v, want vazia", list)
	}

	// A primeira escrita cria o diretório.
	c := mustCreate(t, s, "Primeira conversa")
	if _, err := s.Get(c.ID); err != nil {
		t.Fatalf("Get() erro = %v", err)
	}
}

func TestFileStoreListFalhaComArquivoCorrompido(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	mustCreate(t, s, "Boa")

	if err := os.WriteFile(filepath.Join(dir, "quebrada"+fileExt), []byte("{isso não é json"), filePerm); err != nil {
		t.Fatalf("WriteFile() erro = %v", err)
	}

	// Falha alta, não silenciosa: a conversa que sumisse da lista sem
	// explicação seria pior que o erro.
	if _, err := s.List(); !errors.Is(err, ErrInvalid) {
		t.Errorf("List() erro = %v, want ErrInvalid", err)
	}
	if _, err := s.Get("quebrada"); !errors.Is(err, ErrInvalid) {
		t.Errorf("Get() erro = %v, want ErrInvalid", err)
	}
}

func TestFileStoreRejeitaIDInseguro(t *testing.T) {
	// O id vira <dir>/<id>.json: separador de caminho, ".." e afins escreveriam
	// fora do diretório do histórico.
	tests := []string{"", ".", "..", "../fora", "sub/dir", `sub\dir`, "id.", "id "}

	for _, id := range tests {
		t.Run(fmt.Sprintf("%q", id), func(t *testing.T) {
			s := NewFileStore(t.TempDir())

			if _, err := s.Get(id); !errors.Is(err, ErrInvalid) {
				t.Errorf("Get(%q) erro = %v, want ErrInvalid", id, err)
			}
			if _, err := s.Append(id, userMessage("x")); !errors.Is(err, ErrInvalid) {
				t.Errorf("Append(%q) erro = %v, want ErrInvalid", id, err)
			}
			if _, err := s.Rename(id, "x"); !errors.Is(err, ErrInvalid) {
				t.Errorf("Rename(%q) erro = %v, want ErrInvalid", id, err)
			}
			if err := s.Delete(id); !errors.Is(err, ErrInvalid) {
				t.Errorf("Delete(%q) erro = %v, want ErrInvalid", id, err)
			}
		})
	}
}

func TestFileStoreAppendInexistenteNaoCriaArquivo(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	id := uuid.NewString()

	if _, err := s.Append(id, userMessage("Órfã")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Append() erro = %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(filepath.Join(dir, id+fileExt)); !os.IsNotExist(err) {
		t.Errorf("Append() em id inexistente mexeu no disco (Stat erro = %v)", err)
	}
}

func TestFileStoreDefaultDir(t *testing.T) {
	dir := DefaultDir()

	if !filepath.IsAbs(dir) {
		t.Errorf("DefaultDir() = %q, want caminho absoluto", dir)
	}
	want := filepath.Join(appDirName, historyDir)
	if !strings.HasSuffix(dir, want) {
		t.Errorf("DefaultDir() = %q, want terminar em %q", dir, want)
	}
}

func TestFileStoreConcorrencia(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	c := mustCreate(t, s, "Concorrente")

	const goroutines = 8
	const porGoroutine = 5

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < porGoroutine; i++ {
				msg := Message{
					ID:      fmt.Sprintf("g%d-m%d", g, i),
					Role:    RoleUser,
					Content: "oi",
					Status:  StatusDone,
				}
				if _, err := s.Append(c.ID, msg); err != nil {
					t.Errorf("Append() erro = %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	// Last-write-wins vale entre processos; dentro do processo o mutex garante
	// que ninguém sobrescreve o turno do outro.
	got, err := s.Get(c.ID)
	if err != nil {
		t.Fatalf("Get() erro = %v", err)
	}
	if len(got.Messages) != goroutines*porGoroutine {
		t.Errorf("Get() messages = %d, want %d", len(got.Messages), goroutines*porGoroutine)
	}
	vistos := map[string]bool{}
	for _, m := range got.Messages {
		vistos[m.ID] = true
	}
	if len(vistos) != goroutines*porGoroutine {
		t.Errorf("Get() tem %d mensagens distintas, want %d", len(vistos), goroutines*porGoroutine)
	}

	assertSemTmp(t, dir)
}

// assertSemTmp falha se sobrou arquivo temporário no diretório do histórico.
func assertSemTmp(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s) erro = %v", dir, err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), tmpSuffix) {
			t.Errorf("sobrou arquivo temporário %s no diretório do histórico", e.Name())
		}
	}
}
