package dbtest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/MishkaGql/template/migrations"
)

type Postgres struct {
	Container *postgres.PostgresContainer
	DSN       string
}

const (
	image        string        = "postgres:15-alpine"
	defaultDB    string        = "tripgo_test"
	pgDialect    goose.Dialect = "postgres"
	QueryTimeout time.Duration = 3 * time.Second
)

func Start(ctx context.Context) (*Postgres, error) {
	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase(defaultDB),
		postgres.WithUsername(defaultDB),
		postgres.WithPassword(defaultDB),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("run postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, errors.Join(fmt.Errorf("postgres connection string: %w", err), Terminate(container))
	}

	if err = migrate(ctx, dsn); err != nil {
		return nil, errors.Join(err, Terminate(container))
	}
	return &Postgres{Container: container, DSN: dsn}, nil
}

func (p *Postgres) InitPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, p.DSN)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pool: %w", err)
	}

	return pool, nil
}

func migrate(ctx context.Context, dsn string) (err error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open pgx conn: %w", err)
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration connection: %w", closeErr))
		}
	}()

	provider, err := goose.NewProvider(pgDialect, conn, migrations.FS)
	if err != nil {
		return fmt.Errorf("setup goose provider: %w", err)
	}

	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("goose apply migrations: %w", err)
	}

	return nil
}

func Terminate(container *postgres.PostgresContainer) error {
	if err := testcontainers.TerminateContainer(container); err != nil {
		return fmt.Errorf("terminate postgres container: %w", err)
	}

	return nil
}
