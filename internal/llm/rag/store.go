package rag

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// store encapsula o pool pgxpool com inicialização preguiçosa (lazy
// singleton). As buscas se repetem ao longo da conversação, então o pool
// é compartilhado entre chamadas.
type store struct {
	pool       *pgxpool.Pool
	dsn        string
	checkedDim bool
}

var (
	storeOnce sync.Once
	storePtr  *store
	storeErr  error
)

// openStore devolve o singleton do pool para o DSN dado. Se o DSN mudar
// entre chamadas, um novo pool é criado.
func openStore(ctx context.Context, dsn string) (*store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL vazio: configure no .env")
	}

	if storePtr != nil && storePtr.dsn == dsn {
		if storePtr.pool == nil {
			return nil, fmt.Errorf("pool anterior falhou ao conectar")
		}
		return storePtr, nil
	}

	var s *store
	var err error
	storeOnce.Do(func() {
		pool, perr := pgxpool.New(ctx, dsn)
		if perr != nil {
			err = perr
			return
		}
		if perr = pool.Ping(ctx); perr != nil {
			err = perr
			return
		}
		s = &store{pool: pool, dsn: dsn}
	})

	storeErr = err
	storePtr = s
	if storeErr != nil || storePtr == nil {
		return nil, fmt.Errorf("abrindo pool pgx: %w", storeErr)
	}
	return storePtr, nil
}
