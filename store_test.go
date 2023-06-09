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
)

var testStore *pgStore

func TestMain(m *testing.M) {
	testCtx := context.Background()
	username, password, dbName := "bdapi-test", "bdapi-test", "bdapi-test"
	container, errContainer := postgres.RunContainer(
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
	//host, _ := container.Host(context.Background())
	port, _ := container.MappedPort(context.Background(), "5432")
	config := appConfig{
		DSN: fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName),
	}
	defer func() {
		if errTerm := container.Terminate(testCtx); errTerm != nil {
			logger.Error("Failed to terminate test container")
		}
	}()
	newTestStore, errStore := newStore(testCtx, config.DSN)
	if errStore != nil {
		logger.Fatal("Failed to setup test db", zap.Error(errStore))
	}
	testStore = newTestStore
	m.Run()
}

func TestSBSite(t *testing.T) {
	var s sbSite
	require.Error(t, testStore.sbSiteGet(context.Background(), 99999, &s))
	s2 := newSBSite("test-site")
	require.NoError(t, testStore.sbSiteSave(context.Background(), &s2))
	var s3 sbSite
	require.NoError(t, testStore.sbSiteGet(context.Background(), s2.SiteID, &s3))
	require.Equal(t, s2.Name, s3.Name)
	require.Equal(t, s2.UpdatedOn.Second(), s3.UpdatedOn.Second())
}

func TestPlayerRecord(t *testing.T) {
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
