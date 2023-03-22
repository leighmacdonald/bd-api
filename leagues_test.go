package main

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	testIDb4nny    = steamid.SID64(76561197970669109)
	testIDSquirrel = steamid.SID64(76561197961279983)
)

func TestGetLogsTF(t *testing.T) {
	count, errLogs := getLogsTF(context.Background(), testIDb4nny)
	require.NoError(t, errLogs)
	require.Less(t, int64(13000), count)

	countZero, errLogsZero := getLogsTF(context.Background(), testIDb4nny+2)
	require.NoError(t, errLogsZero)
	require.Equal(t, int64(0), countZero)
}

func TestGetUGC(t *testing.T) {
	seasons, errLogs := getUGC(context.Background(), testIDb4nny)
	require.NoError(t, errLogs)
	require.GreaterOrEqual(t, 30, len(seasons))

}

func TestETF2L(t *testing.T) {
	c, cancel := context.WithTimeout(context.Background(), time.Second*25)
	defer cancel()
	seasons, err := getETF2L(c, testIDb4nny)
	require.NoError(t, err)
	require.Greater(t, len(seasons), 3)
}
