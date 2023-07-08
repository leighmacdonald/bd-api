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
	"github.com/pkg/errors"
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

	t.Run("apiTestBans", apiTestBans(app))                     //nolint:paralleltest
	t.Run("apiTestSummary", apiTestSummary(app))               //nolint:paralleltest
	t.Run("apiTestGetProfile", apiTestGetProfile(app))         //nolint:paralleltest
	t.Run("apiTestGetSourcebans", apiTestGetSourcebans(app))   //nolint:paralleltest
	t.Run("apiTestGetFriends", apiTestGetFriends(app))         //nolint:paralleltest
	t.Run("apiTestInvalidQueries", apiTestInvalidQueries(app)) //nolint:paralleltest
}

func generateIds(count int) steamid.Collection {
	var collection steamid.Collection

	for i := 0; i < count; i++ {
		collection = append(collection, steamid.RandSID64())
	}

	return collection
}

//nolint:unparam
func testReq(t *testing.T, app *App, method string, path string, target any) error {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()

	app.router.ServeHTTP(recorder, request)

	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		return errors.Wrap(err, "Failed to read test response body")
	}

	require.NoError(t, json.Unmarshal(body, &target))

	return nil
}

func apiTestBans(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			var (
				sids      = steamid.Collection{testIDb4nny, testIDCamper}
				banStates []steamweb.PlayerBanState
				path      = fmt.Sprintf("/bans?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &banStates))
			require.Equal(t, len(sids), len(banStates))
		})
		t.Run("tooManyError", func(t *testing.T) {
			t.Parallel()

			var (
				sids = generateIds(maxResults + 10)
				err  apiErr
				path = fmt.Sprintf("/bans?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &err))
			require.Equal(t, err.Error, ErrTooMany.Error())
		})
	}
}

func apiTestInvalidQueries(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("invalidParams", func(t *testing.T) {
			t.Parallel()

			var (
				err  apiErr
				path = "/summary?blah"
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &err))
			require.Equal(t, err.Error, ErrInvalidQueryParams.Error())
		})
		t.Run("invalidSteamID", func(t *testing.T) {
			t.Parallel()

			var (
				err  apiErr
				path = "/summary?steamids=ABC,12X"
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &err))
			require.Equal(t, err.Error, ErrInvalidSteamID.Error())
		})
		t.Run("tooManyRequested", func(t *testing.T) {
			t.Parallel()

			var (
				sids = generateIds(maxResults + 10)
				err  apiErr
				path = fmt.Sprintf("/summary?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &err))
			require.Equal(t, err.Error, ErrTooMany.Error())
		})
	}
}

func apiTestSummary(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			var (
				sids      = steamid.Collection{testIDb4nny, testIDCamper}
				summaries []steamweb.PlayerSummary
				path      = fmt.Sprintf("/summary?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &summaries))
			require.Equal(t, len(sids), len(summaries))
		})
	}
}

func apiTestGetFriends(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			sids := steamid.Collection{testIDb4nny, testIDCamper}

			request := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/friends?steamids=%s", SteamIDStringList(sids)), nil)
			recorder := httptest.NewRecorder()

			app.router.ServeHTTP(recorder, request)

			data, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Errorf("expected error to be nil got %v", err)
			}

			var friends friendMap

			require.NoError(t, json.Unmarshal(data, &friends))
			require.True(t, len(friends[sids[0]]) > 0)
			require.True(t, len(friends[sids[1]]) == 0)
		})
	}
}

func apiTestGetProfile(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		var (
			sids     = steamid.Collection{testIDb4nny, testIDCamper}
			path     = fmt.Sprintf("/profile?steamids=%s", SteamIDStringList(sids))
			profiles []Profile
			validIds steamid.Collection
		)

		require.NoError(t, testReq(t, app, http.MethodGet, path, &profiles))

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

	site := NewSBSite(models.Site(fmt.Sprintf("Test %s", curTime)))
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
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			var (
				record     = createTestSourcebansRecord(t, app)
				banRecords []models.SbBanRecord
				path       = fmt.Sprintf("/sourcebans/%s", record.SteamID)
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &banRecords))
			require.Equal(t, record.BanID, banRecords[0].BanID)
			require.Equal(t, record.SiteName, banRecords[0].SiteName)
			require.Equal(t, record.SiteID, banRecords[0].SiteID)
			require.Equal(t, record.PersonaName, banRecords[0].PersonaName)
			require.Equal(t, record.SteamID, banRecords[0].SteamID)
			require.Equal(t, record.Reason, banRecords[0].Reason)
			require.Equal(t, record.Duration, banRecords[0].Duration)
			require.Equal(t, record.Permanent, banRecords[0].Permanent)
		})
		t.Run("emptyResults", func(t *testing.T) {
			t.Parallel()

			var (
				steamID    = steamid.RandSID64()
				banRecords []models.SbBanRecord
				path       = fmt.Sprintf("/sourcebans/%s", steamID)
			)

			require.NoError(t, testReq(t, app, http.MethodGet, path, &banRecords))
			require.Equal(t, []models.SbBanRecord{}, banRecords)
		})
	}
}
