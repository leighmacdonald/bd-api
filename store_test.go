package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

var testStore *pgStore

func TestStore(t *testing.T) {
	t.Parallel()

	testCtx := context.Background()

	username, password, dbName := "bdapi-test", "bdapi-test", "bdapi-test"
	cont, errContainer := postgres.RunContainer(
		testCtx,
		testcontainers.WithImage("docker.io/postgres:15-bullseye"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(wait.
			ForLog("database system is ready to accept connections").
			WithOccurrence(2)),
	)

	if errContainer != nil {
		logger.Fatal("Failed to setup test db", zap.Error(errContainer))
	}

	port, _ := cont.MappedPort(context.Background(), "5432")
	config := appConfig{ //nolint:exhaustruct
		DSN: fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName),
	}

	defer func() {
		if errTerm := cont.Terminate(testCtx); errTerm != nil {
			logger.Error("Failed to terminate test container")
		}
	}()

	newTestStore, errStore := newStore(testCtx, config.DSN)
	if errStore != nil {
		logger.Fatal("Failed to setup test db", zap.Error(errStore))
	}

	testStore = newTestStore

	testSourceBans(t)
	testPlayerRecord(t)
}

func testSourceBans(t *testing.T) {
	t.Helper()

	var site sbSite

	require.Error(t, testStore.sbSiteGet(context.Background(), 99999, &site))

	site2 := newSBSite("test-site")

	require.NoError(t, testStore.sbSiteSave(context.Background(), &site2))

	var site3 sbSite

	require.NoError(t, testStore.sbSiteGet(context.Background(), site2.SiteID, &site3))
	require.Equal(t, site2.Name, site3.Name)
	require.Equal(t, site2.UpdatedOn.Second(), site3.UpdatedOn.Second())

	pRecord := newPlayerRecord(testIDCamper)
	pRecord.PersonaName = "blah"
	pRecord.Vanity = "poop3r"

	require.NoError(t, testStore.playerRecordSave(context.Background(), &pRecord))

	t0 := time.Now().AddDate(-1, 0, 0)
	t1 := t0.AddDate(0, 1, 0)
	recA := site3.newRecord(testIDCamper, "blah", "test", t0, t1.Sub(t0), false)

	require.NoError(t, testStore.sbBanSave(context.Background(), &recA))
	require.NoError(t, testStore.sbSiteDelete(context.Background(), site3.SiteID))
	require.Error(t, testStore.sbSiteGet(context.Background(), site3.SiteID, &site))
}

func testPlayerRecord(t *testing.T) {
	t.Helper()

	pRecord := newPlayerRecord(76561197961279983)
	pRecord.PersonaName = "blah"
	pRecord.Vanity = "123"

	require.NoError(t, testStore.playerRecordSave(context.Background(), &pRecord))

	names, errNames := testStore.playerGetNames(context.Background(), pRecord.SteamID)

	require.NoError(t, errNames)

	nameOk := false

	for _, name := range names {
		if name.PersonaName == pRecord.PersonaName {
			nameOk = true

			break
		}
	}

	require.True(t, nameOk, "Name not found")

	vNameOk := false
	vNames, errVNames := testStore.playerGetVanityNames(context.Background(), pRecord.SteamID)

	require.NoError(t, errVNames)

	for _, name := range vNames {
		if name.Vanity == pRecord.Vanity {
			vNameOk = true

			break
		}
	}

	require.True(t, vNameOk, "Vanity not found")

	avatarOk := false
	avatars, errAvatars := testStore.playerGetAvatars(context.Background(), pRecord.SteamID)
	require.NoError(t, errAvatars)

	for _, name := range avatars {
		if name.AvatarHash == pRecord.AvatarHash {
			avatarOk = true

			break
		}
	}

	require.True(t, avatarOk, "Avatar not found")
}
