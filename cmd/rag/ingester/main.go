package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/internal/rag"
	"github.com/jackc/pgx/v5"
)

func main() {
	log.Println("Starting ingest of embedding values.")
	dsn := flag.String("dsn", "postgres://postgres:pass@localhost:5432/frr_rag", "postgres://postgres:pass@localhost:5432/frr_rag")
	dir := flag.String("dir", "", "./resources/frrounting/chunks")
	flag.Parse()

	if *dir == "" {
		log.Fatalln("No dir path was given.")
	}

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Read files from %s\n", *dir)

	ctx := context.Background()

	chunks := []rag.Chunk{}

	for _, entry := range entries {
		// Filter out subdirectories if you only want files
		if !entry.IsDir() {
			fp := filepath.Join(*dir, entry.Name())
			log.Printf("Loading chunks from %s", fp)
			c, err := LoadChunks(fp)
			if err != nil {
				log.Fatal(err)
			}
			chunks = append(chunks, c...)
		}
	}

	conn, err := pgx.Connect(
		ctx,
		*dsn,
	)
	if err != nil {
		log.Println("Failed to connect to database.")
		log.Fatal(err)
	}

	log.Println("Connected to database.")
	defer conn.Close(ctx)

	if err := Upsert(ctx, conn, chunks, 64); err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"OK — %d chunks carregados/atualizados em frr_docs.\n",
		len(chunks),
	)
}
