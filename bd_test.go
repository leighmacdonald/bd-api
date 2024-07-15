package main

import (
	"slices"
	"testing"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestBDUpdates(t *testing.T) {
	list1 := domain.BDList{
		BDListID:    1,
		BDListName:  "list1",
		URL:         "",
		Game:        "tf2",
		TrustWeight: 1,
		Deleted:     false,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	entries1 := []domain.BDListEntry{
		{
			BDListEntryID: 1,
			BDListID:      1,
			SteamID:       steamid.New("76561197960265729"),
			Attributes:    []string{"cheater"},
			Proof:         []string{"lol"},
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
			Attributes:    []string{"cheater"},
			Proof:         []string{"lol"},
			LastSeen:      time.Now(),
			LastName:      "player2",
			Deleted:       false,
			CreatedOn:     time.Now(),
			UpdatedOn:     time.Now(),
		},
	}

	schema1 := listMapping{
		list: list1,
		result: &domain.TF2BDSchema{
			Schema:   "",
			FileInfo: domain.FileInfo{},
			Players: []domain.TF2BDPlayer{
				{
					Attributes: []string{"cheater"},
					LastSeen: domain.LastSeen{
						PlayerName: "player1",
						Time:       int(entries1[0].LastSeen.Unix()),
					},
					Steamid: 76561197960265729, Proof: []string{"lol"},
				},
				{
					Attributes: []string{"cheater"},
					LastSeen: domain.LastSeen{
						PlayerName: "player2-edit",
						Time:       int(entries1[1].LastSeen.Unix()),
					},
					Steamid: 76561197960265730, Proof: []string{},
				},
				{
					Attributes: []string{"cheater"},
					LastSeen: domain.LastSeen{
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
