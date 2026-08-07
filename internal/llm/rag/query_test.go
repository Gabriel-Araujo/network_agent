package rag

import (
	"errors"
	"testing"
)

const sampleBriefing = `# Intent Analysis — FRR RAG

**Timestamp:** 2026-07-14T15:20:00-03:00
**Original query:** "meu ospf não converge entre o core (R1) e o edge (R2)"

## 1. Classification
- **Intent type:** troubleshooting
- **Protocol(s):** ospf (primary), bgp (context/possible interaction)
- **Daemon(s):** ospfd, bgpd

## 2. Devices
| Label | Role | Platform | Daemon(s)/Protocol(s) |
|---|---|---|---|
| R1 | core | FRR | ospfd |

## 3. Connections
| From | To | Interface/Link | Protocol/Session |
|---|---|---|---|
| R1 | R2 | Gi0/1 | OSPF area 0 |

## 4. Problem/goal summary
The OSPF adjacency between R1 (core) and R2 (edge) isn't converging in area 0.

## 5. Execution pipeline
1. Confirm ospfd is enabled and running on R1 and R2.
2. Check basic L3 connectivity between R1 and R2 on Gi0/1.
3. If stuck in Exstart/Exchange, check for an MTU mismatch.

## 6. Rewritten queries for RAG
| # | Rewritten query | protocol | daemon | suggested chunk_type |
|---|---|---|---|---|
| 1 | "OSPF neighbor stuck Exstart Exchange state" | ospf | ospfd | concept |
| 2 | "OSPF interface authentication area configuration" | ospf | ospfd | command_reference |
| 3 | "BGP OSPF redistribution interaction" | bgp | bgpd | concept |
`

func TestParseQueriesFromBriefing(t *testing.T) {
	got, err := ParseQueriesFromBriefing([]byte(sampleBriefing))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %+v", len(got), got)
	}

	want := []QuerySuggestion{
		{Query: "OSPF neighbor stuck Exstart Exchange state", Protocol: "ospf", Daemon: "ospfd", ChunkType: "concept"},
		{Query: "OSPF interface authentication area configuration", Protocol: "ospf", Daemon: "ospfd", ChunkType: "command_reference"},
		{Query: "BGP OSPF redistribution interaction", Protocol: "bgp", Daemon: "bgpd", ChunkType: "concept"},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseQueriesFromBriefingQuotedQuery(t *testing.T) {
	// Queries devem ter aspas removidas; caso contrário, o filtro semântico falha.
	briefing := `## 6. Rewritten queries for RAG
| # | Rewritten query | protocol | daemon | suggested chunk_type |
|---|---|---|---|---|
| 1 | 'OSPF MTU mismatch' | ospf | ospfd | concept |
| 2 | "show ip ospf neighbor" | ospf | ospfd | command_reference |
`
	got, err := ParseQueriesFromBriefing([]byte(briefing))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Query != "OSPF MTU mismatch" {
		t.Errorf("query[0] = %q, want %q", got[0].Query, "OSPF MTU mismatch")
	}
	if got[1].Query != "show ip ospf neighbor" {
		t.Errorf("query[1] = %q, want %q", got[1].Query, "show ip ospf neighbor")
	}
}

func TestParseQueriesFromBriefingInvalidChunkType(t *testing.T) {
	// chunk_type inválido deve virar "" (sem filtro), não quebrar o parse.
	briefing := `## 6. Rewritten queries for RAG
| # | Rewritten query | protocol | daemon | suggested chunk_type |
|---|---|---|---|---|
| 1 | "OSPF MTU mismatch" | ospf | ospfd | bogus |
| 2 | "OSPF cost" | ospf | ospfd | concept |
`
	got, err := ParseQueriesFromBriefing([]byte(briefing))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].ChunkType != "" {
		t.Errorf("chunk_type[0] = %q, want %q", got[0].ChunkType, "")
	}
	if got[1].ChunkType != "concept" {
		t.Errorf("chunk_type[1] = %q, want %q", got[1].ChunkType, "concept")
	}
}

func TestParseQueriesFromBriefingMissingSection(t *testing.T) {
	briefing := `## 4. Problem/goal summary
Nothing here about queries.
`
	_, err := ParseQueriesFromBriefing([]byte(briefing))
	if !errors.Is(err, ErrNoQueriesTable) {
		t.Fatalf("err = %v, want ErrNoQueriesTable", err)
	}
}
