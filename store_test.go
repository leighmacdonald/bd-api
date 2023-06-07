package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testCtx := context.Background()
	dbName := fmt.Sprintf("test-db-%d", time.Now().Second())
	username := "gbans-test"
	password := "gbans-test"

	container, errContainer := postgres.RunContainer(
		testCtx,
		testcontainers.WithImage("docker.io/postgres:15-bullseye"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername("gbans-test"),
		postgres.WithPassword("gbans-test"),
		testcontainers.WithWaitStrategy(wait.
			ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(5*time.Second)),
	)
	if errContainer != nil {
		logger.Fatal("Failed to setup test db", zap.Error(errContainer))
		os.Exit(2)
	}
	//host, _ := container.Host(context.Background())
	port, _ := container.MappedPort(context.Background(), "5432")
	config := Config{
		DSN: fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName),
	}
	defer func() {
		if errTerm := container.Terminate(testCtx); errTerm != nil {
			logger.Error("Failed to terminate test container")
		}
	}()
	s, errStore := newStore(config.DSN)
	if errStore != nil {
		logger.Error("Failed to setup test db", zap.Error(errStore))
		os.Exit(2)
	}
	if errMigrate := s.migrate(); errMigrate != nil {
		logger.Fatal("Failed to migrate test database", zap.Error(errMigrate))
	}
	m.Run()
}

func TestStore(t *testing.T) {
	require.True(t, false)
}
