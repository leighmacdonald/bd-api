package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

func testParser(t *testing.T, scraper *sbScraper, count int, nextPage string) {
	t.Helper()

	testBody, errOpen := os.Open(fmt.Sprintf("test_data/%s.html", scraper.name))

	require.NoError(t, errOpen)

	defer logCloser(testBody)

	doc, errDoc := goquery.NewDocumentFromReader(testBody)

	require.NoError(t, errDoc)

	results, _, errParse := scraper.parser(doc.Selection, scraper.parseTIme, scraper.name)

	require.NoError(t, errParse)
	require.Equal(t, count, len(results))

	if nextPage != "" {
		next := scraper.nextURL(scraper, doc.Selection)
		require.Equal(t, scraper.url(nextPage), next)
	}

	for _, d := range results {
		require.NotEqual(t, "", d.Name)
		require.Truef(t, d.SteamID.Valid(), "Invalid steamid: %s", d.SteamID.String())
		require.True(t, d.Length >= 0, "negative duration")
	}
}

func TestParseSkial(t *testing.T) {
	t.Parallel()
	testParser(t, newSkialScraper(), 48, "index.php?p=banlist&page=2")
}

func TestParseUGC(t *testing.T) {
	t.Parallel()
	testParser(t, newUGCScraper(), 49, "index.php?p=banlist&page=2")
}

func TestParseWonderland(t *testing.T) {
	t.Parallel()
	testParser(t, newWonderlandTFScraper(), 22, "index.php?p=banlist&page=2")
}

func TestParseWonderlandGoog(t *testing.T) {
	t.Parallel()
	testParser(t, newWonderlandTFGOOGScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseGFL(t *testing.T) {
	t.Parallel()
	testParser(t, newGFLScraper(), 28, "index.php?p=banlist&page=2")
}

func TestParsePancakes(t *testing.T) {
	t.Parallel()
	testParser(t, newPancakesScraper(), 10, "index.php?p=banlist&page=2")
}

func TestParseOWL(t *testing.T) {
	t.Parallel()
	testParser(t, newOwlTFScraper(), 22, "index.php?p=banlist&page=2")
}

func TestParseSpaceShip(t *testing.T) {
	t.Parallel()
	testParser(t, newSpaceShipScraper(), 69, "index.php?p=banlist&page=2")
}

func TestParseLazyPurple(t *testing.T) {
	t.Parallel()
	testParser(t, newLazyPurpleScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseFirePowered(t *testing.T) {
	t.Parallel()
	testParser(t, newFirePoweredScraper(), 26, "index.php?p=banlist&page=2")
}

func TestDixiGame(t *testing.T) {
	t.Parallel()
	testParser(t, newDixiGameScraper(), 23, "index.php?p=banlist&page=2")
}

func TestParseHarpoon(t *testing.T) {
	t.Parallel()
	testParser(t, newHarpoonScraper(), 38, "index.php?p=banlist&page=2")
}

func TestParsePanda(t *testing.T) {
	t.Parallel()
	testParser(t, newPandaScraper(), 40, "index.php?p=banlist&page=2")
}

func TestParseNeonHeights(t *testing.T) {
	t.Parallel()
	testParser(t, newNeonHeightsScraper(), 28, "index.php?p=banlist&page=2")
}

func TestParseLOOS(t *testing.T) {
	t.Parallel()
	testParser(t, newLOOSScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParsePubsTF(t *testing.T) {
	t.Parallel()
	testParser(t, newPubsTFScraper(), 26, "index.php?p=banlist&page=2")
}

func TestParseScrapTF(t *testing.T) {
	t.Parallel()
	testParser(t, newScrapTFScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseServiliveCl(t *testing.T) {
	t.Parallel()
	testParser(t, newServiliveClScraper(), 27, "index.php?p=banlist&page=2")
}

func TestParseZMBrasil(t *testing.T) {
	t.Parallel()
	testParser(t, newZMBrasilScraper(), 30, "index.php?p=banlist&page=2")
}

func TestParseSirPlease(t *testing.T) {
	t.Parallel()
	testParser(t, newSirPleaseScraper(), 30, "index.php?p=banlist&page=2")
}

func TestVidyaGaems(t *testing.T) {
	t.Parallel()
	testParser(t, newVidyaGaemsScraper(), 30, "index.php?p=banlist&page=2")
}

func TestSGGaming(t *testing.T) {
	t.Parallel()
	testParser(t, newSGGamingScraper(), 50, "index.php?p=banlist&page=2")
}

func TestApeMode(t *testing.T) {
	t.Parallel()
	testParser(t, newApeModeScraper(), 30, "index.php?p=banlist&page=2")
}

func TestMaxDB(t *testing.T) {
	t.Parallel()
	testParser(t, newMaxDBScraper(), 26, "index.php?p=banlist&page=2")
}

func TestSvdosBrothers(t *testing.T) {
	t.Parallel()
	testParser(t, newSvdosBrothersScraper(), 27, "index.php?p=banlist&page=2")
}

func TestElectric(t *testing.T) {
	t.Parallel()
	testParser(t, newElectricScraper(), 24, "index.php?p=banlist&page=2")
}

func TestGlobalParadise(t *testing.T) {
	t.Parallel()
	testParser(t, newGlobalParadiseScraper(), 23, "index.php?p=banlist&page=2")
}

func TestSavageServidores(t *testing.T) {
	t.Parallel()
	testParser(t, newSavageServidoresScraper(), 29, "index.php?p=banlist&page=2")
}

func TestCSIServers(t *testing.T) {
	t.Parallel()
	testParser(t, newCSIServersScraper(), 30, "index.php?p=banlist&page=2")
}

func TestLBGaming(t *testing.T) {
	t.Parallel()
	testParser(t, newLBGamingScraper(), 29, "index.php?p=banlist&page=2")
}

func TestFluxTF(t *testing.T) {
	t.Parallel()
	testParser(t, newFluxTFScraper(), 29, "index.php?p=banlist&page=2")
}

func TestCutiePie(t *testing.T) {
	t.Parallel()
	testParser(t, newCutiePieScraper(), 30, "index.php?p=banlist&page=2")
}

func TestDarkPyro(t *testing.T) {
	t.Parallel()
	testParser(t, newDarkPyroScraper(), 16, "index.php?p=banlist&page=2")
}

func TestOpstOnline(t *testing.T) {
	t.Parallel()
	testParser(t, newOpstOnlineScraper(), 30, "index.php?p=banlist&page=2")
}

func TestBouncyBall(t *testing.T) {
	t.Parallel()
	testParser(t, newBouncyBallScraper(), 49, "index.php?p=banlist&page=2")
}

func TestFurryPound(t *testing.T) {
	t.Parallel()
	testParser(t, newFurryPoundScraper(), 30, "index.php?p=banlist&page=2")
}

func TestRetroServers(t *testing.T) {
	t.Parallel()
	testParser(t, newRetroServersScraper(), 30, "index.php?p=banlist&page=2")
}

func TestSwapShop(t *testing.T) {
	t.Parallel()
	testParser(t, newSwapShopScraper(), 76, "index.php?p=banlist&page=2")
}

func TestECJ(t *testing.T) {
	t.Parallel()
	testParser(t, newECJScraper(), 30, "index.php?p=banlist&page=2")
}

func TestJumpAcademy(t *testing.T) {
	t.Parallel()
	testParser(t, newJumpAcademyScraper(), 30, "index.php?p=banlist&page=2")
}

func TestTF2RO(t *testing.T) {
	t.Parallel()
	testParser(t, newTF2ROScraper(), 21, "")
}

func TestSameTeem(t *testing.T) {
	t.Parallel()
	testParser(t, newSameTeemScraper(), 30, "index.php?p=banlist&page=2")
}

func TestPowerFPS(t *testing.T) {
	t.Parallel()
	testParser(t, newPowerFPSScraper(), 28, "index.php?p=banlist&page=2")
}

func Test7Mau(t *testing.T) {
	t.Parallel()
	testParser(t, new7MauScraper(), 30, "index.php?p=banlist&page=2")
}

func TestGhostCap(t *testing.T) {
	t.Parallel()
	testParser(t, newGhostCapScraper(), 28, "index.php?p=banlist&page=2")
}

func TestSpectre(t *testing.T) {
	t.Parallel()
	testParser(t, newSpectreScraper(), 29, "index.php?p=banlist&page=2")
}

func TestDreamFire(t *testing.T) {
	t.Parallel()
	testParser(t, newDreamFireScraper(), 29, "index.php?p=banlist&page=2")
}

func TestSetti(t *testing.T) {
	t.Parallel()
	testParser(t, newSettiScraper(), 25, "index.php?p=banlist&page=2")
}

func TestGunServer(t *testing.T) {
	t.Parallel()
	testParser(t, newGunServerScraper(), 30, "index.php?p=banlist&page=2")
}

func TestHellClan(t *testing.T) {
	t.Parallel()
	testParser(t, newHellClanScraper(), 59, "index.php?p=banlist&page=2")
}

func TestSneaks(t *testing.T) {
	t.Parallel()
	testParser(t, newSneaksScraper(), 30, "index.php?p=banlist&page=2")
}

func TestNide(t *testing.T) {
	t.Parallel()
	testParser(t, newNideScraper(), 20, "index.php?p=banlist&page=2")
}

func TestAstraMania(t *testing.T) {
	t.Parallel()
	testParser(t, newAstraManiaScraper(), 38, "index.php?p=banlist&page=2")
}

func TestTF2Maps(t *testing.T) {
	t.Parallel()
	testParser(t, newTF2MapsScraper(), 56, "index.php?p=banlist&page=2")
}

func TestPetrolTF(t *testing.T) {
	t.Parallel()
	testParser(t, newPetrolTFScraper(), 98, "index.php?p=banlist&page=2")
}

func TestVaticanCity(t *testing.T) {
	t.Parallel()
	testParser(t, newVaticanCityScraper(), 50, "index.php?p=banlist&page=2")
}

func TestLazyNeer(t *testing.T) {
	t.Parallel()
	testParser(t, newLazyNeerScraper(), 30, "index.php?p=banlist&page=2")
}

func TestTheVille(t *testing.T) {
	t.Parallel()
	testParser(t, newTheVilleScraper(), 48, "index.php?p=banlist&page=2")
}

func TestOreon(t *testing.T) {
	t.Parallel()
	testParser(t, newOreonScraper(), 30, "index.php?p=banlist&page=2")
}

func TestTriggerHappy(t *testing.T) {
	t.Parallel()
	testParser(t, newTriggerHappyScraper(), 27, "index.php?p=banlist&page=2")
}

func TestDefuseRo(t *testing.T) {
	t.Parallel()
	testParser(t, newDefuseRoScraper(), 25, "index.php?p=banlist&page=2")
}

func TestTawerna(t *testing.T) {
	t.Parallel()
	testParser(t, newTawernaScraper(), 30, "index.php?p=banlist&page=2")
}

func TestTitan(t *testing.T) {
	t.Parallel()
	testParser(t, newTitanScraper(), 30, "index.php?p=banlist&page=2")
}

func TestDiscFF(t *testing.T) {
	t.Parallel()
	testParser(t, newDiscFFScraper(), 29, "index.php?p=banlist&page=2")
}

// func TestOtaku(t *testing.T) {
//	testParser(t, NewOtakuScraper(), 30, "index.php?p=banlist&page=2")
//}

func TestAMSGaming(t *testing.T) {
	t.Parallel()
	testParser(t, newAMSGamingScraper(), 29, "index.php?p=banlist&page=2")
}

func TestBaitedCommunity(t *testing.T) {
	t.Parallel()
	testParser(t, newBaitedCommunityScraper(), 28, "index.php?p=banlist&page=2")
}

func TestCedaPugCommunity(t *testing.T) {
	t.Parallel()
	testParser(t, newCedaPugScraper(), 30, "index.php?p=banlist&page=2")
}

func TestGameSitesCommunity(t *testing.T) {
	t.Parallel()
	testParser(t, newGameSitesScraper(), 30, "index.php?p=banlist&page=2")
}

func TestBachuruServasCommunity(t *testing.T) {
	t.Parallel()
	testParser(t, newBachuruServasScraper(), 26, "index.php?p=banlist&page=2")
}

func TestBierwieseCommunity(t *testing.T) {
	t.Parallel()
	testParser(t, newBierwieseScraper(), 30, "index.php?p=banlist&page=2")
}

func TestAceKillCommunity(t *testing.T) {
	t.Parallel()
	testParser(t, newAceKillScraper(), 30, "index.php?p=banlist&page=2")
}

func TestMagyarhns(t *testing.T) {
	t.Parallel()
	testParser(t, newMagyarhnsScraper(), 27, "index.php?p=banlist&page=2")
}

func TestGamesTown(t *testing.T) {
	t.Parallel()
	testParser(t, newGamesTownScraper(), 29, "index.php?p=banlist&page=2")
}

func TestProGamesZet(t *testing.T) {
	t.Parallel()
	testParser(t, newProGamesZetScraper(), 16, "index.php?p=banlist&page=2")
}

func TestG44(t *testing.T) {
	t.Parallel()
	testParser(t, newG44Scraper(), 52, "index.php?p=banlist&page=2")
}

func TestCuteProject(t *testing.T) {
	t.Parallel()
	testParser(t, newCuteProjectScraper(), 12, "index.php?p=banlist&page=2")
}

func TestPhoenixSource(t *testing.T) {
	t.Parallel()
	testParser(t, newPhoenixSourceScraper(), 19, "index.php?p=banlist&page=2")
}

func TestSlavonServer(t *testing.T) {
	t.Parallel()
	testParser(t, newSlavonServerScraper(), 26, "index.php?p=banlist&page=2")
}

func TestGetSome(t *testing.T) {
	t.Parallel()
	testParser(t, newGetSomeScraper(), 30, "index.php?p=banlist&page=2")
}

func TestRushy(t *testing.T) {
	t.Parallel()
	testParser(t, newRushyScraper(), 20, "index.php?p=banlist&page=2")
}

func TestMoevsMachine(t *testing.T) {
	t.Parallel()
	testParser(t, newMoevsMachineScraper(), 27, "index.php?p=banlist&page=2")
}

func TestPRWH(t *testing.T) {
	t.Parallel()
	testParser(t, newPRWHScraper(), 30, "index.php?p=banlist&page=2")
}

func TestVortex(t *testing.T) {
	t.Parallel()
	testParser(t, newVortexScraper(), 46, "index.php?p=banlist&page=2")
}

func TestCasualFun(t *testing.T) {
	t.Parallel()
	testParser(t, newCasualFunScraper(), 30, "index.php?p=banlist&page=2")
}

func TestRandomTF2(t *testing.T) {
	t.Parallel()
	testParser(t, newRandomTF2Scraper(), 3, "")
}

func TestPlayesRO(t *testing.T) {
	t.Parallel()
	testParser(t, newPlayesROScraper(), 29, "index.php?p=banlist&page=2")
}

func TestEOTLGaming(t *testing.T) {
	t.Parallel()
	testParser(t, newEOTLGamingScraper(), 30, "index.php?p=banlist&page=2")
}

func TestBioCrafting(t *testing.T) {
	t.Parallel()
	testParser(t, newBioCraftingScraper(), 30, "index.php?p=banlist&page=2")
}

func TestBigBangGamers(t *testing.T) {
	t.Parallel()
	testParser(t, newBigBangGamersScraper(), 5, "")
}

func TestEpicZone(t *testing.T) {
	t.Parallel()
	testParser(t, newEpicZoneScraper(), 30, "index.php?p=banlist&page=2")
}

func TestZubat(t *testing.T) {
	t.Parallel()
	testParser(t, newZubatScraper(), 30, "index.php?p=banlist&page=2")
}

func TestLunario(t *testing.T) {
	t.Parallel()
	testParser(t, newLunarioScraper(), 15, "index.php?p=banlist&page=2")
}

func TestParseGFLTime(t *testing.T) {
	t.Parallel()

	parsed, e := parseDefaultTime("2023-05-17 03:07:05")

	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 5, 0, time.UTC), parsed)
}

func TestParseWonderlandTime(t *testing.T) {
	t.Parallel()

	parsed, e := parseWonderlandTime("May 17th, 2023 (3:07)")

	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 0, 0, time.UTC), parsed)
}

func TestParseSkialTime(t *testing.T) {
	t.Parallel()

	parsed, e := parseSkialTime("05-17-23 03:07")

	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 0, 0, time.UTC), parsed)

	perm, ePerm := parseSkialTime("Permanent")

	require.NoError(t, ePerm)
	require.Equal(t, time.Time{}, perm)
}

func TestParsePancakesTime(t *testing.T) {
	t.Parallel()

	parsed, e := parsePancakesTime("Thu, May 17, 2023 3:07 AM")

	require.NoError(t, e)
	require.Equal(t, time.Date(2023, time.May, 17, 3, 7, 0, 0, time.UTC), parsed)

	perm, ePerm := parsePancakesTime("never, this is permanent")

	require.NoError(t, ePerm)
	require.Equal(t, time.Time{}, perm)
}

func TestParseFluxTFTime(t *testing.T) {
	t.Parallel()

	parsed, e := parseFluxTime("Tuesday 30th of August 2022 08:30:45 PM")

	require.NoError(t, e)
	require.Equal(t, time.Date(2022, time.August, 30, 20, 30, 45, 0, time.UTC), parsed)
}

// func TestParseMegaScatter(t *testing.T) {
//	testBody, errOpen := os.Open("test_data/megascatter.html")
//	require.NoError(t, errOpen)
//	defer logCloser(testBody)
//	bans, errBans := parseMegaScatter(testBody)
//	require.NoError(t, errBans)
//	require.True(t, len(bans) > 100)
//}
