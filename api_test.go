package main

import (
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if key, found := os.LookupEnv("BDAPI_STEAM_API_KEY"); found && key != "" {
		if errKey := steamweb.SetKey(key); errKey != nil {
			os.Exit(2)
		}
	}
	os.Exit(m.Run())
}

func TestGetBans(t *testing.T) {
	router := createRouter(testStore)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bans?steam_id=%d", testIDb4nny), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var bs []steamweb.PlayerBanState
	require.NoError(t, json.Unmarshal(body, &bs))
	sid := testIDb4nny
	require.Equal(t, sid.String(), bs[0].SteamID)
}

func TestGetSummary(t *testing.T) {
	router := createRouter(testStore)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/summary?steam_id=%d", testIDb4nny), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	data, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var bs []steamweb.PlayerSummary
	require.NoError(t, json.Unmarshal(data, &bs))
	sid := testIDb4nny
	require.Equal(t, sid.String(), bs[0].SteamID)
}

func TestGetProfile(t *testing.T) {
	router := createRouter(testStore)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/profile?steam_id=%d", testIDb4nny), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	data, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var profile Profile
	require.NoError(t, json.Unmarshal(data, &profile))
	sid := testIDb4nny
	require.Equal(t, "none", profile.BanState.EconomyBan)
	require.Equal(t, sid.String(), profile.Summary.SteamID)
	//require.True(t, len(profile.Friends) > 0)
}
