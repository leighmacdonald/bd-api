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
	ctx := context.Background()
	cache := newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
	testGetProfileServer := httptest.NewServer(http.HandlerFunc(handleGetProfile(cache)))
	defer testGetProfileServer.Close()
	client := NewClient(testGetProfileServer.URL)
	var profile Profile
	require.NoError(t, client.GetProfile(ctx, testIDb4nny, &profile))
	sid := testIDb4nny
	require.Equal(t, sid.String(), profile.Summary.Steamid)
	testGetSummaryServer := httptest.NewServer(http.HandlerFunc(handleGetSummary(cache)))
	defer testGetSummaryServer.Close()
	clientSummary := NewClient(testGetSummaryServer.URL)
	var summary steamweb.PlayerSummary
	errSummaries := clientSummary.PlayerSummary(ctx, sid, &summary)
	require.NoError(t, errSummaries)
	require.EqualValues(t, profile.Summary, summary)
}

func TestClientBans(t *testing.T) {
	ctx := context.Background()
	cache := newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
	testGetProfileServer := httptest.NewServer(http.HandlerFunc(handleGetBans(cache)))
	defer testGetProfileServer.Close()
	client := NewClient(testGetProfileServer.URL)
	var bans steamweb.PlayerBanState
	require.NoError(t, client.GetPlayerBan(ctx, testIDSquirrel, &bans))
	require.True(t, bans.VACBanned)
	require.True(t, bans.DaysSinceLastBan > 0)
}
