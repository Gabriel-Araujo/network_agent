package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
)

// Drive ao vivo do ticket 22 (.agent/ui/issues/22-caminho-no-sucesso-do-writefile.md).
//
// O mesmo harness dos drives dos tickets 07, 11, 12, 13 e 19 (appAoVivo,
// turnoDoDrive), ligado por UI_DRIVE_AO_VIVO=1.
//
// O ticket não exige drive — o que o modelo faz com a linha é fato do modelo —,
// mas o defeito que ele conserta é prosa: o drive do ticket 11 mediu o builder
// dizendo "criado com sucesso no diretório atual" e hedgeando "substituiu o
// arquivo existente (se houver)", as duas únicas coisas que a saída de sucesso
// não carregava. Com a primeira linha carregando o absoluto contido, a prova é
// a resposta nomeando a pasta escolhida.
//
//	LOG_FILE=.agent/ui/assets/22-caminho-no-sucesso UI_DRIVE_AO_VIVO=1 \
//	  go test ./cmd/ui/ -run TestDriveAoVivoCaminhoNoSucesso -v -count=1 -timeout 60m

// TestDriveAoVivoCaminhoNoSucesso roda dois turnos na mesma Conversa, um por
// direção da saída: o primeiro cria o frr.conf (primeira linha
// "Created new file:"), o segundo o substitui ("Replaced existing file:"). O
// que ele mede é o que o usuário lê — a resposta nomeia a pasta escolhida, e
// não "o diretório atual".
func TestDriveAoVivoCaminhoNoSucesso(t *testing.T) {
	a := appAoVivo(t)

	// A escolha entra na configuração em memória, como nos drives dos tickets
	// 11 e 19: o config.json do usuário não é cobaia de drive.
	escolhida := t.TempDir()
	cfg, err := readConfig()
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	config.Normalize(&cfg)
	cfg.Workspace = escolhida
	a.reconnect(cfg)
	esperaConectar(t, a)
	t.Logf("Workspace escolhido: %s", escolhida)

	conv, err := a.CreateConversation("drive ao vivo do caminho no sucesso")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// O pedido do drive do ticket 11 mais o hostname: sem o router-id e sem o
	// prefixo o builder pergunta em vez de gravar (medido lá), e medido aqui
	// ele pergunta também pelo hostname. O que este drive precisa é da escrita,
	// não da conversa.
	const pedido = "Crie um arquivo de configuração FRR novo, chamado frr.conf, " +
		"para o roteador de hostname R1, com BGP no AS 65001, router-id 10.0.0.1, " +
		"anunciando o prefixo 192.168.10.0/24, com o vizinho 10.0.0.2 no AS 65002. " +
		"Grave o arquivo."

	msg := turnoDoDrive(t, a, conv.ID, "builder-novo", pedido)
	alvo := filepath.Join(escolhida, "frr.conf")
	if _, err := os.Stat(alvo); err != nil {
		t.Fatalf("o builder não gravou o frr.conf na pasta escolhida (%v): sem escrita não há saída de sucesso para medir.\nResposta:\n%s", err, msg.Content)
	}
	conferePasta(t, "builder-novo", msg.Content, escolhida)

	// A outra direção, na MESMA Conversa: o arquivo agora existe, logo uma
	// reescrita inteira cai no caminho da substituição. O pedido diz "do zero"
	// e "inteiro" de propósito — "acrescente o vizinho X" é trabalho de
	// EditFile, e o drive mediu que é para lá que ele vai (Subagente modifier),
	// onde o sucesso tem outro contrato e este ticket não toca.
	msg = turnoDoDrive(t, a, conv.ID, "builder-substitui",
		"Gere o frr.conf inteiro de novo, do zero, substituindo por completo o arquivo atual: "+
			"mesmo AS 65001 e router-id 10.0.0.1, mesmo prefixo 192.168.10.0/24, "+
			"agora com os vizinhos 10.0.0.2 no AS 65002 e 10.0.0.3 no AS 65003.")
	conferePasta(t, "builder-substitui", msg.Content, escolhida)
}

// conferePasta mede a resposta nas duas direções do defeito: ela nomeia a pasta
// contra a qual a tool resolveu, e não a descreve com as palavras que o modelo
// não tem como saber. O caminho é checado nas duas grafias porque a tool
// devolve com barra para frente (filepath.ToSlash) e o modelo pode reescrever.
func conferePasta(t *testing.T, rotulo, resposta, escolhida string) {
	t.Helper()

	nomeia := strings.Contains(resposta, filepath.ToSlash(escolhida)) ||
		strings.Contains(resposta, escolhida)
	if !nomeia {
		t.Errorf("[%s] a resposta não nomeia a pasta escolhida (%s):\n%s", rotulo, escolhida, resposta)
	} else {
		t.Logf("[%s] a resposta nomeia a pasta escolhida:\n%s", rotulo, resposta)
	}

	// As frases que o drive do ticket 11 mediu. São fato do modelo, não da
	// ponte — mas são o defeito que este ticket existe para tirar da tela.
	for _, frase := range []string{"diretório atual", "diretorio atual", "current directory", "se houver", "if any"} {
		if strings.Contains(strings.ToLower(resposta), frase) {
			t.Errorf("[%s] a resposta ainda diz %q:\n%s", rotulo, frase, resposta)
		}
	}
}
