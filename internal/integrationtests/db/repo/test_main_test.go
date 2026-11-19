package repo

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/internal/integrationtests/db/dbtest"
)

const startContainerTimeout = time.Minute

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
		if err := dbtest.Terminate(pg.Container); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
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
