package db

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB encapsula o pool pgxpool com inicialização preguiçosa (lazy
// singleton). As buscas se repetem ao longo da conversação, então o pool
// é compartilhado entre chamadas.
type DB struct {
	Pool       *pgxpool.Pool
	dsn        string
	CheckedDim bool
}
