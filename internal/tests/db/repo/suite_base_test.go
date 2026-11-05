//go:build integration

package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/MishkaGql/template/internal/core/infrastructure/trip"
	"github.com/MishkaGql/template/internal/tests/db/dbtest"
)

type BaseSuite struct {
	suite.Suite
	Pool     *pgxpool.Pool
	TripRepo *trip.Repository
}

func (s *BaseSuite) SetupSuite() {
	s.T().Helper()

	require.NotNil(s.T(), pool, "Test main is not initialized")

	s.Pool = pool
	s.TripRepo = trip.NewRepository(s.Pool, dbtest.QueryTimeout)
}

func (s *BaseSuite) SetupTest() {
	s.T().Helper()
	s.CleanTables()
}

func (s *BaseSuite) CleanTables() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), dbtest.QueryTimeout)
	defer cancel()

	_, err := s.Pool.Exec(ctx, `TRUNCATE trips, trip_status_history RESTART IDENTITY CASCADE`)
	require.NoError(s.T(), err, "clean all tables")
}

func (s *BaseSuite) Context() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), dbtest.QueryTimeout)
	s.T().Cleanup(cancel)

	return ctx
}
