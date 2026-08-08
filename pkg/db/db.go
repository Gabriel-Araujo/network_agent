package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbOnce sync.Once
	dbPtr  *DB
	dbErr  error
)

// OpenDB devolve o singleton do pool para o DSN dado. Se o DSN mudar
// entre chamadas, um novo pool é criado.
func OpenDB(ctx context.Context, dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL vazio: configure no .env")
	}

	if dbPtr != nil && dbPtr.dsn == dsn {
		if dbPtr.Pool == nil {
			return nil, fmt.Errorf("pool anterior falhou ao conectar")
		}
		return dbPtr, nil
	}

	var s *DB
	var err error
	dbOnce.Do(func() {
		pool, _err := pgxpool.New(ctx, dsn)
		if _err != nil {
			err = _err
			return
		}
		if _err = pool.Ping(ctx); _err != nil {
			err = _err
			return
		}
		s = &DB{Pool: pool, dsn: dsn}
	})

	dbErr = err
	dbPtr = s
	if dbErr != nil || dbPtr == nil {
		return nil, fmt.Errorf("abrindo pool pgx: %w", dbErr)
	}
	return dbPtr, nil
}
