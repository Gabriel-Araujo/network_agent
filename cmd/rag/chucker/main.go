package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
)

var log = logger.Named("CHUCKER")

func main() {
	// Utilitário curto: console basta. LOG_FILE liga o arquivo sob demanda.
	logger.Init(logger.ConfigFromEnv())

	path := flag.String("path", "", ".rst file path")
	protocol := flag.String("protocol", "", "protocol")
	daemon := flag.String("daemon", "", "frrounting daemon")

	flag.Parse()

	if *path == "" || *daemon == "" || *protocol == "" {
		fmt.Fprintln(os.Stderr, "the following required flags are not set:")
		if *path == "" {
			fmt.Fprintln(os.Stderr, "- path")
		}
		if *daemon == "" {
			fmt.Fprintln(os.Stderr, "- daemon")
		}
		if *protocol == "" {
			fmt.Fprintln(os.Stderr, "- protocol")
		}
		os.Exit(1)
	}

	chunks, err := chunkFile(
		*path, *daemon, *protocol,
		"https://github.com/FRRouting/frr/blob/master/doc/user/bgp.rst",
	)

	if err != nil {
		log.Fatalf("erro ao ler %s: %v", *path, err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(chunks); err != nil {
		log.Fatalf("erro ao gerar JSON: %v", err)
	}

	targetDir := filepath.Dir(*path) + "/chunks"

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		log.Fatalf("erro ao criar %s: %v", targetDir, err)
	}

	if err := os.WriteFile(targetDir+"/"+*protocol+".json", bytes.TrimRight(buf.Bytes(), "\n"), 0644); err != nil {
		log.Fatalf("erro ao escrever %s.json: %v", targetDir+"/"+*protocol, err)
	}

	nCmd, nConcept := 0, 0
	tokens := make([]int, 0, len(chunks))
	for _, c := range chunks {
		switch c.ChunkType {
		case "command_reference":
			nCmd++
		case "concept":
			nConcept++
		}
		tokens = append(tokens, c.TokenCount)
	}

	log.Info("chunks gerados", "total", len(chunks), "command_reference", nCmd, "concept", nConcept)

	if len(tokens) > 0 {
		minT, maxT, sum := tokens[0], tokens[0], 0
		for _, t := range tokens {
			if t < minT {
				minT = t
			}
			if t > maxT {
				maxT = t
			}
			sum += t
		}
		avg := float64(sum) / float64(len(tokens))
		log.Infof("tokens por chunk — min=%d max=%d média=%.0f", minT, maxT, avg)
	}

}
