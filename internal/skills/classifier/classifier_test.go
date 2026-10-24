package classifier

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeCompletar injeta respostas canned: os testes rodam sem rede e sem LLM.
type fakeCompletar struct {
	fila     []respostaCanned
	chamadas []chamada
}

type respostaCanned struct {
	texto string
	err   error
}

type chamada struct {
	msgs   []message
	schema map[string]any
}

func (f *fakeCompletar) completar(ctx context.Context, msgs []message, schema map[string]any) (string, error) {
	f.chamadas = append(f.chamadas, chamada{msgs: copiarMsgs(msgs), schema: schema})
	if len(f.fila) == 0 {
		panic("fakeCompletar: sem respostas na fila")
	}
	r := f.fila[0]
	f.fila = f.fila[1:]
	return r.texto, r.err
}

func copiarMsgs(msgs []message) []message {
	copia := make([]message, len(msgs))
	copy(copia, msgs)
	return copia
}

func listasDeTeste() []List {
	return []List{
		{Name: "protocol", Values: []Value{
			{Name: "bgp", Description: "BGP routing"},
			{Name: "ospf"},
		}},
		{Name: "daemon", Values: []Value{
			{Name: "bgpd"},
			{Name: "ospfd"},
		}},
	}
}

// conferirPertencimento é a verificação de pertencimento exigida pelo aceite:
// todo valor de saída é "" ou pertence à lista, em todos os caminhos.
func conferirPertencimento(t *testing.T, got map[string]string, listas []List) {
	t.Helper()
	if len(got) != len(listas) {
		t.Errorf("saída tem %d entradas, want %d (%v)", len(got), len(listas), got)
	}
	for _, l := range listas {
		v, ok := got[l.Name]
		if !ok {
			t.Errorf("lista %q sem entrada na saída", l.Name)
			continue
		}
		if v == "" {
			continue
		}
		pertence := false
		for _, cand := range l.Values {
			if cand.Name == v {
				pertence = true
			}
		}
		if !pertence {
			t.Errorf("valor %q da lista %q não pertence à lista", v, l.Name)
		}
	}
}

// Caso 1: escolha válida em cada lista.
func TestEscolhaValida(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{{texto: `{"protocol":"bgp","daemon":"bgpd"}`}}}

	got, err := classify(context.Background(), "Quero configurar uma sessão BGP", listasDeTeste(), false, fake.completar)
	if err != nil {
		t.Fatalf("classificar: %v", err)
	}
	if got["protocol"] != "bgp" || got["daemon"] != "bgpd" {
		t.Errorf("saída = %v, want protocol=bgp daemon=bgpd", got)
	}
	if len(fake.chamadas) != 1 {
		t.Errorf("chamadas = %d, want 1", len(fake.chamadas))
	}
	conferirPertencimento(t, got, listasDeTeste())
}

// Caso 2: nada se encaixa → vazio explícito (""), nunca valor inventado.
func TestNadaSeEncaixa(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{{texto: `{"protocol":"","daemon":""}`}}}

	got, err := classify(context.Background(), "Como faço backup do switch?", listasDeTeste(), false, fake.completar)
	if err != nil {
		t.Fatalf("classificar: %v", err)
	}
	if got["protocol"] != "" || got["daemon"] != "" {
		t.Errorf("saída = %v, want vazios explícitos", got)
	}
	conferirPertencimento(t, got, listasDeTeste())
}

// Caso 3: valor fora da lista → retry com o erro colado na próxima tentativa.
func TestValorForaDaLista_Refaz(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: `{"protocol":"frr","daemon":"bgpd"}`},
		{texto: `{"protocol":"ospf","daemon":"ospfd"}`},
	}}

	got, err := classify(context.Background(), "Configurar OSPF", listasDeTeste(), false, fake.completar)
	if err != nil {
		t.Fatalf("classificar: %v", err)
	}
	if got["protocol"] != "ospf" || got["daemon"] != "ospfd" {
		t.Errorf("saída = %v, want protocol=ospf daemon=ospfd", got)
	}
	if len(fake.chamadas) != 2 {
		t.Fatalf("chamadas = %d, want 2", len(fake.chamadas))
	}
	correcao := fake.chamadas[1].msgs[len(fake.chamadas[1].msgs)-1]
	if correcao.Role != "user" || !strings.Contains(correcao.Content, "frr") || !strings.Contains(correcao.Content, "bgp") {
		t.Errorf("correção da 2ª chamada = %+v, want menciona o valor rejeitado e os permitidos", correcao)
	}
	conferirPertencimento(t, got, listasDeTeste())
}

// Caso 3 (esgotado): valor fora da lista que persiste → erro explícito.
func TestValorForaDaLista_Esgota(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: `{"protocol":"frr","daemon":"bgpd"}`},
		{texto: `{"protocol":"FRR","daemon":"bgpd"}`},
		{texto: `{"protocol":"cisco","daemon":"bgpd"}`},
	}}

	got, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar)
	if err == nil {
		t.Fatalf("classificar devolveu %v sem erro, want erro", got)
	}
	if got != nil {
		t.Errorf("saída = %v, want nil (valor inventado não vaza)", got)
	}
	if len(fake.chamadas) != maxAttempts {
		t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxAttempts)
	}
	if !strings.Contains(err.Error(), "frr") && !strings.Contains(err.Error(), "cisco") {
		t.Errorf("erro = %v, want menciona o valor rejeitado", err)
	}
}

// Caso 4: JSON malformado → erro após as tentativas.
func TestJSONMalformado(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: "Desculpe, não posso ajudar com isso."},
		{texto: `{"protocol": "bgp",`},
		{texto: "```json\n{protocol: bgp}\n```"},
	}}

	got, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar)
	if err == nil {
		t.Fatalf("classificar devolveu %v sem erro, want erro", got)
	}
	if got != nil {
		t.Errorf("saída = %v, want nil", got)
	}
	if len(fake.chamadas) != maxAttempts {
		t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxAttempts)
	}
}

// Caso 5: resposta que ignora o schema → a validação pega. Prosa com JSON
// inventado, valor não-texto e lista ausente nunca viram saída.
func TestRespostaQueIgnoraSchema(t *testing.T) {
	casos := []struct {
		nome  string
		bruto string
	}{
		{"valor inventado em fence", "Claro! ```json\n{\"protocol\":\"frr\",\"daemon\":\"bgpd\"}\n```"},
		{"valor não-texto", `{"protocol": 42, "daemon": "bgpd"}`},
		{"lista ausente", `{"protocol": "bgp"}`},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			fake := &fakeCompletar{fila: []respostaCanned{
				{texto: c.bruto}, {texto: c.bruto}, {texto: c.bruto},
			}}
			got, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar)
			if err == nil {
				t.Fatalf("classificar devolveu %v sem erro, want erro", got)
			}
			if got != nil {
				t.Errorf("saída = %v, want nil", got)
			}
			if len(fake.chamadas) != maxAttempts {
				t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxAttempts)
			}
		})
	}
}

// Rede de segurança: JSON válido dentro de code fence é aceito (modelos locais
// emitem fences mesmo instruídos a não fazer).
func TestJSONEmFence(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: "```json\n{\"protocol\":\"bgp\",\"daemon\":\"bgpd\"}\n```"},
	}}

	got, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar)
	if err != nil {
		t.Fatalf("classificar: %v", err)
	}
	if got["protocol"] != "bgp" || got["daemon"] != "bgpd" {
		t.Errorf("saída = %v, want protocol=bgp daemon=bgpd", got)
	}
	conferirPertencimento(t, got, listasDeTeste())
}

// A grafia do modelo (caixa, anotação, espaço) nunca vaza: sai a grafia
// canônica da lista.
func TestNormalizacaoCanonica(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{
		{texto: `{"protocol":"BGP (primary)","daemon":"  bgpd  "}`},
	}}

	got, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar)
	if err != nil {
		t.Fatalf("classificar: %v", err)
	}
	if got["protocol"] != "bgp" || got["daemon"] != "bgpd" {
		t.Errorf("saída = %v, want os valores canônicos", got)
	}
	conferirPertencimento(t, got, listasDeTeste())
}

// Entrada inválida é recusada antes de qualquer ida ao modelo.
func TestEntradaInvalida(t *testing.T) {
	casos := []struct {
		nome   string
		listas []List
	}{
		{"nenhuma lista", nil},
		{"lista sem nome", []List{{Name: "  ", Values: []Value{{Name: "bgp"}}}}},
		{"lista sem valores", []List{{Name: "protocol"}}},
		{"valor sem nome", []List{{Name: "protocol", Values: []Value{{Name: ""}}}}},
		{"valores equivalentes", []List{{Name: "protocol", Values: []Value{{Name: "BGP"}, {Name: "bgp"}}}}},
		{"lista repetida", []List{
			{Name: "protocol", Values: []Value{{Name: "bgp"}}},
			{Name: "protocol", Values: []Value{{Name: "ospf"}}},
		}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			fake := &fakeCompletar{} // sem fila: panic se o modelo for chamado
			got, err := classify(context.Background(), "Configurar BGP", c.listas, false, fake.completar)
			if err == nil {
				t.Fatalf("classificar devolveu %v sem erro, want erro", got)
			}
			if len(fake.chamadas) != 0 {
				t.Errorf("chamadas = %d, want 0 (entrada inválida não vai ao modelo)", len(fake.chamadas))
			}
		})
	}
}

func TestInputVazio(t *testing.T) {
	fake := &fakeCompletar{}
	if _, err := classify(context.Background(), "   ", listasDeTeste(), false, fake.completar); err == nil {
		t.Fatal("classificar aceitou input vazio")
	}
	if len(fake.chamadas) != 0 {
		t.Errorf("chamadas = %d, want 0", len(fake.chamadas))
	}
}

// Erro do provedor: retry ×2 e depois erro explícito, com o motivo.
func TestErroDoProvedor(t *testing.T) {
	foraDoAr := errors.New("provedor fora do ar")
	fake := &fakeCompletar{fila: []respostaCanned{
		{err: foraDoAr}, {err: foraDoAr}, {err: foraDoAr},
	}}

	got, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar)
	if err == nil {
		t.Fatalf("classificar devolveu %v sem erro, want erro", got)
	}
	if !strings.Contains(err.Error(), "fora do ar") {
		t.Errorf("erro = %v, want menciona o motivo do provedor", err)
	}
	if len(fake.chamadas) != maxAttempts {
		t.Errorf("chamadas = %d, want %d", len(fake.chamadas), maxAttempts)
	}
}

// O schema pedido ao provedor é o contrato: uma propriedade por lista, enum
// com os valores permitidos e o vazio explícito, tudo required, sem extras.
func TestSchema(t *testing.T) {
	fake := &fakeCompletar{fila: []respostaCanned{{texto: `{"protocol":"bgp","daemon":"bgpd"}`}}}
	if _, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), false, fake.completar); err != nil {
		t.Fatalf("classificar: %v", err)
	}
	schema := fake.chamadas[0].schema

	if schema["type"] != "object" {
		t.Errorf("type = %v, want object", schema["type"])
	}
	if schema["additionalProperties"] != false {
		t.Errorf("additionalProperties = %v, want false", schema["additionalProperties"])
	}
	required, _ := schema["required"].([]string)
	if len(required) != 2 || required[0] != "protocol" || required[1] != "daemon" {
		t.Errorf("required = %v, want [protocol daemon]", required)
	}
	props, _ := schema["properties"].(map[string]any)
	proto, _ := props["protocol"].(map[string]any)
	enum, _ := proto["enum"].([]string)
	want := []string{"", "bgp", "ospf"}
	if len(enum) != len(want) {
		t.Fatalf("enum = %v, want %v", enum, want)
	}
	for i, v := range want {
		if enum[i] != v {
			t.Errorf("enum = %v, want %v", enum, want)
		}
	}
}

// A flag é o Modo RAG: parâmetro da chamada que muda o comportamento interno
// (prompt), não a forma do contrato.
func TestRAGMode(t *testing.T) {
	for _, modoRAG := range []bool{false, true} {
		fake := &fakeCompletar{fila: []respostaCanned{{texto: `{"protocol":"bgp","daemon":"bgpd"}`}}}
		if _, err := classify(context.Background(), "Configurar BGP", listasDeTeste(), modoRAG, fake.completar); err != nil {
			t.Fatalf("classificar(modoRAG=%v): %v", modoRAG, err)
		}
		system := fake.chamadas[0].msgs[0]
		if system.Role != "system" {
			t.Errorf("primeira mensagem = %q, want system", system.Role)
		}
		if contains := strings.Contains(system.Content, "RAG Mode"); contains != modoRAG {
			t.Errorf("system prompt mentions RAG Mode = %v, want %v", contains, modoRAG)
		}
	}
}
