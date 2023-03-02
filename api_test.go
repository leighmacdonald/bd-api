package main

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	testIdb4nny = steamid.SID64(76561197970669109)
)

func TestGetLogsTF(t *testing.T) {
	count, errLogs := getLogsTF(context.Background(), testIdb4nny)
	require.NoError(t, errLogs)
	require.Less(t, int64(13000), count)

	countZero, errLogsZero := getLogsTF(context.Background(), testIdb4nny+2)
	require.NoError(t, errLogsZero)
	require.Equal(t, int64(0), countZero)
}

func TestGetUGC(t *testing.T) {
	count, errLogs := getUGC(context.Background(), testIdb4nny)
	require.NoError(t, errLogs)
	require.Less(t, int64(13000), count)

}

func TestETF2L(t *testing.T) {
	c, cancel := context.WithTimeout(context.Background(), time.Second*25)
	defer cancel()
	seasons, err := getETF2L(c, 76561198004469267)
	require.NoError(t, err)
	require.Greater(t, len(seasons), 2)
}
