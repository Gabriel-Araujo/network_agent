package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
)

// Drive ao vivo do ticket 19 (.agent/ui/issues/19-aviso-da-falha-de-tool.md).
//
// O mesmo harness dos drives dos tickets 07, 11, 12 e 13 (appAoVivo,
// turnoDoDrive): o App da produção, o runTurn de verdade e um modelo de
// verdade. Ligado por UI_DRIVE_AO_VIVO=1, como os outros.
//
// O que ele prova é o caminho inteiro do canal que o ticket 18 abriu e este
// fechou: a tool recusa, a falha sobe tipada pelo Chat, o turno a drena para
// os avisos e a Message grava — a tira da bolha lê daí (ticket 04). O pixel da
// tira fica para a conferência de app rodando, que está na neblina do mapa.

// TestDriveAoVivoAvisoDaFalhaDeTool pede um arquivo que não existe dentro do
// Workspace escolhido e mede as duas coisas que o ticket exige: o Aviso chega a
// history.Message.Warnings, e a pasta que ele mostra é a mesma contra a qual a
// tool resolveu — o absoluto que o .log traz no path= cai dentro dela.
//
// O caminho vai ABSOLUTO no pedido de propósito: a descrição do ReadFile manda
// o modelo procurar com o SearchFile antes quando recebe só um nome de arquivo,
// e uma busca vazia não é falha de tool — não registraria nada e o drive
// mediria o silêncio em vez da recusa.
func TestDriveAoVivoAvisoDaFalhaDeTool(t *testing.T) {
	a := appAoVivo(t)

	// A escolha entra na configuração em memória, como no drive do ticket 11:
	// o config.json do usuário não é cobaia de drive.
	escolhida := t.TempDir()
	cfg, err := readConfig()
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	config.Normalize(&cfg)
	cfg.Workspace = escolhida
	a.reconnect(cfg)
	esperaConectar(t, a)

	ausente := filepath.Join(escolhida, "frr.conf")
	if _, err := os.Stat(ausente); !os.IsNotExist(err) {
		t.Fatalf("o arquivo do drive existe (%v): o caso é o que NÃO existe", err)
	}
	t.Logf("Workspace escolhido: %s (vazio, o %s não existe)", escolhida, filepath.Base(ausente))

	conv, err := a.CreateConversation("drive ao vivo do Aviso de falha de tool")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	antes := len(a.GetLogLines())
	msg := turnoDoDrive(t, a, conv.ID, "leitura-ausente",
		"Leia o arquivo "+ausente+" e me diga em qual AS o BGP dele está configurado.")

	// 1. O Aviso está na mensagem gravada. Antes deste ticket o turno
	//    descartava a falha com `_` e a tela ficava à mercê do modelo — que,
	//    medido no ticket 14, acusou o caminho do usuário.
	aviso := ""
	for _, w := range msg.Warnings {
		t.Logf("aviso na mensagem: %s", w)
		if strings.HasPrefix(w, "leitura de") {
			aviso = w
		}
	}
	if aviso == "" {
		t.Fatalf("nenhum aviso de leitura em %q: o modelo não chamou o ReadFile, ou a falha morreu no caminho", msg.Warnings)
	}
	if msg.Status == history.StatusError {
		t.Errorf("o turno falhou (%q): falha de tool é degradação, não turno morto", msg.Content)
	}

	// 2. A pasta do Aviso é a raiz contra a qual a tool resolveu: o absoluto
	//    que o log.Error traz no path= cai dentro dela. É o par que o ticket
	//    pede — a raiz na tela, o absoluto no log que a gaveta mostra.
	raiz := entreAspas(t, aviso, `na pasta "`)
	if raiz != escolhida {
		t.Errorf("pasta no aviso = %q, esperava o Workspace %q", raiz, escolhida)
	}
	absoluto := ""
	for _, linha := range a.GetLogLines()[antes:] {
		if strings.Contains(linha, "falha ao ler arquivo") {
			absoluto = campoDoLog(linha, "path=")
			t.Logf("no log: %s", linha)
		}
	}
	if absoluto == "" {
		t.Fatal("nenhuma linha de 'falha ao ler arquivo' no log do run: a evidência do asset não existe")
	}
	if !strings.HasPrefix(absoluto, raiz) {
		t.Errorf("o absoluto do log (%q) não sai da pasta do aviso (%q)", absoluto, raiz)
	}

	// 3. O dedup (§2): o mesmo texto nunca aparece duas vezes, mesmo quando o
	//    modelo repete a tool no mesmo arquivo.
	vistos := map[string]bool{}
	for _, w := range msg.Warnings {
		if vistos[w] {
			t.Errorf("aviso repetido na mensagem: %q", w)
		}
		vistos[w] = true
	}
}

// entreAspas devolve o que está entre aspas depois do prefixo, e falha o teste
// quando o prefixo não está lá: o formato do Aviso é o que o ticket decidiu, e
// um Aviso sem a pasta é a metade que faltava antes dele.
func entreAspas(t *testing.T, texto, prefixo string) string {
	t.Helper()
	_, depois, ok := strings.Cut(texto, prefixo)
	if !ok {
		t.Fatalf("aviso %q não traz %q", texto, prefixo)
	}
	valor, _, ok := strings.Cut(depois, `"`)
	if !ok {
		t.Fatalf("aviso %q não fecha as aspas depois de %q", texto, prefixo)
	}
	return valor
}

// campoDoLog lê um campo chave=valor de uma linha do logger. O valor vai até o
// próximo campo (" <algo>=") e não até o próximo espaço: caminho do Windows com
// espaço no meio é o caso comum, não a exceção.
func campoDoLog(linha, chave string) string {
	_, depois, ok := strings.Cut(linha, chave)
	if !ok {
		return ""
	}
	for i := 1; i < len(depois); i++ {
		if depois[i] != '=' {
			continue
		}
		if corte := strings.LastIndex(depois[:i], " "); corte > 0 {
			return depois[:corte]
		}
	}
	return strings.TrimSpace(depois)
}
