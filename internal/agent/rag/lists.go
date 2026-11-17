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
// This set is closed by contract and is never derived from the database at
// runtime. Its protocol and daemon values do mirror, by hand, the labels the
// frr_docs rows carry — one per FRR user-guide file the chucker ingested —
// because a filter value works only through an exact WHERE match: a value with
// no rows hides the whole corpus, and a label missing here is reachable only
// through an empty value. Re-ingesting a new doc means adding its label here
// (decision: .agent/classifier/issues/10-listas-de-filtro-alinhadas-ao-corpus.md).
// Descriptions tell same-daemon siblings apart (bgp, evpn, bmp, bgp-linkstate
// are all bgpd; ospf and ospf-fundamentals are both ospfd).
func Lists() []classifier.List {
	return []classifier.List{
		{
			Name: "protocol",
			Values: []classifier.Value{
				{Name: "bgp", Description: "BGP in FRR: neighbors, address families, policy, L3VPN"},
				{Name: "evpn", Description: "BGP EVPN / VXLAN: L2VNI, L3VNI, type-2 and type-5 routes"},
				{Name: "bmp", Description: "BGP Monitoring Protocol: exporting BGP data to a BMP collector"},
				{Name: "bgp-linkstate", Description: "BGP-LS: link-state topology distributed over BGP"},
				{Name: "ospf", Description: "OSPFv2 in FRR: configuration and commands"},
				{Name: "ospf-fundamentals", Description: "OSPF theory: link-state routing, LSA types, LSDB, areas, SPF"},
				{Name: "ospf6", Description: "OSPFv3 / IPv6"},
				{Name: "isis", Description: "IS-IS"},
				{Name: "openfabric", Description: "OpenFabric: IS-IS based fabric routing"},
				{Name: "ldp", Description: "MPLS LDP"},
				{Name: "bfd", Description: "BFD"},
				{Name: "babel", Description: "Babel routing protocol"},
				{Name: "eigrp", Description: "EIGRP"},
				{Name: "nhrp", Description: "NHRP / DMVPN"},
				{Name: "basic", Description: "common FRR commands: VTY access, passwords, logging, hostname, config files"},
				{Name: "filter", Description: "access-lists and prefix-lists"},
				{Name: "ipv6", Description: "IPv6 router advertisement / neighbor discovery"},
				{Name: "kernel", Description: "kernel interface: netlink, sysctl"},
				{Name: "nexthop-groups", Description: "nexthop groups / ECMP"},
				{Name: "affinitymap", Description: "affinity maps: link administrative groups"},
				{Name: "extlog", Description: "extended syslog targets"},
				{Name: "frr-reload", Description: "the frr-reload.py script"},
				{Name: "grpc", Description: "gRPC northbound interface"},
				{Name: "mgmt", Description: "mgmtd: management daemon, YANG datastores"},
			},
		},
		{
			Name: "daemon",
			Values: []classifier.Value{
				{Name: "bgpd"},
				{Name: "ospfd"},
				{Name: "ospf6d"},
				{Name: "isisd"},
				{Name: "fabricd"},
				{Name: "ldpd"},
				{Name: "bfdd"},
				{Name: "babeld"},
				{Name: "eigrpd"},
				{Name: "nhrpd"},
				{Name: "mgmtd"},
				{Name: "zebra", Description: "general routing / RIB / kernel, and the common FRR commands, filters and logging"},
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
			// The Classificador reads only the user's input, so each
			// description says how that Subagente's request arrives, not
			// what the Subagente does — its limits live in its AGENT.md and
			// in deniedTools (decision:
			// .agent/generico/issues/09-descricoes-da-lista-subagent.md).
			Values: []classifier.Value{
				{Name: "builder", Description: "Asks for a new FRR configuration to be written — create, write, generate, from scratch — from requirements such as AS, router-id, neighbors or networks"},
				{Name: "modifier", Description: "Asks to change a configuration file that already exists — add, remove, replace, fix or enable something in a named file such as frr.conf"},
				{Name: "illustrator", Description: "Asks for a picture — draw, diagram, visualize — of a topology, a configuration or a protocol flow"},
				{Name: "teacher", Description: "Asks to understand something — what, why or how a concept, command, behavior or existing file works — or what this assistant is and does"},
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
