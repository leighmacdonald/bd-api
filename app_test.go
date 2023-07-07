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
	"time"

	"github.com/leighmacdonald/bd-api/models"
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
		if errKey := steamid.SetKey(key); errKey != nil {
			panic(errKey)
		}

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
		LogLevel:                 "info",
		LogFileEnabled:           false,
		LogFilePath:              "",
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

	if !steamid.KeyConfigured() {
		t.Skip("BDAPI_STEAM_API_KEY not set")
	}

	t.Run("apiTestBans", apiTestBans(app))                   //nolint:paralleltest
	t.Run("apiTestSummary", apiTestSummary(app))             //nolint:paralleltest
	t.Run("apiTestGetProfile", apiTestGetprofile(app))       //nolint:paralleltest
	t.Run("apiTestGetSourcebans", apiTestGetSourcebans(app)) //nolint:paralleltest
}

func apiTestBans(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		sids := steamid.Collection{testIDb4nny, testIDCamper}

		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bans?steamids=%s", SteamIDStringList(sids)), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		body, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var banStates []steamweb.PlayerBanState

		require.NoError(t, json.Unmarshal(body, &banStates))
		require.Equal(t, len(sids), len(banStates))
	}
}

func apiTestSummary(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		sids := steamid.Collection{testIDb4nny, testIDCamper}

		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/summary?steamids=%s", SteamIDStringList(sids)), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		data, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var summaries []steamweb.PlayerSummary

		require.NoError(t, json.Unmarshal(data, &summaries))
		require.Equal(t, len(sids), len(summaries))
	}
}

func apiTestGetprofile(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		sids := steamid.Collection{testIDb4nny, testIDCamper}

		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/profile?steamids=%s", SteamIDStringList(sids)), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		data, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var (
			profiles []Profile
			validIds steamid.Collection
		)

		require.NoError(t, json.Unmarshal(data, &profiles))

		for _, sid := range sids {
			for _, profile := range profiles {
				if profile.Summary.SteamID == sid {
					require.Equal(t, steamweb.EconBanNone, profile.BanState.EconomyBan)
					require.NotEqual(t, "", profile.Summary.PersonaName)
					require.Equal(t, sid, profile.Summary.SteamID)
					require.Equal(t, sid, profile.BanState.SteamID)

					validIds = append(validIds, profile.Summary.SteamID)
				}
			}
		}

		require.EqualValues(t, sids, validIds)
	}
}

func createTestSourcebansRecord(t *testing.T, app *App) models.SbBanRecord {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	curTime := time.Now()

	player := newPlayerRecord(testIDb4nny)
	if errPlayer := app.db.playerGetOrCreate(ctx, testIDb4nny, &player); errPlayer != nil {
		t.Error(errPlayer)
	}

	site := NewSBSite(fmt.Sprintf("Test %s", curTime))
	if errSave := app.db.sbSiteSave(ctx, &site); errSave != nil {
		t.Error(errSave)
	}

	record := newRecord(site, testIDb4nny, "Name Goes Here", "Smelly",
		curTime.AddDate(-1, 0, 0), time.Hour*24, false)
	record.CreatedOn = curTime

	if errSave := app.db.sbBanSave(ctx, &record); errSave != nil {
		t.Error(errSave)
	}

	return record
}

func apiTestGetSourcebans(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		record := createTestSourcebansRecord(t, app)
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sourcebans/%s", record.SteamID), nil)
		recorder := httptest.NewRecorder()

		app.router.ServeHTTP(recorder, request)

		data, err := io.ReadAll(recorder.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		var banRecords []models.SbBanRecord

		require.NoError(t, json.Unmarshal(data, &banRecords))
		require.Equal(t, record.BanID, banRecords[0].BanID)
		require.Equal(t, record.SiteName, banRecords[0].SiteName)
		require.Equal(t, record.SiteID, banRecords[0].SiteID)
		require.Equal(t, record.PersonaName, banRecords[0].PersonaName)
		require.Equal(t, record.SteamID, banRecords[0].SteamID)
		require.Equal(t, record.Reason, banRecords[0].Reason)
		require.Equal(t, record.Duration, banRecords[0].Duration)
		require.Equal(t, record.Permanent, banRecords[0].Permanent)
	}
}
