package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func testParser(t *testing.T, scraper *Scraper, count int, nextPage string) {
	testBody, errOpen := os.Open(fmt.Sprintf("test_data/%s.html", scraper.name))
	require.NoError(t, errOpen)
	defer logCloser(testBody)
	doc, errDoc := goquery.NewDocumentFromReader(testBody)
	require.NoError(t, errDoc)

	next, results, errParse := scraper.parser(doc.Selection, scraper.nextUrl, scraper.parseTIme)
	require.NoError(t, errParse)
	require.Equal(t, count, len(results))
	require.Equal(t, nextPage, next)
	for _, d := range results {
		require.Truef(t, d.SteamId.Valid(), "Invalid steamid: %s", d.SteamId.String())
	}
}

func TestParseSkial(t *testing.T) {
	testParser(t, NewSkialScraper(), 50, "index.php?p=banlist&page=2")
}

func TestParseUGC(t *testing.T) {
	testParser(t, NewUGCScraper(), 50, "index.php?p=banlist&page=2")
}

func TestParseWonderland(t *testing.T) {
	testParser(t, NewWonderlandTFScraper(), 23, "index.php?p=banlist&page=2")
}

func TestParseGFL(t *testing.T) {
	testParser(t, NewGFLScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParsePancakes(t *testing.T) {
	testParser(t, NewPancakesScraper(), 10, "index.php?p=banlist&page=2")
}

func TestParseOWL(t *testing.T) {
	testParser(t, NewOwlTFScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseSpaceShip(t *testing.T) {
	testParser(t, NewSpaceShipScraper(), 69, "index.php?p=banlist&page=2")
}

func TestParseLazyPurple(t *testing.T) {
	testParser(t, NewLazyPurpleScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseFirePowered(t *testing.T) {
	testParser(t, NewFirePoweredScraper(), 28, "index.php?p=banlist&page=2")
}

func TestParseHarpoon(t *testing.T) {
	testParser(t, NewHarpoonScraper(), 38, "index.php?p=banlist&page=2")
}

func TestParsePanda(t *testing.T) {
	testParser(t, NewPandaScraper(), 40, "index.php?p=banlist&page=2")
}

func TestParseNeonHeights(t *testing.T) {
	testParser(t, NewNeonHeightsScraper(), 28, "index.php?p=banlist&page=2")
}

func TestParseLOOS(t *testing.T) {
	testParser(t, NewLOOSScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParsePubsTF(t *testing.T) {
	testParser(t, NewPubsTFScraper(), 29, "index.php?p=banlist&page=2")
}

func TestParseScrapTF(t *testing.T) {
	testParser(t, NewScrapTFScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseServiliveCl(t *testing.T) {
	testParser(t, NewServiliveClScraper(), 27, "index.php?p=banlist&page=2")
}

func TestParseZMBrasil(t *testing.T) {
	testParser(t, NewZMBrasilScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseSirPlease(t *testing.T) {
	testParser(t, NewSirPleaseScraper(), 30, "index.php?p=banlist&page=2")
}

func TestVidyaGaems(t *testing.T) {
	testParser(t, NewVidyaGaemsScraper(), 30, "index.php?p=banlist&page=2")
}

func TestSGGaming(t *testing.T) {
	testParser(t, NewSGGamingScraper(), 50, "index.php?p=banlist&page=2")
}

func TestApeMode(t *testing.T) {
	testParser(t, NewApeModeScraper(), 30, "index.php?p=banlist&page=2")
}

func TestMaxDB(t *testing.T) {
	testParser(t, NewMaxDBScraper(), 30, "index.php?p=banlist&page=2")
}

func TestSvdosBrothers(t *testing.T) {
	testParser(t, NewSvdosBrothersScraper(), 27, "index.php?p=banlist&page=2")
}

func TestElectric(t *testing.T) {
	testParser(t, NewElectricScraper(), 24, "index.php?p=banlist&page=2")
}

func TestGlobalParadise(t *testing.T) {
	testParser(t, NewGlobalParadiseScraper(), 25, "index.php?p=banlist&page=2")
}

func TestSavageServidores(t *testing.T) {
	testParser(t, NewSavageServidoresScraper(), 29, "index.php?p=banlist&page=2")
}

func TestCSIServers(t *testing.T) {
	testParser(t, NewCSIServersScraper(), 30, "index.php?p=banlist&page=2")
}

func TestLBGaming(t *testing.T) {
	testParser(t, NewLBGamingScraper(), 30, "index.php?p=banlist&page=2")
}

func TestFluxTF(t *testing.T) {
	testParser(t, NewFluxTFScraper(), 30, "index.php?p=banlist&page=2")
}

func TestCutiePie(t *testing.T) {
	testParser(t, NewCutiePieScraper(), 30, "index.php?p=banlist&page=2")
}

func TestDarkPyro(t *testing.T) {
	testParser(t, NewDarkPyroScraper(), 16, "index.php?p=banlist&page=2")
}

func TestOpstOnline(t *testing.T) {
	testParser(t, NewOpstOnlineScraper(), 30, "index.php?p=banlist&page=2")
}

func TestBouncyBall(t *testing.T) {
	testParser(t, NewBouncyBallScraper(), 50, "index.php?p=banlist&page=2")
}

func TestFurryPound(t *testing.T) {
	testParser(t, NewFurryPoundScraper(), 30, "index.php?p=banlist&page=2")
}

func TestRetroServers(t *testing.T) {
	testParser(t, NewRetroServersScraper(), 30, "index.php?p=banlist&page=2")
}

func TestSwapShop(t *testing.T) {
	testParser(t, NewSwapShopScraper(), 77, "index.php?p=banlist&page=2")
}

func TestECJ(t *testing.T) {
	testParser(t, NewECJScraper(), 30, "index.php?p=banlist&page=2")
}

func TestJumpAcademy(t *testing.T) {
	testParser(t, NewJumpAcademyScraper(), 30, "index.php?p=banlist&page=2")
}

func TestTF2RO(t *testing.T) {
	testParser(t, NewTF2ROScraper(), 21, "index.php?p=banlist&hideinactive=true")
}

func TestSameTeem(t *testing.T) {
	testParser(t, NewSameTeemScraper(), 30, "index.php?p=banlist&page=2")
}

func TestPowerFPS(t *testing.T) {
	testParser(t, NewPowerFPSScraper(), 28, "index.php?p=banlist&page=2")
}

func Test7Mau(t *testing.T) {
	testParser(t, New7MauScraper(), 30, "index.php?p=banlist&page=2")
}

func TestGhostCap(t *testing.T) {
	testParser(t, NewGhostCapScraper(), 28, "index.php?p=banlist&page=2")
}

func TestSpectre(t *testing.T) {
	testParser(t, NewSpectreScraper(), 29, "index.php?p=banlist&page=2")
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
