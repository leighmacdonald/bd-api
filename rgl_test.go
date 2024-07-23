package main

import (
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestRGLPlayer(t *testing.T) {
	body, errRead := os.Open("testdata/rgl_player.html")
	require.NoError(t, errRead)
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	require.NoError(t, err)

	match, errDetails := parseMatchFromDoc(doc.Find("html"))
	require.NoError(t, errDetails)

	require.Equal(t, "Qixalite Booking: RED vs BLU", match.Title)
	require.Equal(t, "koth_cascade_rc2", match.Map)
	require.Equal(t, "16m56s", match.Duration.String())
	require.Equal(t, "2022-02-05 06:39:42 +0000 UTC", match.CreatedOn.String())
	require.Equal(t, 0, match.ScoreBLU)
	require.Equal(t, 3, match.ScoreRED)
	require.Len(t, match.Players, 19)
	require.EqualValues(t, domain.LogsTFPlayer{
		LogID:   3124689,
		SteamID: steamid.New(76561198164892406),
		Team:    domain.BLU,
		Name:    "var",
		Classes: []domain.LogsTFPlayerClass{
			{
				LogID:   3124689,
				SteamID: steamid.New(76561198164892406),
				Class:   domain.Scout,
				Played:  domain.JSONDuration{Duration: 1016000000000},
				Kills:   12,
				Assists: 10,
				Deaths:  14,
				Damage:  5078,
				Weapons: nil,
			},
		},
		Kills:        12,
		Assists:      10,
		Deaths:       14,
		Damage:       5078,
		DPM:          299,
		KAD:          1.6,
		KD:           0.9,
		DamageTaken:  5399,
		DTM:          318,
		HealthPacks:  16,
		Backstabs:    0,
		Headshots:    0,
		Airshots:     0,
		Caps:         3,
		HealingTaken: 0,
	}, match.Players[0])
	require.Len(t, match.Medics, 2)
	require.EqualValues(t, domain.LogsTFMedic{
		LogID:            3124689,
		SteamID:          steamid.New(76561198113244106),
		Healing:          17368,
		HealingPerMin:    1025,
		ChargesKritz:     0,
		ChargesQuickfix:  0,
		ChargesMedigun:   4,
		ChargesVacc:      0,
		Drops:            2,
		AvgTimeBuild:     domain.JSONDuration{Duration: 42000000000},
		AvgTimeUse:       domain.JSONDuration{Duration: 28000000000},
		NearFullDeath:    1,
		AvgUberLen:       domain.JSONDuration{Duration: 7000000000},
		DeathAfterCharge: 0,
		MajorAdvLost:     1,
		BiggestAdvLost:   domain.JSONDuration{Duration: 39000000000},
	}, match.Medics[0])

	require.Len(t, match.Rounds, 3)
	require.EqualValues(t, domain.LogsTFRound{
		LogID:     3124689,
		Round:     3,
		Length:    domain.JSONDuration{Duration: 325000000000},
		ScoreBLU:  0,
		ScoreRED:  3,
		KillsBLU:  34,
		KillsRED:  46,
		UbersBLU:  1,
		UbersRED:  1,
		DamageBLU: 15258,
		DamageRED: 12677,
		MidFight:  domain.RED,
	}, match.Rounds[2])
}

func TestLogsTFDetailsOld(t *testing.T) {
	body, errRead := os.Open("testdata/logstf_details_old.html")
	require.NoError(t, errRead)
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	require.NoError(t, err)

	match, errDetails := parseMatchFromDoc(doc.Find("html"))
	require.NoError(t, errDetails)

	require.Equal(t, 100, match.LogID)
	require.Equal(t, "Log 100", match.Title)
	require.Equal(t, "", match.Map)
	require.Equal(t, "39m3s", match.Duration.String())
	require.Equal(t, "2012-11-26 07:39:59 +0000 UTC", match.CreatedOn.String())
	require.Equal(t, 4, match.ScoreBLU)
	require.Equal(t, 2, match.ScoreRED)

	require.Nil(t, match.Rounds)
	require.Len(t, match.Players, 19)
	require.Len(t, match.Medics, 2)
	require.EqualValues(t, domain.LogsTFPlayer{
		LogID:   100,
		SteamID: steamid.New(76561198006069420),
		Team:    domain.BLU,
		Name:    "paradox",
		Classes: []domain.LogsTFPlayerClass{
			{
				LogID:   100,
				SteamID: steamid.New(76561198006069420),
				Class:   domain.Sniper,
				Played:  domain.JSONDuration{Duration: 2343000000000},
				Kills:   57,
				Assists: 21,
				Deaths:  30,
				Damage:  18981,
				Weapons: nil,
			},
		},
		Kills:        57,
		Assists:      21,
		Deaths:       30,
		Damage:       18981,
		DPM:          486,
		KAD:          2.6,
		KD:           1.9,
		DamageTaken:  0,
		DTM:          0,
		HealthPacks:  15,
		Backstabs:    41,
		Headshots:    5,
		Airshots:     0,
		Caps:         0,
		HealingTaken: 0,
	}, match.Players[0])
}
