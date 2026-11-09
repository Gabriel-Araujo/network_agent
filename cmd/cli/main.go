package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/turn"
)

var log = logger.Named("MAIN")

func main() {
	cfg := logger.ConfigFromEnv()
	if cfg.File == "" {
		cfg.File = "logs"
	}
	logger.Init(cfg)

	config.SetCreator(createConfig)

	appCfg, err := config.Load()
	if err != nil {
		log.Fatalf("falha ao carregar a configuração (%s): %v", config.Path(), err)
	}

	agent, err := agentapi.Connect(appCfg)

	if err != nil {
		log.Fatalf("falha ao conectar ao LLM: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Network Agent started. Type your message (or 'exit' to quit):")
	reader := bufio.NewReader(os.Stdin)

	var transcript []history.Message

	// The composition's state between turns (the skip rule): turn.Run reads
	// it, the CLI only holds it — one opaque value for the one Conversation.
	var state turn.State

	for {
		fmt.Print("\nUser: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Error("falha ao ler a entrada", "err", err)
			continue
		}
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			break
		}
		if userInput == "" {
			continue
		}

		// The whole turn composition lives in turn.Run (generic
		// classification + routing, RAG mode, skip rule, retrieval,
		// recovered_context, the LLM call). This front only reads, prints
		// and keeps the transcript.
		out, next, err := turn.Run(ctx, agent, transcript, userInput, state)
		state = next

		// Display: one name=value line per generic list ("(vazio)" for
		// empty), then the reply. The lines print even when the LLM call
		// failed — the classification metadata is valid either way.
		if out.Generic != nil {
			for _, l := range rag.Lists() {
				value := out.Generic[l.Name]
				if value == "" {
					value = "(vazio)"
				}
				fmt.Printf("%s=%s\n", l.Name, value)
			}
		}
		for _, warning := range out.Warnings {
			log.Warn(warning)
		}
		if err != nil {
			log.Error("falha ao chamar o LLM", "err", err)
			continue
		}

		fmt.Printf("Agent: %s\n", out.Text)

		// The transcript keeps the composed prompt (input + the
		// recovered_context block): that is what the next turns re-send.
		transcript = append(transcript,
			history.Message{Role: history.RoleUser, Content: out.Prompt, Status: history.StatusDone},
			history.Message{Role: history.RoleAgent, Content: out.Text, Status: history.StatusDone},
		)
	}
}
