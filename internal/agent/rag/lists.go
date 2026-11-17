package rag

import "github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"

// SubagentListName names the routing list: the dimension that chooses which
// Subagente answers the turn. It rides the routing call alone, never the RAG
// call — switching Subagentes must not re-trigger the retrieval
// (decision: .agent/roteamento/issues/05-roteamento-lista-subagent-e-despacho.md).
const SubagentListName = "subagent"

// DefaultSubagent names the Subagente that answers when the routing
// classification names none: the Teacher. The entry point is never the author
// of a reply, so "nothing fits" is answered by one of the four rather than by
// nobody (decision: .agent/generico/issues/01-quem-responde-o-nenhum.md).
//
// The Teacher and not another of the four: it is the only one that does not
// lie out of domain — it can answer in prose that the input is not FRR, where
// builder, modifier and illustrator only know how to produce an artifact — and
// it is one of the two that carry the mermaid skill, so routing the empty value
// to it keeps the diagram block alive.
//
// It is a value of the SubagentListName List below, which is what
// TestSubagentConfigsOnDisk asserts: a default with no configuration on disk
// fails at the gate instead of at the turn.
const DefaultSubagent = "teacher"

// Lists is the single source of truth for the turn's classification
// dimensions: what gets selected from the input. The first three are the
// frr_docs filters (FilterLists rides the RAG call); SubagentListName is the
// routing dimension (SubagentList rides the routing call) and is intentionally
// excluded from FiltersFrom. No call carries all four — the two are disjoint
// questions — but this is where both derive from, and prompts derive their
// vocabulary from here rather than from copies.
// Decision: .agent/classifier/issues/03-como-o-modo-rag-alimenta-a-busca.md.
//
// Every value of SubagentListName is also the folder name of its Subagente
// configuration: resources/agents/<nome>/AGENT.md mounts the turn's system
// prompt. These rows are the single source of that name -> folder table.
//
// This set is closed by contract and must not be derived from the database,
// whose frr_docs contents vary with ingestion. Annotated spellings from the
// intent-analyser table ("ospf (v2)") become value descriptions.
func Lists() []classifier.List {
	return []classifier.List{
		{
			Name: "protocol",
			Values: []classifier.Value{
				{Name: "bgp"},
				{Name: "ospf", Description: "v2"},
				{Name: "ospf6", Description: "v3/IPv6"},
				{Name: "isis"},
				{Name: "rip"},
				{Name: "ripng"},
				{Name: "pim"},
				{Name: "ldp"},
				{Name: "vrrp"},
				{Name: "bfd"},
				{Name: "static"},
			},
		},
		{
			Name: "daemon",
			Values: []classifier.Value{
				{Name: "bgpd"},
				{Name: "ospfd"},
				{Name: "ospf6d"},
				{Name: "isisd"},
				{Name: "ripd"},
				{Name: "ripngd"},
				{Name: "pimd"},
				{Name: "pim6d"},
				{Name: "ldpd"},
				{Name: "vrrpd"},
				{Name: "bfdd"},
				{Name: "staticd"},
				{Name: "zebra", Description: "general routing / RIB / kernel"},
			},
		},
		{
			Name: "chunk_type",
			Values: []classifier.Value{
				{Name: ChunkTypeCommandReference, Description: "specific command syntax/behavior"},
				{Name: ChunkTypeConcept, Description: "general protocol behavior"},
			},
		},
		{
			Name: SubagentListName,
			Values: []classifier.Value{
				{Name: "builder", Description: "Request to create a new FRR configuration: writes a new config file, never edits an existing one"},
				{Name: "modifier", Description: "Request to change an FRR configuration file that already exists: edits that file, never creates one"},
				{Name: "illustrator", Description: "Request to draw or diagram a topology or a configuration: answers with a diagram, never prose"},
				{Name: "teacher", Description: "Request to explain a concept or an existing file: answers in text, never writes or edits a file"},
			},
		},
	}
}

// FilterLists returns the frr_docs filter lists: the only lists that ride the
// RAG call, and therefore the only keys the skip rule compares — the routing
// list is deliberately out, so changing Subagente does not re-trigger the
// retrieval. It is the filter prefix of Lists(), whose shape the seam tests pin.
func FilterLists() []classifier.List {
	return Lists()[:3]
}

// SubagentList returns the routing list alone: the whole question the turn's
// routing call carries. The two classifications of a turn are disjoint by
// construction — this list routes, FilterLists filters — which is what retired
// the conditional re-ask that used to ask this narrower question only after an
// empty answer (decision:
// .agent/generico/issues/02-caminho-de-classificacao-do-ponto-de-entrada.md).
// It is the routing complement of FilterLists, and the seam tests pin its
// position in Lists().
func SubagentList() classifier.List {
	return Lists()[3]
}

// FiltersFrom converts classifier output (list name to canonical value) into
// the retrieval filter set. Filter names match frr_docs columns, and "" means
// no filter (the SQL NULL). The routing list is routing metadata, not a filter.
func FiltersFrom(result map[string]string) Filters {
	return Filters{
		Protocol:  result["protocol"],
		Daemon:    result["daemon"],
		ChunkType: result["chunk_type"],
	}
}
