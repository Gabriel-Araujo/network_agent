package briefing

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

// fakeCompletar injeta respostas canned: os testes rodam sem rede e sem LLM.
type fakeCompletar struct {
	fila     []respostaCanned
	chamadas [][]mensagem
}

type respostaCanned struct {
	texto string
	err   error
}

func (f *fakeCompletar) completar(ctx context.Context, msgs []mensagem) (string, error) {
	copia := make([]mensagem, len(msgs))
	copy(copia, msgs)
	f.chamadas = append(f.chamadas, copia)
	if len(f.fila) == 0 {
		panic("fakeCompletar: sem respostas na fila")
	}
	r := f.fila[0]
	f.fila = f.fila[1:]
	return r.texto, r.err
}

// briefingExemplo é o template do prompt: um input de exemplo respondido com o
// briefing no formato decidido — as quatro partes, com os nomes e shapes de
// hoje.
const briefingExemplo = `{
  "devices": [
    {"label": "R1", "role": "core", "platform": "FRR", "daemonsAndProtocols": ["ospfd"]},
    {"label": "R2", "role": "edge", "platform": "FRR", "daemonsAndProtocols": ["ospfd", "bgpd"]},
    {"label": "upstream (AS 65010)", "role": "eBGP peer", "platform": "unknown/non-FRR", "daemonsAndProtocols": ["bgp"]}
  ],
  "connections": [
    {"from": "R1", "to": "R2", "InterfaceAndLink": "Gi0/1", "protocolAndSession": "OSPF area 0"},
    {"from": "R2", "to": "upstream", "InterfaceAndLink": "Gi0/2", "protocolAndSession": "eBGP AS 65010"}
  ],
  "problemAndGoalSummary": "The OSPF adjacency between R1 (core) and R2 (edge) isn't converging in area 0 over Gi0/1, both running FRR. R2 also holds a separate eBGP session with an upstream (AS 65010) over Gi0/2. The user wants the cause of the OSPF non-convergence and whether the eBGP session could be related.",
  "executionPipeline": {
    "1": "Confirm ospfd is enabled and running on R1 and R2.",
    "2": "Check basic L3 connectivity between R1 and R2 on Gi0/1 (ping, MTU).",
    "3": "Verify R1 and R2 are in the same area with matching interface parameters (network type, cost, authentication).",
    "4": "Check which state the adjacency is stuck in (Init, 2-Way, Exstart/Exchange, Loading).",
    "5": "Assess whether the eBGP session on R2 has any plausible relation to OSPF on R1/R2.",
    "6": "If stuck in Exstart/Exchange, check for an MTU mismatch between the interfaces."
  }
}`

const inputExemplo = "meu ospf não converge entre o core (R1) e o edge (R2), os dois rodando FRR, ligados pela Gi0/1 em area 0"

// tempHome põe o diretório de dados do app num t.TempDir() — é onde o espelho
// resolve agora, e sem a troca os testes gravariam no ~/.network_agent da
// máquina — e tira o CWD do repo, para um espelho que volte a nascer no
// diretório de trabalho aparecer num diretório vazio em vez de sujar a árvore.
func tempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Cleanup(config.SetDir(home))
	t.Chdir(t.TempDir())
	return home
}

// Caso 1: input de exemplo → briefing no formato decidido, com as quatro
// partes extraídas e o espelho gravado com o JSON da mesma struct.
func TestBriefingValido(t *testing.T) {
	home := tempHome(t)
	fake := &fakeCompletar{fila: []respostaCanned{{texto: briefingExemplo}}}

	got, path, err := do(context.Background(), inputExemplo, fake.completar)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if len(fake.chamadas) != 1 {
		t.Errorf("chamadas = %d, want 1", len(fake.chamadas))
	}

	if len(got.Devices) != 3 {
		t.Errorf("devices = %d, want 3", len(got.Devices))
	} else {
		r1 := got.Devices[0]
		if r1.Label != "R1" || r1.Role != "core" || r1.Platform != "FRR" || len(r1.DaemonsAndProtocols) != 1 || r1.DaemonsAndProtocols[0] != "ospfd" {
			t.Errorf("device[0] = %+v, want R1/core/FRR/[ospfd]", r1)
		}
	}
	if len(got.Connections) != 2 {
		t.Errorf("connections = %d, want 2", len(got.Connections))
	} else {
		c := got.Connections[0]
		if c.From != "R1" || c.To != "R2" || c.InterfaceAndLink != "Gi0/1" || c.ProtocolAndSession != "OSPF area 0" {
			t.Errorf("connection[0] = %+v, want R1→R2/Gi0/1/OSPF area 0", c)
		}
	}
	if !strings.Contains(got.ProblemAndGoalSummary, "OSPF adjacency") {
		t.Errorf("problemAndGoalSummary = %q, want o resumo padronizado", got.ProblemAndGoalSummary)
	}
	if len(got.ExecutionPipeline) != 6 || got.ExecutionPipeline["1"] == "" {
		t.Errorf("executionPipeline = %v, want 6 etapas numeradas", got.ExecutionPipeline)
	}

	// Espelho: <home>/tmp/<slug>-<hash8>.json com o JSON da mesma struct.
	wantNome := util.Slugify(inputExemplo) + "-" + util.Hash8(inputExemplo) + ".json"
	if want := filepath.Join(home, espelhoDir, wantNome); path != want {
		t.Errorf("espelho = %q, want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lendo espelho: %v", err)
	}
	var espelho Briefing
	if err := json.Unmarshal(data, &espelho); err != nil {
		t.Fatalf("espelho não é o JSON da struct: %v", err)
	}
	gotJSON, _ := json.Marshal(got)
	espelhoJSON, _ := json.Marshal(espelho)
	if string(gotJSON) != string(espelhoJSON) {
		t.Errorf("espelho difere do valor em memória:\n%s\n%s", espelhoJSON, gotJSON)
	}
}

// O mesmo input regrava o mesmo espelho (determinístico, sobrescreve).
func TestEspelhoDeterministico(t *testing.T) {
	tempHome(t)
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: briefingExemplo}, {texto: briefingExemplo},
	}}

	_, path1, err := do(context.Background(), inputExemplo, fake.completar)
	if err != nil {
		t.Fatalf("do (1a chamada): %v", err)
	}
	_, path2, err := do(context.Background(), inputExemplo, fake.completar)
	if err != nil {
		t.Fatalf("do (2a chamada): %v", err)
	}
	if path2 != path1 {
		t.Errorf("espelho mudou de nome: %q != %q", path2, path1)
	}
}

// Caso 2: chave ausente → retry com o erro colado na próxima tentativa.
func TestChaveAusente_Refaz(t *testing.T) {
	tempHome(t)
	semConnections := `{"devices": [], "problemAndGoalSummary": "x", "executionPipeline": {"1": "y"}}`
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: semConnections}, {texto: briefingExemplo},
	}}

	got, _, err := do(context.Background(), inputExemplo, fake.completar)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if len(got.Devices) != 3 {
		t.Errorf("devices = %d, want 3 (a 2a tentativa vence)", len(got.Devices))
	}
	if len(fake.chamadas) != 2 {
		t.Fatalf("chamadas = %d, want 2", len(fake.chamadas))
	}
	correcao := fake.chamadas[1][len(fake.chamadas[1])-1]
	if correcao.Papel != "user" || !strings.Contains(correcao.Conteudo, "connections") {
		t.Errorf("correção da 2a chamada = %+v, want menciona a chave rejeitada", correcao)
	}
}

// Caso 2 (esgotado): saída inválida que persiste → erro explícito, sem
// fallback silencioso.
func TestRespostaRejeitada_Esgota(t *testing.T) {
	tempHome(t)
	tipoErrado := `{"devices": "não é lista", "connections": [], "problemAndGoalSummary": "x", "executionPipeline": {"1": "y"}}`
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: tipoErrado}, {texto: tipoErrado}, {texto: tipoErrado},
	}}

	got, path, err := do(context.Background(), inputExemplo, fake.completar)
	if err == nil {
		t.Fatalf("do devolveu %+v sem erro, want erro", got)
	}
	if path != "" {
		t.Errorf("path = %q, want vazio", path)
	}
	if len(fake.chamadas) != maxTentativas {
		t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxTentativas)
	}
	if !strings.Contains(err.Error(), "devices") {
		t.Errorf("erro = %v, want menciona a parte rejeitada", err)
	}
}

// Caso 3: JSON malformado → erro após as tentativas.
func TestJSONMalformado(t *testing.T) {
	tempHome(t)
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: "Desculpe, não posso ajudar com isso."},
		{texto: `{"devices": [`},
		{texto: "```json\n{devices: x}\n```"},
	}}

	got, _, err := do(context.Background(), inputExemplo, fake.completar)
	if err == nil {
		t.Fatalf("do devolveu %+v sem erro, want erro", got)
	}
	if len(fake.chamadas) != maxTentativas {
		t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxTentativas)
	}
}

// Rede de segurança: JSON válido dentro de code fence é aceito (modelos locais
// emitem fences mesmo instruídos a não fazer).
func TestJSONEmFence(t *testing.T) {
	tempHome(t)
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: "Claro! ```json\n" + briefingExemplo + "\n```"},
	}}

	got, _, err := do(context.Background(), inputExemplo, fake.completar)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if len(got.Devices) != 3 {
		t.Errorf("devices = %d, want 3", len(got.Devices))
	}
}

// null não é seção vazia: a chave presente com null é rejeitada (o prompt manda
// emitir []).
func TestSecaoNula(t *testing.T) {
	tempHome(t)
	comNull := `{"devices": null, "connections": [], "problemAndGoalSummary": "x", "executionPipeline": {"1": "y"}}`
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: comNull}, {texto: comNull}, {texto: comNull},
	}}

	_, _, err := do(context.Background(), inputExemplo, fake.completar)
	if err == nil {
		t.Fatal("do aceitou devices null")
	}
	if !strings.Contains(err.Error(), "devices") {
		t.Errorf("erro = %v, want menciona a chave nula", err)
	}
}

// Falha de gravação do espelho: o briefing volta completo em memória, o path
// vem vazio e o erro é a sentinela checável com errors.Is.
func TestEspelhoNaoGravado(t *testing.T) {
	home := tempHome(t)
	// O espelhoDir como ARQUIVO impede o MkdirAll do espelho.
	if err := os.WriteFile(filepath.Join(home, espelhoDir), []byte("bloqueio"), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &fakeCompletar{fila: []respostaCanned{{texto: briefingExemplo}}}

	got, path, err := do(context.Background(), inputExemplo, fake.completar)
	if !errors.Is(err, ErrEspelhoNaoGravado) {
		t.Fatalf("erro = %v, want sentinela ErrEspelhoNaoGravado", err)
	}
	if path != "" {
		t.Errorf("path = %q, want vazio", path)
	}
	if len(got.Devices) != 3 || got.ProblemAndGoalSummary == "" {
		t.Errorf("briefing incompleto na falha de espelho: %+v", got)
	}
}

// Entrada inválida é recusada antes de qualquer ida ao modelo.
func TestInputVazio(t *testing.T) {
	fake := &fakeCompletar{} // sem fila: panic se o modelo for chamado
	if _, _, err := do(context.Background(), "   ", fake.completar); err == nil {
		t.Fatal("do aceitou input vazio")
	}
	if len(fake.chamadas) != 0 {
		t.Errorf("chamadas = %d, want 0 (input vazio não vai ao modelo)", len(fake.chamadas))
	}
}

// Erro do provedor: retry ×2 e depois erro explícito, com o motivo.
func TestErroDoProvedor(t *testing.T) {
	tempHome(t)
	foraDoAr := errors.New("provedor fora do ar")
	fake := &fakeCompletar{fila: []respostaCanned{
		{err: foraDoAr}, {err: foraDoAr}, {err: foraDoAr},
	}}

	_, _, err := do(context.Background(), inputExemplo, fake.completar)
	if err == nil {
		t.Fatal("do devolveu sem erro, want erro")
	}
	if !strings.Contains(err.Error(), "fora do ar") {
		t.Errorf("erro = %v, want menciona o motivo do provedor", err)
	}
	if len(fake.chamadas) != maxTentativas {
		t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxTentativas)
	}
}

// Do recusa agente nil antes de qualquer coisa.
func TestAgenteNil(t *testing.T) {
	if _, _, err := Do(context.Background(), inputExemplo, nil); err == nil {
		t.Fatal("Do aceitou agente nil")
	}
}
