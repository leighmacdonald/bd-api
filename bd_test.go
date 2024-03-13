package main

import (
	"github.com/leighmacdonald/steamid/v4/steamid"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBDUpdates(t *testing.T) {
	list1 := BDList{
		BDListID:    1,
		BDListName:  "list1",
		URL:         "",
		Game:        "tf2",
		TrustWeight: 1,
		Deleted:     false,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	entries1 := []BDListEntry{
		{
			BDListEntryID: 1,
			BDListID:      1,
			SteamID:       steamid.New("76561197960265729"),
			Attribute:     "cheater",
			LastSeen:      time.Now(),
			LastName:      "player1",
			Deleted:       false,
			CreatedOn:     time.Now(),
			UpdatedOn:     time.Now(),
		},
		{
			BDListEntryID: 2,
			BDListID:      1,
			SteamID:       steamid.New("76561197960265730"),
			Attribute:     "cheater",
			LastSeen:      time.Now(),
			LastName:      "player2",
			Deleted:       false,
			CreatedOn:     time.Now(),
			UpdatedOn:     time.Now(),
		},
	}

	schema1 := listMapping{
		list: list1,
		result: &TF2BDSchema{
			Schema:   "",
			FileInfo: FileInfo{},
			Players: []Players{
				{
					Attributes: []string{"cheater"},
					LastSeen: LastSeen{
						PlayerName: "player1",
						Time:       int(entries1[0].LastSeen.Unix()),
					},
					Steamid: 76561197960265729, Proof: []string{},
				},
				{
					Attributes: []string{"cheater"},
					LastSeen: LastSeen{
						PlayerName: "player2-edit",
						Time:       int(entries1[1].LastSeen.Unix()),
					},
					Steamid: 76561197960265730, Proof: []string{},
				},
				{
					Attributes: []string{"cheater"},
					LastSeen: LastSeen{
						PlayerName: "player3",
						Time:       1709935391,
					},
					Steamid: 76561197960265731, Proof: []string{},
				},
			},
		},
	}

	n, u := findNewAndUpdated(entries1, schema1)
	require.Equal(t, 1, len(n))
	require.Equal(t, 1, len(u))

	schema1.result.Players = slices.Delete(schema1.result.Players, 0, 1)

	deleted := findDeleted(entries1, schema1)
	require.Equal(t, 1, len(deleted))
}
