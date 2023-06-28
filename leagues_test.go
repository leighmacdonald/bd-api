package main

import (
	"context"
	"testing"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
)

var (
	testIDb4nny = steamid.New(76561197970669109)

	testIDCamper = steamid.New(76561197992870439)
)

func TestGetLogsTF(t *testing.T) {
	t.Parallel()

	count, errLogs := getLogsTF(context.Background(), testIDb4nny)

	require.NoError(t, errLogs)
	require.Less(t, int64(13000), count)

	countZero, errLogsZero := getLogsTF(context.Background(), steamid.New(testIDb4nny.Int64()+2))
	require.NoError(t, errLogsZero)
	require.Equal(t, int64(0), countZero)
}

func TestGetUGC(t *testing.T) {
	t.Parallel()

	seasons, errLogs := getUGC(context.Background(), testIDb4nny)

	require.NoError(t, errLogs)
	require.GreaterOrEqual(t, 30, len(seasons))
}

func TestETF2L(t *testing.T) {
	t.Parallel()

	c, cancel := context.WithTimeout(context.Background(), time.Second*25)
	defer cancel()

	seasons, err := getETF2L(c, testIDb4nny)

	require.NoError(t, err)
	require.Greater(t, len(seasons), 3)
}

func TestRGL(t *testing.T) {
	t.Parallel()

	seasons, errSeasons := getRGL(context.Background(), steamid.New(76561198084134025))
	require.NoError(t, errSeasons)
	require.LessOrEqual(t, 1, len(seasons))
}
