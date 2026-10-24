package rag

import "github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"

// Lists is the single source of truth for the RAG caller's lists: the dimensions
// selected from the input. The first three become frr_docs filters; "agent" is
// a routing dimension and is intentionally excluded from FiltersFrom.
// Prompts derive their vocabulary from here rather than from copies.
// Decision: .agent/classifier/issues/03-como-o-modo-rag-alimenta-a-busca.md.
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
			Name: "agent",
			Values: []classifier.Value{
				{Name: "builder", Description: "Agent that creates config files"},
				{Name: "modifier", Description: "Agent that modifies config files"},
				{Name: "teacher", Description: "Agent that explain things"},
			},
		},
	}
}

// FiltersFrom converts classifier output (list name to canonical value) into
// the retrieval filter set. Filter names match frr_docs columns, and "" means
// no filter (the SQL NULL). The "agent" list is routing metadata, not a filter.
func FiltersFrom(result map[string]string) Filters {
	return Filters{
		Protocol:  result["protocol"],
		Daemon:    result["daemon"],
		ChunkType: result["chunk_type"],
	}
}
