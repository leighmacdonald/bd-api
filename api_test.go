package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/leighmacdonald/steamweb/v2"
	"github.com/stretchr/testify/require"
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
	t.Parallel()

	sid := testIDb4nny
	router := createRouter(testStore)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bans?steam_id=%d", testIDb4nny.Int64()), nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	var banState []steamweb.PlayerBanState

	require.NoError(t, json.Unmarshal(body, &banState))
	require.Equal(t, sid, banState[0].SteamID)
}

func TestGetSummary(t *testing.T) {
	t.Parallel()

	sid := testIDb4nny
	router := createRouter(testStore)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/summary?steam_id=%d", testIDb4nny.Int64()), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	data, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	var bs []steamweb.PlayerSummary

	require.NoError(t, json.Unmarshal(data, &bs))
	require.Equal(t, sid, bs[0].SteamID)
}

func TestGetProfile(t *testing.T) {
	t.Parallel()

	sid := testIDb4nny
	router := createRouter(testStore)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/profile?steam_id=%d", testIDb4nny.Int64()), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	data, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	var profile Profile

	require.NoError(t, json.Unmarshal(data, &profile))
	require.Equal(t, steamweb.EconBanNone, profile.BanState.EconomyBan)
	require.Equal(t, sid, profile.Summary.SteamID)
}
