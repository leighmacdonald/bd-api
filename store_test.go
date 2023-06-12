package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"testing"
	"time"
)

var testStore *pgStore

func TestStore(t *testing.T) {
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
	//host, _ := cont.Host(context.Background())
	port, _ := cont.MappedPort(context.Background(), "5432")
	config := appConfig{
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
	var s sbSite
	require.Error(t, testStore.sbSiteGet(context.Background(), 99999, &s))
	s2 := newSBSite("test-site")
	require.NoError(t, testStore.sbSiteSave(context.Background(), &s2))
	var s3 sbSite
	require.NoError(t, testStore.sbSiteGet(context.Background(), s2.SiteID, &s3))
	require.Equal(t, s2.Name, s3.Name)
	require.Equal(t, s2.UpdatedOn.Second(), s3.UpdatedOn.Second())

	pr := newPlayerRecord(testIDCamper)
	pr.PersonaName = "blah"
	pr.Vanity = "poop3r"
	require.NoError(t, testStore.playerRecordSave(context.Background(), &pr))

	t0 := time.Now().AddDate(-1, 0, 0)
	t1 := t0.AddDate(0, 1, 0)
	recA := s3.newRecord(testIDCamper, "test", t0, t1.Sub(t0), false)
	require.NoError(t, testStore.sbBanSave(context.Background(), &recA))

	require.NoError(t, testStore.sbSiteDelete(context.Background(), s3.SiteID))
	require.Error(t, testStore.sbSiteGet(context.Background(), s3.SiteID, &s))
}

func testPlayerRecord(t *testing.T) {
	pr := newPlayerRecord(76561197961279983)
	pr.PersonaName = "blah"
	pr.Vanity = "123"
	require.NoError(t, testStore.playerRecordSave(context.Background(), &pr))
	names, errNames := testStore.playerGetNames(context.Background(), pr.SteamID)
	require.NoError(t, errNames)
	nameOk := false
	for _, name := range names {
		if name.PersonaName == pr.PersonaName {
			nameOk = true
			break
		}
	}
	require.True(t, nameOk, "Name not found")

	vNameOk := false
	vNames, errVNames := testStore.playerGetVanityNames(context.Background(), pr.SteamID)
	require.NoError(t, errVNames)
	for _, name := range vNames {
		if name.Vanity == pr.Vanity {
			vNameOk = true
			break
		}
	}
	require.True(t, vNameOk, "Vanity not found")

	avatarOk := false
	avatars, errAvatars := testStore.playerGetAvatars(context.Background(), pr.SteamID)
	require.NoError(t, errAvatars)
	for _, name := range avatars {
		if name.AvatarHash == pr.AvatarHash {
			avatarOk = true
			break
		}
	}
	require.True(t, avatarOk, "Avatar not found")
}
