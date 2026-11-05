//go:build integration

package repo

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"

	"github.com/MishkaGql/template/internal/tests/db/dbtest"
)

const startContainerTimeout = 5 * time.Second

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), startContainerTimeout)
	defer cancel()

	pg, err := dbtest.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start test postgres: %v\n", err)
		return 1
	}

	defer func() {
		if termErr := testcontainers.TerminateContainer(pg.Container); termErr != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres container: %v\n", termErr)
		}
	}()

	pool, err = pg.InitPool(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create pool: %v\n", err)
		return 1
	}
	defer pool.Close()

	return m.Run()
}
