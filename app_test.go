package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

func newTestDB(ctx context.Context) (string, *postgres.PostgresContainer) {
	const testInfo = "bdapi-test"
	username, password, dbName := testInfo, testInfo, testInfo
	cont, errContainer := postgres.RunContainer(
		ctx,
		testcontainers.WithImage("docker.io/postgres:15-bullseye"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(wait.
			ForLog("database system is ready to accept connections").
			WithOccurrence(2)),
	)

	if errContainer != nil {
		panic(errContainer)
	}

	port, _ := cont.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName)

	return dsn, cont
}

func TestApp(t *testing.T) {
	t.Parallel()

	if key, found := os.LookupEnv("BDAPI_STEAM_API_KEY"); found && key != "" {
		if errKey := steamweb.SetKey(key); errKey != nil {
			panic(errKey)
		}
	}

	ctx := context.Background()
	dsn, databaseContainer := newTestDB(ctx)
	conf := appConfig{
		ListenAddr:               "",
		SteamAPIKey:              "",
		DSN:                      dsn,
		RunMode:                  "test",
		SourcebansScraperEnabled: false,
		ProxiesEnabled:           false,
		Proxies:                  nil,
		PrivateKeyPath:           "",
		EnableCache:              false,
		CacheDir:                 ".",
	}
	logger := zap.NewNop()

	t.Cleanup(func() {
		if errTerm := databaseContainer.Terminate(ctx); errTerm != nil {
			t.Error("Failed to terminate test container")
		}
	})

	db, errStore := newStore(ctx, logger, dsn)
	if errStore != nil {
		panic(errStore)
	}

	app := NewApp(logger, conf, db, &nopCache{}, newProxyManager(logger))

	t.Run("apiTestGans", apiTestGans(app))             //nolint:paralleltest
	t.Run("apiTestSummary", apiTestSummary(app))       //nolint:paralleltest
	t.Run("apiTestGetprofile", apiTestGetprofile(app)) //nolint:paralleltest
}

func apiTestGans(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		if !steamid.KeyConfigured() {
			t.Skip("BDAPI_STEAM_API_KEY not set")
		}

		sid := testIDb4nny

		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bans?steam_id=%d", testIDb4nny.Int64()), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		body, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var banState []steamweb.PlayerBanState

		require.NoError(t, json.Unmarshal(body, &banState))
		require.Equal(t, sid, banState[0].SteamID)
	}
}

func apiTestSummary(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		if !steamid.KeyConfigured() {
			t.Skip("BDAPI_STEAM_API_KEY not set")
		}

		sid := testIDb4nny
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/summary?steam_id=%d", testIDb4nny.Int64()), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		data, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var bs []steamweb.PlayerSummary

		require.NoError(t, json.Unmarshal(data, &bs))
		require.Equal(t, sid, bs[0].SteamID)
	}
}

func apiTestGetprofile(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		if !steamid.KeyConfigured() {
			t.Skip("BDAPI_STEAM_API_KEY not set")
		}

		sid := testIDb4nny

		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/profile?steam_id=%d", testIDb4nny.Int64()), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		data, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var profile Profile

		require.NoError(t, json.Unmarshal(data, &profile))
		require.Equal(t, steamweb.EconBanNone, profile.BanState.EconomyBan)
		require.Equal(t, sid, profile.Summary.SteamID)
	}
}
