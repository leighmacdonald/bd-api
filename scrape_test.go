package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func testParser(t *testing.T, path string, parser parserFunc, urlFunc nextUrlFunc, timeFunc parseTimeFunc, count int, nextPage string) {
	testBody, errOpen := os.Open(path)
	require.NoError(t, errOpen)
	defer logCloser(testBody)
	doc, errDoc := goquery.NewDocumentFromReader(testBody)
	require.NoError(t, errDoc)

	next, results, errParse := parser(doc.Selection, urlFunc, timeFunc, nil)
	require.NoError(t, errParse)
	require.Equal(t, count, len(results))
	require.Equal(t, nextPage, next)
	for _, d := range results {
		require.Truef(t, d.SteamId.Valid(), "Invalid steamid: %s", d.SteamId.String())
	}
}

func TestParseSkial(t *testing.T) {
	testParser(t, "test_data/skial_home.html", parseDefault, nextUrlFirst, parseSkialTime, 50, "index.php?p=banlist&page=2")
}

func TestParseUGC(t *testing.T) {
	testParser(t, "test_data/ugc_home.html", parseFluent, nextUrlFluent, parseDefaultTime, 50, "index.php?p=banlist&page=2")
}

func TestParseWonderland(t *testing.T) {
	testParser(t, "test_data/wonderland_home.html", parseDefault, nextUrlLast, parseWonderlandTime, 30, "index.php?p=banlist&page=2")
}

func TestParseGFL(t *testing.T) {
	testParser(t, "test_data/gfl_home.html", parseDefault, nextUrlLast, parseDefaultTime, 30, "index.php?p=banlist&page=2")
}

func TestParsePancakes(t *testing.T) {
	testParser(t, "test_data/pancakes_home.html", parseDefault, nextUrlLast, parsePancakesTime, 10, "index.php?p=banlist&page=2")
}

func TestParseOWL(t *testing.T) {
	testParser(t, "test_data/owl_home.html", parseDefault, nextUrlLast, parseDefaultTime, 30, "index.php?p=banlist&page=2")
}

func TestParseSpaceShip(t *testing.T) {
	testParser(t, "test_data/ss_home.html", parseDefault, nextUrlLast, parseDefaultTime, 69, "index.php?p=banlist&page=2")
}

func TestParseLazyPurple(t *testing.T) {
	testParser(t, "test_data/lp_home.html", parseDefault, nextUrlLast, parseDefaultTime, 30, "index.php?p=banlist&page=2")
}

func TestParseFirePowered(t *testing.T) {
	testParser(t, "test_data/firepowered_home.html", parseDefault, nextUrlLast, parseSkialTime, 30, "index.php?p=banlist&page=2")
}

func TestParseHarpoon(t *testing.T) {
	testParser(t, "test_data/harpoon_home.html", parseDefault, nextUrlLast, parseDefaultTime, 50, "index.php?p=banlist&page=2")
}

func TestParsePanda(t *testing.T) {
	testParser(t, "test_data/panda_home.html", parseDefault, nextUrlLast, parseSkialTime, 50, "index.php?p=banlist&page=2")
}

func TestParseNeonHeights(t *testing.T) {
	testParser(t, "test_data/neonheights_home.html", parseDefault, nextUrlLast, parseSkialTime, 30, "index.php?p=banlist&page=2")
}

func TestParseLOOS(t *testing.T) {
	testParser(t, "test_data/loos_home.html", parseDefault, nextUrlLast, parseDefaultTime, 30, "index.php?p=banlist&page=2")
}

func TestParsePubsTF(t *testing.T) {
	testParser(t, "test_data/pubstf_home.html", parseDefault, nextUrlLast, parseSkialTime, 30, "index.php?p=banlist&page=2")
}

func TestParseGFLTime(t *testing.T) {
	parsed, e := parseDefaultTime("2023-05-17 03:07:05")
	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 5, 0, time.UTC), parsed)
}

func TestParseWonderlandTime(t *testing.T) {
	parsed, e := parseWonderlandTime("May 17th, 2023 (3:07)")
	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 0, 0, time.UTC), parsed)
}

func TestParseSkialTime(t *testing.T) {
	parsed, e := parseSkialTime("05-17-23 03:07")
	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 0, 0, time.UTC), parsed)
	perm, ePerm := parseSkialTime("Permanent")
	require.NoError(t, ePerm)
	require.Equal(t, time.Time{}, perm)
}

func TestParsePancakesTime(t *testing.T) {
	parsed, e := parsePancakesTime("Thu, May 17, 2023 3:07 AM")
	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 0, 0, time.UTC), parsed)
	perm, ePerm := parsePancakesTime("never, this is permanent")
	require.NoError(t, ePerm)
	require.Equal(t, time.Time{}, perm)
}

func TestParseMegaScatter(t *testing.T) {
	testBody, errOpen := os.Open("test_data/megascatter.html")
	require.NoError(t, errOpen)
	defer logCloser(testBody)
	bans, errBans := parseMegaScatter(testBody)
	require.NoError(t, errBans)
	require.True(t, len(bans) > 100)
}
