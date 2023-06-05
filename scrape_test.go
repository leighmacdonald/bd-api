package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func testParser(t *testing.T, path string, scraper *Scraper, count int, nextPage string) {
	testBody, errOpen := os.Open(path)
	require.NoError(t, errOpen)
	defer logCloser(testBody)
	doc, errDoc := goquery.NewDocumentFromReader(testBody)
	require.NoError(t, errDoc)

	next, results, errParse := scraper.parser(doc.Selection, scraper.nextUrl, scraper.parseTIme, nil)
	require.NoError(t, errParse)
	require.Equal(t, count, len(results))
	require.Equal(t, nextPage, next)
	for _, d := range results {
		require.Truef(t, d.SteamId.Valid(), "Invalid steamid: %s", d.SteamId.String())
	}
}

func TestParseSkial(t *testing.T) {
	testParser(t, "test_data/skial_home.html", NewSkialScraper(), 50, "index.php?p=banlist&page=2")
}

func TestParseUGC(t *testing.T) {
	testParser(t, "test_data/ugc_home.html", NewUGCScraper(), 50, "index.php?p=banlist&page=2")
}

func TestParseWonderland(t *testing.T) {
	testParser(t, "test_data/wonderland_home.html", NewWonderlandTFScraper(), 23, "index.php?p=banlist&page=2")
}

func TestParseGFL(t *testing.T) {
	testParser(t, "test_data/gfl_home.html", NewGFLScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParsePancakes(t *testing.T) {
	testParser(t, "test_data/pancakes_home.html", NewPancakesScraper(), 10, "index.php?p=banlist&page=2")
}

func TestParseOWL(t *testing.T) {
	testParser(t, "test_data/owl_home.html", NewOwlTFScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseSpaceShip(t *testing.T) {
	testParser(t, "test_data/ss_home.html", NewSpaceShipScraper(), 69, "index.php?p=banlist&page=2")
}

func TestParseLazyPurple(t *testing.T) {
	testParser(t, "test_data/lp_home.html", NewLazyPurpleScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseFirePowered(t *testing.T) {
	testParser(t, "test_data/firepowered_home.html", NewFirePoweredScraper(), 28, "index.php?p=banlist&page=2")
}

func TestParseHarpoon(t *testing.T) {
	testParser(t, "test_data/harpoon_home.html", NewHarpoonScraper(), 38, "index.php?p=banlist&page=2")
}

func TestParsePanda(t *testing.T) {
	testParser(t, "test_data/panda_home.html", NewPandaScraper(), 40, "index.php?p=banlist&page=2")
}

func TestParseNeonHeights(t *testing.T) {
	testParser(t, "test_data/neonheights_home.html", NewNeonHeightsScraper(), 28, "index.php?p=banlist&page=2")
}

func TestParseLOOS(t *testing.T) {
	testParser(t, "test_data/loos_home.html", NewLOOSScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParsePubsTF(t *testing.T) {
	testParser(t, "test_data/pubstf_home.html", NewPubsTFScraper(), 29, "index.php?p=banlist&page=2")
}

func TestParseScrapTF(t *testing.T) {
	testParser(t, "test_data/scraptf_home.html", NewScrapTFScraper(), 30, "index.php?p=banlist&page=2")
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
