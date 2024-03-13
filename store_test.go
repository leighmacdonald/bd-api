package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/stretchr/testify/require"
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

	database, errStore := newStore(ctx, dsn)
	if errStore != nil {
		panic(errStore)
	}

	t.Run("sourceBansStoreTest", sourceBansStoreTest(database))               //nolint:paralleltest
	t.Run("sourceBansPlayerRecordTest", sourceBansPlayerRecordTest(database)) //nolint:paralleltest
	t.Run("bot_detector", bdTest(database))
}

func sourceBansStoreTest(database *pgStore) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		var site SbSite
		require.Error(t, database.sbSiteGet(context.Background(), 99999, &site))

		site2 := NewSBSite("test-site")
		require.NoError(t, database.sbSiteSave(context.Background(), &site2))

		var site3 SbSite
		require.NoError(t, database.sbSiteGet(context.Background(), site2.SiteID, &site3))
		require.Equal(t, site2.Name, site3.Name)
		require.Equal(t, site2.UpdatedOn.Second(), site3.UpdatedOn.Second())

		pRecord := newPlayerRecord(testIDCamper)
		pRecord.PersonaName = "blah"
		pRecord.Vanity = "poop3r"
		require.NoError(t, database.playerRecordSave(context.Background(), &pRecord))

		t0 := time.Now().AddDate(-1, 0, 0)
		t1 := t0.AddDate(0, 1, 0)
		recA := newRecord(site3, testIDCamper, "blah", "test", t0, t1.Sub(t0), false)
		require.NoError(t, database.sbBanSave(context.Background(), &recA))
		require.NoError(t, database.sbSiteDelete(context.Background(), site3.SiteID))
		require.Error(t, database.sbSiteGet(context.Background(), site3.SiteID, &site))
	}
}

func sourceBansPlayerRecordTest(database *pgStore) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		pRecord := newPlayerRecord(steamid.New(76561197961279983))
		pRecord.PersonaName = "blah"
		pRecord.Vanity = "123"
		require.NoError(t, database.playerRecordSave(context.Background(), &pRecord))

		names, errNames := database.playerGetNames(context.Background(), pRecord.SteamID)
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
		vNames, errVNames := database.playerGetVanityNames(context.Background(), pRecord.SteamID)
		require.NoError(t, errVNames)

		for _, name := range vNames {
			if name.Vanity == pRecord.Vanity {
				vNameOk = true

				break
			}
		}
		require.True(t, vNameOk, "Vanity not found")

		avatarOk := false
		avatars, errAvatars := database.playerGetAvatars(context.Background(), pRecord.SteamID)
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

func bdTest(database *pgStore) func(t *testing.T) {
	var (
		a = BDList{
			BDListName:  "list a",
			URL:         "http://localhost/a",
			Game:        "tf2",
			TrustWeight: 5,
			Deleted:     false,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
		b = BDList{
			BDListName:  "list b",
			URL:         "http://localhost/b",
			Game:        "tf2",
			TrustWeight: 9,
			Deleted:     false,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
	)

	return func(t *testing.T) {
		ctx := context.Background()
		listA, errA := database.bdListCreate(ctx, a)
		require.NoError(t, errA)
		require.True(t, listA.BDListID > 0)
		listB, errB := database.bdListCreate(ctx, b)
		require.NoError(t, errB)
		require.True(t, listB.BDListID > 0)
		require.NotEqual(t, listA.BDListID, listB.BDListID)

		var entriesA []BDListEntry
		for i := 0; i < 5; i++ {
			entry, errEntry := database.bdListEntryCreate(ctx, BDListEntry{
				BDListID:   listA.BDListID,
				SteamID:    steamid.RandSID64(),
				Attributes: []string{"cheater"},
				LastSeen:   time.Now(),
				LastName:   fmt.Sprintf("name_%d", i),
				Deleted:    false,
				CreatedOn:  time.Now(),
				UpdatedOn:  time.Now(),
			})
			require.NotNil(t, errEntry)
			require.True(t, entry.BDListEntryID > 0)
			entriesA = append(entriesA, entry)
		}
	}
}
