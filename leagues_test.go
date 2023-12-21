package main

import (
	"context"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	testIDb4nny  steamid.SID64 = "76561197970669109"
	testIDCamper steamid.SID64 = "76561197992870439"
	testIDBanned steamid.SID64 = "76561198203516436"
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
