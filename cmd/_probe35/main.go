package main

// Throwaway probe (ticket 35): drives the real RepairMermaid against LM Studio
// with the exact block that failed the live repair on 2026-11-15, proving the
// fenced-JSON tolerance round-trips with the real model. Removed after the run.

import (
	"context"
	"fmt"
	"os"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
)

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("lendo o bloco:", err)
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("config:", err)
		os.Exit(1)
	}
	ag, err := agent.Connect(cfg, "")
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}
	res, err := ag.RepairMermaid(context.Background(), []agent.MermaidFailure{{
		Index:  1,
		Source: string(raw),
		Error:  os.Args[2],
	}})
	if err != nil {
		fmt.Println("RepairMermaid ERRO:", err)
		os.Exit(1)
	}
	if res.Usage != nil {
		fmt.Printf("uso: entrada=%d saida=%d\n", res.Usage.InputTokens, res.Usage.OutputTokens)
	}
	for _, r := range res.Replacements {
		fmt.Printf("=== substituto %d (%d chars) ===\n%s\n", r.Index, len(r.Source), r.Source)
	}
}
