package main

import (
	"context"
	"github.com/leighmacdonald/steamweb"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientProfile(t *testing.T) {
	cache = newCaches(context.Background(), steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
	testGetProfileServer := httptest.NewServer(http.HandlerFunc(handleGetProfile()))
	defer testGetProfileServer.Close()
	client := NewClient(testGetProfileServer.URL)
	var profile Profile
	require.NoError(t, client.GetProfile(context.Background(), testIDb4nny, &profile))
	sid := testIDb4nny
	require.Equal(t, sid.String(), profile.Summary.Steamid)
	testGetSummaryServer := httptest.NewServer(http.HandlerFunc(handleGetSummary()))
	defer testGetSummaryServer.Close()
	clientSummary := NewClient(testGetSummaryServer.URL)
	var summary steamweb.PlayerSummary
	errSummaries := clientSummary.PlayerSummary(context.Background(), sid, &summary)
	require.NoError(t, errSummaries)
	require.EqualValues(t, profile.Summary, summary)
	hasUGC, hasETF2L, hasRGL := false, false, false
	for _, season := range profile.Seasons {
		switch season.League {
		case leagueUGC:
			hasUGC = true
		case leagueETF2L:
			hasETF2L = true
		case leagueRGL:
			hasRGL = true
		}
	}
	require.True(t, hasETF2L)
	require.True(t, hasUGC)
	require.False(t, hasRGL)
	require.True(t, profile.LogsCount > 5000)
}

func TestClientBans(t *testing.T) {
	cache = newCaches(context.Background(), steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
	testGetProfileServer := httptest.NewServer(http.HandlerFunc(handleGetBans()))
	defer testGetProfileServer.Close()
	client := NewClient(testGetProfileServer.URL)
	var bans steamweb.PlayerBanState
	require.NoError(t, client.GetPlayerBan(context.Background(), testIDSquirrel, &bans))
	require.True(t, bans.VACBanned)
	require.True(t, bans.DaysSinceLastBan > 0)
}
