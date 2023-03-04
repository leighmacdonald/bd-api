package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamweb"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBans(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bans?steam_id=%d", testIdb4nny), nil)
	w := httptest.NewRecorder()
	getHandler(handleGetBans(newCaches(context.Background(), steamCacheTimeout, compCacheTimeout, steamCacheTimeout)))(w, req)
	res := w.Result()
	defer func() {
		require.NoError(t, res.Body.Close())
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var bs steamweb.PlayerBanState
	require.NoError(t, json.Unmarshal(data, &bs))
	sid := testIdb4nny
	require.Equal(t, sid.String(), bs.SteamID)
}

func TestGetSummary(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/summary?steam_id=%d", testIdb4nny), nil)
	w := httptest.NewRecorder()
	getHandler(handleGetBans(newCaches(context.Background(), steamCacheTimeout, compCacheTimeout, steamCacheTimeout)))(w, req)
	res := w.Result()
	defer func() {
		require.NoError(t, res.Body.Close())
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var bs steamweb.PlayerSummary
	require.NoError(t, json.Unmarshal(data, &bs))
	sid := testIdb4nny
	require.Equal(t, sid.String(), bs.Steamid)
}

func TestGetProfile(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/profile?steam_id=%d", testIdb4nny), nil)
	w := httptest.NewRecorder()
	getHandler(handleGetProfile(newCaches(context.Background(), steamCacheTimeout, compCacheTimeout, steamCacheTimeout)))(w, req)
	res := w.Result()
	defer func() {
		require.NoError(t, res.Body.Close())
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var profile Profile
	require.NoError(t, json.Unmarshal(data, &profile))
	sid := testIdb4nny
	require.Equal(t, "none", profile.BanState.EconomyBan)
	require.Equal(t, sid.String(), profile.Summary.Steamid)
}
