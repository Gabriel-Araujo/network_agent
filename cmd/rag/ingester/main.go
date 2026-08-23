package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/jackc/pgx/v5"
)

var log = logger.Named("INGESTER")

func main() {
	// Utilitário curto: console basta. LOG_FILE liga o arquivo sob demanda.
	logger.Init(logger.ConfigFromEnv())

	log.Info("iniciando ingestão de embeddings")
	dsn := flag.String("dsn", "postgres://postgres:pass@localhost:5432/frr_rag", "postgres://postgres:pass@localhost:5432/frr_rag")
	dir := flag.String("dir", "", "./resources/frrounting/chunks")
	flag.Parse()

	if *dir == "" {
		log.Fatal("nenhum diretório foi informado (-dir)")
	}

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("erro ao ler %s: %v", *dir, err)
	}
	log.Info("diretório lido", "dir", *dir, "entries", len(entries))

	ctx := context.Background()

	chunks := []rag.Chunk{}

	for _, entry := range entries {
		// Filter out subdirectories if you only want files
		if !entry.IsDir() {
			fp := filepath.Join(*dir, entry.Name())
			log.Info("carregando chunks", "path", fp)
			c, err := LoadChunks(fp)
			if err != nil {
				log.Fatalf("erro ao carregar chunks de %s: %v", fp, err)
			}
			chunks = append(chunks, c...)
		}
	}

	conn, err := pgx.Connect(
		ctx,
		*dsn,
	)
	if err != nil {
		log.Fatalf("falha ao conectar ao banco: %v", err)
	}

	log.Info("conectado ao banco")
	defer conn.Close(ctx)

	if err := Upsert(ctx, conn, chunks, 64); err != nil {
		log.Fatalf("erro no upsert: %v", err)
	}

	log.Info("ingestão concluída", "chunks", len(chunks), "table", "frr_docs")
}
