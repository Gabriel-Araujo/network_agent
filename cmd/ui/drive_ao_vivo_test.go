package main

import (
	"cmp"
	"encoding/json"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
)

// Drive ao vivo do ticket 07 (.agent/ui/issues/07-drive-ao-vivo-do-roteamento-pela-ui.md).
//
// Os testes de history_test.go trocam a composição por um seam (turnoSoDoChat)
// porque o turn.Run de verdade busca no banco do RAG e chama o LLM — e por isso
// a chamada real nunca foi exercitada. Este arquivo é o drive: o MESMO App da
// produção (NewApp lê ~/.network_agent/config.json), o runTurn de verdade, e o
// CWD que o `go test ./cmd/ui/...` dá ao binário — cmd/ui, exatamente o CWD que
// o ticket 01 mediu no app sob `wails dev`.
//
// Fica desligado por default: sem UI_DRIVE_AO_VIVO=1 o gate escopado segue sem
// depender do Postgres nem do servidor de embedding da máquina.
//
//	LOG_FILE=<dir do asset> UI_DRIVE_AO_VIVO=1 \
//	  go test ./cmd/ui/ -run TestDriveAoVivo -v -count=1 -timeout 60m

// logDoDrive liga o logger do projeto como o main() liga: é ele que grava a
// evidência dos drives, e o arquivo dele É o asset do ticket.
func logDoDrive(t *testing.T) {
	t.Helper()

	// As mesmas quatro linhas do main().
	cfg := logger.ConfigFromEnv()
	if cfg.File == "" {
		cfg.File = filepath.Join(filepath.Dir(config.Path()), "logs")
	}
	cfg.Binary = "ui"
	// A gaveta de Logs do ticket 12: o mesmo tee que o main.go liga antes do
	// Init, para o anel receber desde a primeira linha do run.
	cfg.ConsoleTee = logs
	logger.Init(cfg)
}

func appAoVivo(t *testing.T) *App {
	t.Helper()
	if os.Getenv("UI_DRIVE_AO_VIVO") != "1" {
		t.Skip("drive ao vivo: exige UI_DRIVE_AO_VIVO=1, o LM Studio e o banco do RAG de pé")
	}

	logDoDrive(t)

	// NewApp é o App da produção: lê o config.json real e conecta o provedor e
	// o modelo que o usuário escolheu. Só o histórico e os eventos são
	// desviados — o disco do usuário não é cobaia e a ponte do Wails não existe
	// num binário de teste.
	dir := cmp.Or(os.Getenv("UI_DRIVE_STORE"), t.TempDir())
	a := NewApp()
	a.ctx = t.Context()
	a.store = history.NewFileStore(dir)
	a.emit = func(string, ...any) {}

	// A conexão é background (ticket 13) e a carga do modelo no LM Studio leva
	// dezenas de segundos a frio: o drive espera por ela, que é justamente o
	// que o app não espera mais.
	esperaConectar(t, a)
	t.Logf("provedor=%q modelo=%q histórico=%s",
		a.activeConfig().Provider, a.activeConfig().Model, dir)
	return a
}

// turnoDoDrive roda um turno de verdade e relata o que a composição devolveu.
// Não afirma qual Subagente o modelo deveria ter escolhido: o ticket fecha com
// a evidência, inclusive quando ela é de falha, e classificação errada é fato
// do modelo, não defeito da ponte. O que ele exige é que o turno exista.
func turnoDoDrive(t *testing.T, a *App, convID, rotulo, prompt string) history.Message {
	t.Helper()

	conv, err := a.AskAgent(convID, prompt)
	if err != nil {
		t.Fatalf("[%s] AskAgent: %v", rotulo, err)
	}

	msg := conv.Messages[len(conv.Messages)-1]
	t.Logf("[%s] subagente=%q status=%q avisos=%v chars=%d",
		rotulo, msg.Subagent, msg.Status, msg.Warnings, len(msg.Content))
	if msg.Status == history.StatusError {
		t.Logf("[%s] o turno falhou: %s", rotulo, msg.Content)
	}
	return msg
}

// TestDriveAoVivoRoteamento é o §1 e o §2 do ticket: um turno de cada Subagente
// com o autor chegando persistido na mensagem, e dois turnos seguidos de mesma
// classificação para a skip rule do Modo RAG pular o segundo retrieval.
//
// Os cinco turnos rodam na MESMA Conversa, de propósito: o State da composição
// mora num map[convID]turn.State no App, então é a Conversa que tem turno
// anterior e só dentro dela a skip rule pode pular.
func TestDriveAoVivoRoteamento(t *testing.T) {
	a := appAoVivo(t)

	conv, err := a.CreateConversation("drive ao vivo do roteamento")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	turnos := []struct{ rotulo, prompt string }{
		// Os dois primeiros são a mesma pergunta de outro jeito: mesma
		// classificação esperada (bgp/bgpd/concept), logo `retrieval saved`
		// no log do primeiro e ausente no do segundo.
		{"teacher-1", "O que é um route reflector no BGP e por que ele existe?"},
		{"teacher-2", "Explique por que o route reflector dispensa o full mesh de iBGP."},
		{"builder", "Crie um arquivo de configuração FRR novo para um roteador com BGP no AS 65001 e o vizinho 10.0.0.2 no AS 65002."},
		{"modifier", "Altere o arquivo frr.conf que já existe para acrescentar o vizinho 10.0.0.3 no AS 65003."},
		{"illustrator", "Desenhe um diagrama da topologia de dois roteadores iBGP ligados a um route reflector."},
	}

	autores := map[string]string{}
	for _, turno := range turnos {
		msg := turnoDoDrive(t, a, conv.ID, turno.rotulo, turno.prompt)
		autores[turno.rotulo] = msg.Subagent
	}

	t.Logf("autores por turno: %v", autores)
}

// TestDriveAoVivoDegradado é o §4: o turno degradado. Roda com o banco do RAG
// derrubado (`podman stop agent-rag-db`), que é o que força o caso sem esperar
// a sorte — o retrieval falha, a composição segue sem contexto e o aviso tem de
// chegar à mensagem, que é de onde a bolha tira a tira âmbar (ticket 04).
func TestDriveAoVivoDegradado(t *testing.T) {
	a := appAoVivo(t)

	conv, err := a.CreateConversation("drive ao vivo degradado")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	msg := turnoDoDrive(t, a, conv.ID, "degradado",
		"O que é um route reflector no BGP e por que ele existe?")

	if len(msg.Warnings) == 0 {
		t.Fatal("nenhum aviso na mensagem: o banco do RAG estava de pé? o drive degradado exige o retrieval falhando")
	}
	for _, aviso := range msg.Warnings {
		if strings.TrimSpace(aviso) == "" {
			t.Error("aviso vazio na mensagem — a tira âmbar renderizaria em branco")
		}
	}
	if msg.Status == history.StatusError {
		t.Error("o retrieval caído derrubou o turno: degradação nunca falha o turno (internal/turn/turn.go:100-105)")
	}
}

// TestDriveAoVivoGaveta é o drive do ticket 12: a gaveta de Logs com as linhas
// de um turno real. Prova o que o teste de unidade não alcança — o tee ligado
// no caminho do app (ConsoleTee antes do Init), o anel recebendo desde a
// primeira linha do run, e o "copiar o caminho" apontando para o arquivo DESTA
// execução — e deixa o asset pronto: o .log da execução e, ao lado dele, o
// despejo do que a gaveta mostra.
//
// A igualdade byte a byte entre gaveta e terminal é do TestConsoleTee, no
// internal/logger; aqui o que se mede é que os dois sinks estão vendo o mesmo
// run: todo Lugar que aparece na gaveta aparece no arquivo.
func TestDriveAoVivoGaveta(t *testing.T) {
	a := appAoVivo(t)

	// O anel já tem as linhas do NewApp(), que no app rodam antes do startup —
	// é onde "falha ao ler a configuração" e "falha ao conectar ao provedor"
	// aparecem, e por isso elas entram no anel.
	antes := a.GetLogLines()
	if len(antes) == 0 {
		t.Fatal("anel vazio antes do turno: o ConsoleTee não chegou ao sink de console")
	}
	t.Logf("antes do turno: %d linhas; a primeira é %q", len(antes), antes[0])

	conv, err := a.CreateConversation("drive ao vivo da gaveta de logs")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	turnoDoDrive(t, a, conv.ID, "gaveta",
		"O que é um route reflector no BGP e por que ele existe?")

	linhas := a.GetLogLines()
	if len(linhas) <= len(antes) {
		t.Fatal("o turno não acrescentou linha nenhuma ao anel")
	}

	path := a.GetLogPath()
	if path == "" {
		t.Fatal("GetLogPath vazio: sem caminho não existe o copiar o caminho da gaveta")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("o caminho que a gaveta copia não abre: %v", err)
	}

	lugares := map[string]bool{}
	for _, linha := range linhas {
		if lugar := lugarDaLinha(linha); lugar != "" {
			lugares[lugar] = true
		}
	}
	if len(lugares) == 0 {
		t.Fatal("nenhum Lugar reconhecido nas linhas do anel: o formato do console mudou")
	}
	for lugar := range lugares {
		if !strings.Contains(string(raw), `"place":"`+lugar+`"`) {
			t.Errorf("o Lugar %q está na gaveta e não no arquivo: os dois sinks não estão vendo o mesmo run", lugar)
		}
	}

	// O despejo é o que a gaveta mostra, em texto, ao lado do .log — as duas
	// metades da evidência no mesmo diretório.
	dump := filepath.Join(filepath.Dir(path), "gaveta.txt")
	if err := os.WriteFile(dump, []byte(strings.Join(linhas, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("gravando o despejo da gaveta: %v", err)
	}

	niveis := map[string]int{}
	for _, linha := range linhas {
		niveis[nivelDaLinha(linha)]++
	}
	t.Logf("depois do turno: %d linhas, Lugares=%v, níveis=%v", len(linhas), chavesOrdenadas(lugares), niveis)
	t.Logf("arquivo da execução: %s (%d bytes)", path, len(raw))
	t.Logf("despejo da gaveta: %s", dump)
}

// lugarDaLinha tira o Lugar de uma linha do console: "<hora> [NIVEL ] LUGAR - msg".
func lugarDaLinha(linha string) string {
	_, resto, ok := strings.Cut(linha, "] ")
	if !ok {
		return ""
	}
	lugar, _, ok := strings.Cut(resto, " - ")
	if !ok {
		return ""
	}
	return lugar
}

// nivelDaLinha lê o nível pelo mesmo offset fixo que o front usa para colorir
// (LEVEL_AT no LogDrawer.tsx): 23 chars de timestamp, espaço, e o rótulo.
func nivelDaLinha(linha string) string {
	const at = 24
	if len(linha) < at+7 || linha[at] != '[' {
		return "?"
	}
	return strings.TrimSpace(linha[at+1 : at+6])
}

func chavesOrdenadas(m map[string]bool) []string {
	return slices.Sorted(maps.Keys(m))
}

// ---------- drive da conexão (ticket 13) ----------

// instanciasCarregadas sonda o LM Studio e devolve as janelas das instâncias
// carregadas do modelo. É a precondição do drive deste ticket: com o modelo já
// quente a carga é instantânea e o drive passa com o defeito de pé.
func instanciasCarregadas(t *testing.T, providerURL, modelID string) []int64 {
	t.Helper()

	root := strings.TrimSuffix(strings.TrimRight(providerURL, "/"), "/v1") + "/api/v1/models"
	resp, err := http.Get(root)
	if err != nil {
		t.Fatalf("sondando o LM Studio em %s: %v", root, err)
	}
	defer resp.Body.Close()

	var corpo struct {
		Models []struct {
			Key             string `json:"key"`
			ID              string `json:"id"`
			LoadedInstances []struct {
				Config struct {
					ContextLength int64 `json:"context_length"`
				} `json:"config"`
			} `json:"loaded_instances"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&corpo); err != nil {
		t.Fatalf("resposta da sondagem ilegível: %v", err)
	}

	var janelas []int64
	for _, m := range corpo.Models {
		if m.Key != modelID && m.ID != modelID {
			continue
		}
		for _, inst := range m.LoadedInstances {
			janelas = append(janelas, inst.Config.ContextLength)
		}
	}
	return janelas
}

// TestDriveDaConexaoComOModeloDescarregado é o drive ao vivo do ticket 13, no
// único estado que distingue o conserto: o modelo DESCARREGADO, que é como o
// LM Studio fica depois do idle e em toda partida a frio.
//
// Diferente dos outros drives deste arquivo, não precisa do banco do RAG nem
// roda turno nenhum: o que ele mede é o arranque do App — que NewApp volta na
// hora em vez de esperar a carga, que a conexão termina em agente, e que a
// carga durou mais do que os 3s da sondagem (internal/agent.lmStudioProbeTimeout),
// que é o deadline que o defeito aplicava nela.
//
//	lms unload --all
//	LOG_FILE=<dir do asset> UI_DRIVE_AO_VIVO=1 \
//	  go test ./cmd/ui/ -run TestDriveDaConexaoComOModeloDescarregado -v -count=1 -timeout 20m
func TestDriveDaConexaoComOModeloDescarregado(t *testing.T) {
	if os.Getenv("UI_DRIVE_AO_VIVO") != "1" {
		t.Skip("drive ao vivo: exige UI_DRIVE_AO_VIVO=1, o LM Studio de pé e o modelo descarregado")
	}
	logDoDrive(t)

	cfg, err := readConfig()
	if err != nil {
		t.Fatalf("lendo a configuração real: %v", err)
	}
	config.Normalize(&cfg)
	provider, model, ok := selectedModel(cfg)
	if !ok {
		t.Fatal("nenhum provedor/modelo selecionado: o drive precisa do config.json real")
	}

	// A precondição é o drive: com instância carregada, a carga não acontece e
	// o teste não discrimina conserto de defeito.
	if janelas := instanciasCarregadas(t, provider.Url, model.Id); len(janelas) > 0 {
		t.Fatalf("o modelo %q já está carregado (janelas %v): rode `lms unload --all` antes — quente, este drive passa com o defeito de pé", model.Id, janelas)
	}
	t.Logf("precondição: %q descarregado no %s, janela pedida %d", model.Id, provider.Url, model.ContextSize)

	// 1. O arranque não espera a carga. NewApp roda antes do wails.Run: o que
	//    demorar aqui é tela nenhuma na frente do usuário.
	comecou := time.Now()
	a := NewApp()
	arranque := time.Since(comecou)
	estadoNoArranque := a.GetConnection().State

	if arranque > time.Second {
		t.Errorf("NewApp levou %s: o arranque voltou a esperar a conexão", arranque.Round(time.Millisecond))
	}
	if estadoNoArranque != connConnecting {
		t.Errorf("estado no arranque %q, esperava %q", estadoNoArranque, connConnecting)
	}
	t.Logf("NewApp voltou em %s com a conexão em %q", arranque.Round(time.Millisecond), estadoNoArranque)

	// 2. A conexão termina em agente, e a carga é o que demora.
	for a.GetConnection().State == connConnecting {
		if time.Since(comecou) > 10*time.Minute {
			t.Fatal("a conexão não terminou em 10min")
		}
		time.Sleep(100 * time.Millisecond)
	}
	carga := time.Since(comecou)

	conn := a.GetConnection()
	if conn.State != connReady {
		t.Fatalf("a conexão terminou em %q depois de %s — causa: %s", conn.State, carga.Round(time.Millisecond), conn.Cause)
	}
	if a.currentAgent() == nil {
		t.Fatal("conexão ready sem agente")
	}
	t.Logf("conexão %q em %s; provedor=%q modelo=%q", conn.State, carga.Round(time.Millisecond),
		a.activeConfig().Provider, a.activeConfig().Model)

	// 3. O drive discrimina: a carga passou do deadline que o defeito aplicava
	//    nela. Abaixo disso, o conserto e o defeito são indistinguíveis aqui.
	const prazoDaSondagem = 3 * time.Second
	if carga < prazoDaSondagem {
		t.Errorf("a carga levou só %s: menos que os %s da sondagem, então este drive não distingue o conserto do defeito",
			carga.Round(time.Millisecond), prazoDaSondagem)
	}

	// 4. A janela pedida é a que ficou carregada: a carga fez o que o Connect
	//    pediu, não só devolveu 200.
	janelas := instanciasCarregadas(t, provider.Url, model.Id)
	if !slices.Contains(janelas, model.ContextSize) {
		t.Errorf("janelas carregadas %v, nenhuma com os %d pedidos", janelas, model.ContextSize)
	}
	t.Logf("janelas carregadas depois da conexão: %v", janelas)
}
