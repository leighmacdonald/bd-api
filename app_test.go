package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var errTestContainer = errors.New("failed to bring up test container")

var (
	testIDb4nny  = steamid.New("76561197970669109")
	testIDCamper = steamid.New("76561197992870439")
)

func newTestDB(ctx context.Context) (string, *postgres.PostgresContainer, error) {
	const testInfo = "bdapi-test"
	username, password, dbName := testInfo, testInfo, testInfo
	cont, errContainer := postgres.Run(
		ctx,
		"timescale/timescaledb-ha:pg15",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(wait.
			ForLog("database system is ready to accept connections").
			WithOccurrence(2)),
	)

	if errContainer != nil {
		return "", nil, errors.Join(errContainer, errTestContainer)
	}

	port, _ := cont.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName)

	return dsn, cont, nil
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

	dsn, databaseContainer, errDB := newTestDB(ctx)
	if errDB != nil {
		t.Skipf("Failed to bring up testcontainer db: %v", errDB)
	}

	// conf := appConfig{
	//	ListenAddr:               "",
	//	SteamAPIKey:              "",
	//	DSN:                      dsn,
	//	RunMode:                  "test",
	//	LogLevel:                 "info",
	//	LogFileEnabled:           false,
	//	LogFilePath:              "",
	//	SourcebansScraperEnabled: false,
	//	ProxiesEnabled:           false,
	//	Proxies:                  nil,
	//	PrivateKeyPath:           "",
	//	EnableCache:              false,
	//	CacheDir:                 ".",
	// }

	t.Cleanup(func() {
		if errTerm := databaseContainer.Terminate(ctx); errTerm != nil {
			t.Error("Failed to terminate test container")
		}
	})

	database, errStore := newStore(ctx, dsn)
	if errStore != nil {
		panic(errStore)
	}

	cacheHandler := &nopCache{}

	if !steamid.KeyConfigured() {
		t.Skip("BDAPI_STEAM_API_KEY not set")
	}

	router, err := createRouter(database, cacheHandler, appConfig{})
	require.NoError(t, err)

	t.Run("apiTestBans", apiTestBans(router))                             //nolint:paralleltest
	t.Run("apiTestSummary", apiTestSummary(router))                       //nolint:paralleltest
	t.Run("apiTestGetProfile", apiTestGetProfile(router))                 //nolint:paralleltest
	t.Run("apiTestGetSourcebans", apiTestGetSourcebans(router, database)) //nolint:paralleltest
	t.Run("apiTestGetFriends", apiTestGetFriends(router))                 //nolint:paralleltest
	t.Run("apiTestInvalidQueries", apiTestInvalidQueries(router))         //nolint:paralleltest
}

func generateIDs(count int) steamid.Collection {
	var collection steamid.Collection

	for i := 0; i < count; i++ {
		collection = append(collection, steamid.RandSID64())
	}

	return collection
}

//nolint:unparam
func testReq(t *testing.T, router *http.ServeMux, method string, path string, target any) error {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&target))

	return nil
}

func apiTestBans(router *http.ServeMux) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			var (
				sids      = steamid.Collection{testIDb4nny, testIDCamper}
				banStates []steamweb.PlayerBanState
				path      = fmt.Sprintf("/bans?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &banStates))
			require.Equal(t, len(sids), len(banStates))
		})
		t.Run("tooManyError", func(t *testing.T) {
			t.Parallel()

			var (
				sids = generateIDs(maxResults + 10)
				err  apiErr
				path = fmt.Sprintf("/bans?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &err))
			require.Equal(t, err.Error, errTooMany.Error())
		})
	}
}

func apiTestInvalidQueries(router *http.ServeMux) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("invalidParams", func(t *testing.T) {
			t.Parallel()

			var (
				err  apiErr
				path = "/summary?blah"
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &err))
			require.Equal(t, err.Error, errInvalidQueryParams.Error())
		})
		t.Run("invalidSteamID", func(t *testing.T) {
			t.Parallel()

			var (
				err  apiErr
				path = "/summary?steamids=ABC,12X"
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &err))
			require.Equal(t, err.Error, errInvalidSteamID.Error())
		})
		t.Run("tooManyRequested", func(t *testing.T) {
			t.Parallel()

			var (
				sids = generateIDs(maxResults + 10)
				err  apiErr
				path = fmt.Sprintf("/summary?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &err))
			require.Equal(t, err.Error, errTooMany.Error())
		})
	}
}

func apiTestSummary(router *http.ServeMux) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			var (
				sids      = steamid.Collection{testIDb4nny, testIDCamper}
				summaries []steamweb.PlayerSummary
				path      = fmt.Sprintf("/summary?steamids=%s", SteamIDStringList(sids))
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &summaries))
			require.Equal(t, len(sids), len(summaries))
		})
	}
}

func apiTestGetFriends(router *http.ServeMux) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			sids := steamid.Collection{testIDb4nny, testIDCamper}

			request := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/friends?steamids=%s", SteamIDStringList(sids)), nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			data, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Errorf("expected error to be nil got %v", err)
			}

			var friends friendMap

			require.NoError(t, json.Unmarshal(data, &friends))
			require.True(t, len(friends[sids[0].String()]) > 0)
			require.True(t, len(friends[sids[1].String()]) == 0)
		})
	}
}

func apiTestGetProfile(router *http.ServeMux) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		var (
			sids     = steamid.Collection{testIDb4nny, testIDCamper}
			path     = fmt.Sprintf("/profile?steamids=%s", SteamIDStringList(sids))
			profiles []domain.Profile
			validIDs steamid.Collection
		)

		require.NoError(t, testReq(t, router, http.MethodGet, path, &profiles))

		for _, sid := range sids {
			for _, profile := range profiles {
				if profile.Summary.SteamID == sid {
					require.Equal(t, steamweb.EconBanNone, profile.BanState.EconomyBan)
					require.NotEqual(t, "", profile.Summary.PersonaName)
					require.Equal(t, sid, profile.Summary.SteamID)
					require.Equal(t, sid, profile.BanState.SteamID)

					validIDs = append(validIDs, profile.Summary.SteamID)
				}
			}
		}

		require.EqualValues(t, sids, validIDs)
	}
}

func createTestSourcebansRecord(t *testing.T, database *pgStore, sid64 steamid.SteamID) domain.SbBanRecord {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	curTime := time.Now()

	player := newPlayerRecord(sid64)
	if errPlayer := database.playerGetOrCreate(ctx, sid64, &player); errPlayer != nil {
		t.Error(errPlayer)
	}

	site := newSourcebansSite(domain.Site(fmt.Sprintf("Test %s", curTime)))
	if errSave := database.sourcebansSiteSave(ctx, &site); errSave != nil {
		t.Error(errSave)
	}

	record := newSourcebansRecord(site, sid64, "Name Goes Here", "Smelly",
		curTime.AddDate(-1, 0, 0), time.Hour*24, false)
	record.CreatedOn = curTime

	if errSave := database.sourcebansBanRecordSave(ctx, &record); errSave != nil {
		t.Error(errSave)
	}

	return record
}

func apiTestGetSourcebans(router *http.ServeMux, database *pgStore) func(t *testing.T) {
	return func(t *testing.T) {
		recordA := createTestSourcebansRecord(t, database, testIDb4nny)
		recordB := createTestSourcebansRecord(t, database, testIDCamper)

		t.Run("single", func(t *testing.T) {
			t.Parallel()

			var (
				banRecords []domain.SbBanRecord
				path       = fmt.Sprintf("/sourcebans/%d", recordA.SteamID.Int64())
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &banRecords))
			require.Equal(t, recordA.BanID, banRecords[0].BanID)
			require.Equal(t, recordA.SiteName, banRecords[0].SiteName)
			require.Equal(t, recordA.SiteID, banRecords[0].SiteID)
			require.Equal(t, recordA.PersonaName, banRecords[0].PersonaName)
			require.Equal(t, recordA.SteamID, banRecords[0].SteamID)
			require.Equal(t, recordA.Reason, banRecords[0].Reason)
			require.Equal(t, recordA.Duration, banRecords[0].Duration)
			require.Equal(t, recordA.Permanent, banRecords[0].Permanent)
		})
		t.Run("multi", func(t *testing.T) {
			t.Parallel()

			var (
				banRecords BanRecordMap
				path       = fmt.Sprintf("/sourcebans?steamids=%s",
					SteamIDStringList(steamid.Collection{recordA.SteamID, recordB.SteamID}))
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &banRecords))
			require.Equal(t, 2, len(banRecords))
		})
		t.Run("emptyResults", func(t *testing.T) {
			t.Parallel()

			var (
				steamID    = steamid.RandSID64()
				banRecords []domain.SbBanRecord
				path       = fmt.Sprintf("/sourcebans/%d", steamID.Int64())
			)

			require.NoError(t, testReq(t, router, http.MethodGet, path, &banRecords))
			require.Equal(t, []domain.SbBanRecord{}, banRecords)
		})
	}
}
