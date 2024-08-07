package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
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

		var site domain.SbSite
		require.Error(t, database.sourcebansSiteGet(context.Background(), 99999, &site))

		site2 := newSourcebansSite("test-site")
		require.NoError(t, database.sourcebansSiteSave(context.Background(), &site2))

		var site3 domain.SbSite
		require.NoError(t, database.sourcebansSiteGet(context.Background(), site2.SiteID, &site3))
		require.Equal(t, site2.Name, site3.Name)
		require.Equal(t, site2.UpdatedOn.Second(), site3.UpdatedOn.Second())

		pRecord := newPlayerRecord(testIDCamper)
		pRecord.PersonaName = "blah"
		pRecord.Vanity = "poop3r"
		require.NoError(t, database.playerRecordSave(context.Background(), &pRecord))

		t0 := time.Now().AddDate(-1, 0, 0)
		t1 := t0.AddDate(0, 1, 0)
		recA := newSourcebansRecord(site3, testIDCamper, "blah", "test", t0, t1.Sub(t0), false)
		require.NoError(t, database.sourcebansBanRecordSave(context.Background(), &recA))
		require.NoError(t, database.sourcebansSiteDelete(context.Background(), site3.SiteID))
		require.Error(t, database.sourcebansSiteGet(context.Background(), site3.SiteID, &site))
	}
}

func sourceBansPlayerRecordTest(database *pgStore) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		pRecord := newPlayerRecord(steamid.New(76561197961279983))
		pRecord.PersonaName = "blah"
		pRecord.Vanity = "123"
		require.NoError(t, database.playerRecordSave(context.Background(), &pRecord))

		names, errNames := database.playerNames(context.Background(), pRecord.SteamID)
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
		vNames, errVNames := database.playerVanityNames(context.Background(), pRecord.SteamID)
		require.NoError(t, errVNames)

		for _, name := range vNames {
			if name.Vanity == pRecord.Vanity {
				vNameOk = true

				break
			}
		}
		require.True(t, vNameOk, "Vanity not found")

		avatarOk := false
		avatars, errAvatars := database.playerAvatars(context.Background(), pRecord.SteamID)
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
		aList = domain.BDList{
			BDListName:  "list a",
			URL:         "http://localhost/a",
			Game:        "tf2",
			TrustWeight: 5,
			Deleted:     false,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
		bList = domain.BDList{
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
		t.Parallel()

		ctx := context.Background()
		listA, errA := database.botDetectorListCreate(ctx, aList)
		require.NoError(t, errA)
		require.True(t, listA.BDListID > 0)
		listB, errB := database.botDetectorListCreate(ctx, bList)
		require.NoError(t, errB)
		require.True(t, listB.BDListID > 0)
		require.NotEqual(t, listA.BDListID, listB.BDListID)

		var entriesA []domain.BDListEntry
		for idx := 0; idx < 5; idx++ {
			record := newPlayerRecord(steamid.RandSID64())
			require.NoError(t, database.playerRecordSave(ctx, &record))
			entry, errEntry := database.botDetectorListEntryCreate(ctx, domain.BDListEntry{
				BDListID:   listA.BDListID,
				SteamID:    record.SteamID,
				Attributes: []string{"cheater"},
				Proof:      []string{"proof", "prooof"},
				LastSeen:   time.Now(),
				LastName:   fmt.Sprintf("name_%d", idx),
				Deleted:    false,
				CreatedOn:  time.Now(),
				UpdatedOn:  time.Now(),
			})
			require.Nil(t, errEntry)
			require.True(t, entry.BDListEntryID > 0)
			entriesA = append(entriesA, entry)
		}

		require.Equal(t, 5, len(entriesA))

		newName := listA.BDListName + listA.BDListName
		listA.BDListName = newName
		require.NoError(t, database.botDetectorListSave(ctx, listA))

		listAEdited, errEdited := database.botDetectorListByName(ctx, listA.BDListName)
		require.NoError(t, errEdited)
		require.Equal(t, newName, listAEdited.BDListName)
	}
}
