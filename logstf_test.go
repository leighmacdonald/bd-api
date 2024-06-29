package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestParseLogsTFDuration(t *testing.T) {
	ms, errMS := parseLogsTFDuration("11:22")
	require.NoError(t, errMS)
	require.Equal(t, "11m22s", ms.String())

	hms, errHMS := parseLogsTFDuration("79:04")
	require.NoError(t, errHMS)
	require.Equal(t, "1h19m4s", hms.String())
}

func TestLogsTFDetails(t *testing.T) {
	body, errRead := os.Open("testdata/logstf_detail.html")
	require.NoError(t, errRead)
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	require.NoError(t, err)

	match, errDetails := newDetailsFromDoc(doc)
	require.NoError(t, errDetails)

	require.Equal(t, "Qixalite Booking: RED vs BLU", match.Title)
	require.Equal(t, "koth_cascade_rc2", match.Map)
	require.Equal(t, "16m56s", match.Duration.String())
	require.Equal(t, "2022-02-05 06:39:42 +0000 UTC", match.CreatedOn.String())
	require.Equal(t, 0, match.ScoreBLU)
	require.Equal(t, 3, match.ScoreRED)

}

func TestLogsTFDetailsOld(t *testing.T) {
	body, errRead := os.Open("testdata/logstf_detail_old.html")
	require.NoError(t, errRead)
	defer body.Close()
}
