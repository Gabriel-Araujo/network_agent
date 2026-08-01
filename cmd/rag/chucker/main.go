package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
)

func main() {
	path := flag.String("path", "", ".rst file path")
	protocol := flag.String("protocol", "", "protocol")
	daemon := flag.String("daemon", "", "frrounting daemon")

	flag.Parse()

	if *path == "" || *daemon == "" || *protocol == "" {
		log.Println("the following required flags are not set:")
		if *path == "" {
			log.Println("- path")
		}
		if *daemon == "" {
			log.Println("- daemon")
		}
		if *protocol == "" {
			log.Println("- protocol")
		}
		os.Exit(1)
	}

	chunks, err := chunkFile(
		*path, *daemon, *protocol,
		"https://github.com/FRRouting/frr/blob/master/doc/user/bgp.rst",
	)

	if err != nil {
		log.Panicf("Error while reading %s:\n%s", *path, err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(chunks); err != nil {
		log.Panicf("erro ao gerar JSON: %s", err)
		os.Exit(1)
	}

	targetDir := filepath.Dir(*path) + "/chunks"

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		log.Panic(err)
	}

	if err := os.WriteFile(targetDir+"/"+*protocol+".json", bytes.TrimRight(buf.Bytes(), "\n"), 0644); err != nil {
		log.Fatalf("erro ao escrever %s.json:\n%s", targetDir+"/"+*protocol, err)
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

	log.Printf("Total chunks: %d  (command_reference=%d, concept=%d)\n", len(chunks), nCmd, nConcept)

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
		log.Printf("Tokens por chunk — min=%d max=%d média=%.0f\n", minT, maxT, avg)
	}

}
