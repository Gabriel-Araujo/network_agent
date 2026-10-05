package rag

import "github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"

// Listas é a fonte única das Listas do chamador RAG: as dimensões que o
// Classificador escolhe por input. As três primeiras viram filtros da tabela
// frr_docs (os nomes são os das colunas, para o map[string]string sair sem
// tradução); a lista "agent" NÃO é coluna da frr_docs — não entra em
// FiltrosFrom, fica só no mapa da classificação.
// Os prompts que precisam do vocabulário derivam daqui — nunca de cópias.
// Decisão: .agent/classifier/issues/03-como-o-modo-rag-alimenta-a-busca.md.
//
// Conjunto fechado por contrato: NÃO derivar do banco (o conteúdo da frr_docs
// varia com a ingesta). As grafias anotadas da tabela do intent-analyser
// ("ospf (v2)") viram descrição dos valores, para o modelo desempatar.
func Listas() []classifier.Lista {
	return []classifier.Lista{
		{
			Nome: "protocol",
			Valores: []classifier.Valor{
				{Nome: "bgp"},
				{Nome: "ospf", Descricao: "v2"},
				{Nome: "ospf6", Descricao: "v3/IPv6"},
				{Nome: "isis"},
				{Nome: "rip"},
				{Nome: "ripng"},
				{Nome: "pim"},
				{Nome: "ldp"},
				{Nome: "vrrp"},
				{Nome: "bfd"},
				{Nome: "static"},
			},
		},
		{
			Nome: "daemon",
			Valores: []classifier.Valor{
				{Nome: "bgpd"},
				{Nome: "ospfd"},
				{Nome: "ospf6d"},
				{Nome: "isisd"},
				{Nome: "ripd"},
				{Nome: "ripngd"},
				{Nome: "pimd"},
				{Nome: "pim6d"},
				{Nome: "ldpd"},
				{Nome: "vrrpd"},
				{Nome: "bfdd"},
				{Nome: "staticd"},
				{Nome: "zebra", Descricao: "general routing / RIB / kernel"},
			},
		},
		{
			Nome: "chunk_type",
			Valores: []classifier.Valor{
				{Nome: ChunkTypeCommandReference, Descricao: "specific command syntax/behavior"},
				{Nome: ChunkTypeConcept, Descricao: "general protocol behavior"},
			},
		},
		{
			Nome: "agent",
			Valores: []classifier.Valor{
				{Nome: "construtor"},
				{Nome: "modificar"},
				{Nome: "professor"},
			},
		},
	}
}

// FiltrosFrom converte o resultado do Classificador (mapa nome-da-lista →
// valor canônico) no conjunto único de filtros do retrieval. Os nomes das
// Listas são os das colunas da frr_docs — sai sem tradução — e "" vira
// "sem filtro" (o NULL do SQL).
func FiltrosFrom(resultado map[string]string) Filtros {
	return Filtros{
		Protocol:  resultado["protocol"],
		Daemon:    resultado["daemon"],
		ChunkType: resultado["chunk_type"],
	}
}
