package main

import (
	"context"
	"github.com/leighmacdonald/etf2l"
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStore(t *testing.T) {
	t.Parallel()

	if key, found := os.LookupEnv("BDAPI_STEAM_API_KEY"); found && key != "" {
		if errKey := steamweb.SetKey(key); errKey != nil {
			panic(errKey)
		}
	}

	ctx := context.Background()
	dsn, databaseContainer, errDB := newTestDB(ctx)

	if errDB != nil {
		t.Skipf("Failed to bring up testcontainer db: %v", errDB)
	}

	t.Cleanup(func() {
		if errTerm := databaseContainer.Terminate(ctx); errTerm != nil {
			t.Error("Failed to terminate test container")
		}
	})

	database, errStore := newStore(ctx, zap.NewNop(), dsn)
	if errStore != nil {
		panic(errStore)
	}

	t.Run("sourceBansStoreTest", sourceBansStoreTest(database))
	t.Run("sourceBansPlayerRecordTest", sourceBansPlayerRecordTest(database))
	t.Run("etf2l", testETF2L(database))
}

func testETF2L(database *pgStore) func(t *testing.T) {
	client := etf2l.New()

	return func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		prA := newPlayerRecord(testIDb4nny)
		prB := newPlayerRecord(testIDBanned)

		require.NoError(t, playerGetOrCreate(ctx, database, testIDb4nny, &prA))
		require.NoError(t, playerGetOrCreate(ctx, database, testIDBanned, &prB))
		require.NoError(t, updateETF2LPlayer(ctx, client, database, prA.SteamID))
		require.NoError(t, updateETF2LPlayer(ctx, client, database, prB.SteamID))

		var p1 ETF2LPlayer
		require.NoError(t, etf2lPlayerBySteamID(ctx, database, prB.SteamID, &p1))
		require.Equal(t, prB.SteamID, p1.SteamID)

		bans, errBans := etf2lBans(ctx, database, prB.SteamID)
		require.NoError(t, errBans)
		require.True(t, len(bans) >= 3)

		teams, errTeams := etf2lPlayerTeams(ctx, database, prB.SteamID)
		require.NoError(t, errTeams)
		require.True(t, len(teams) >= 2)
	}
}

func sourceBansStoreTest(database *pgStore) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		var site models.SbSite

		require.Error(t, sbSiteGet(context.Background(), database, 99999, &site))

		site2 := NewSBSite("test-site")

		require.NoError(t, sbSiteSave(context.Background(), database, &site2))

		var site3 models.SbSite

		require.NoError(t, sbSiteGet(context.Background(), database, site2.SiteID, &site3))
		require.Equal(t, site2.Name, site3.Name)
		require.Equal(t, site2.UpdatedOn.Second(), site3.UpdatedOn.Second())

		pRecord := newPlayerRecord(testIDCamper)
		pRecord.PersonaName = "blah"
		pRecord.Vanity = "poop3r"

		require.NoError(t, playerRecordSave(context.Background(), database, &pRecord))

		t0 := time.Now().AddDate(-1, 0, 0)
		t1 := t0.AddDate(0, 1, 0)
		recA := newRecord(site3, testIDCamper, "blah", "test", t0, t1.Sub(t0), false)

		require.NoError(t, sbBanSave(context.Background(), database, &recA))
		require.NoError(t, sbSiteDelete(context.Background(), database, site3.SiteID))
		require.Error(t, sbSiteGet(context.Background(), database, site3.SiteID, &site))
	}
}

func sourceBansPlayerRecordTest(database *pgStore) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		pRecord := newPlayerRecord(steamid.New(76561197961279983))
		pRecord.PersonaName = "blah"
		pRecord.Vanity = "123"

		require.NoError(t, playerRecordSave(context.Background(), database, &pRecord))

		names, errNames := playerGetNames(context.Background(), database, pRecord.SteamID)

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
		vNames, errVNames := playerGetVanityNames(context.Background(), database, pRecord.SteamID)

		require.NoError(t, errVNames)

		for _, name := range vNames {
			if name.Vanity == pRecord.Vanity {
				vNameOk = true

				break
			}
		}

		require.True(t, vNameOk, "Vanity not found")

		avatarOk := false
		avatars, errAvatars := playerGetAvatars(context.Background(), database, pRecord.SteamID)
		require.NoError(t, errAvatars)

		for _, name := range avatars {
			if name.AvatarHash == pRecord.AvatarHash {
				avatarOk = true

				break
			}
		}

		require.True(t, avatarOk, "Avatar not found")
	}
}
