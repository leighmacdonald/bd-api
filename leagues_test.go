package main

import (
	"context"
	"testing"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var (
	testIDb4nny  = steamid.New("76561197970669109")
	testIDCamper = steamid.New("76561197992870439")
)

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
