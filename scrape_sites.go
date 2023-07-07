package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func newSkialScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "skial", "https://www.skial.com/sourcebans/", "",
		parseDefault, nextURLFirst, parseSkialTime, nil,
	)
}

func newGFLScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gfl", "https://sourcebans.gflclan.com/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSpaceShipScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "spaceship", "https://sappho.io/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newUGCScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "ugc", "https://sb.ugc-gaming.net/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newSirPleaseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sirplease", "https://sirplease.gg/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newVidyaGaemsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "vidyagaems", "https://www.vidyagaems.net/sourcebans/", "",
		parseFluent, nextURLFluent, parseTrailYear, nil)
}

func newOwlTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "owl", "https://kingpandagamer.xyz/sb/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newZMBrasilScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "zmbrasil", "http://bans.zmbrasil.com.br/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newDixiGameScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "dixigame", "https://dixigame.com/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newScrapTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "scraptf", "https://bans.scrap.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newWonderlandTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 10

	transport := newCFTransport()

	if errOpen := transport.Open(context.Background()); errOpen != nil {
		return nil, errors.Wrap(errOpen, "Failed to setup browser")
	}

	scraper, errScraper := newScraper(logger, cacheDir,
		"wonderland",
		"https://bans.wonderland.tf/",
		"",
		parseDefault,
		nextURLLast,
		parseWonderlandTime,
		transport)
	if errScraper != nil {
		return nil, errScraper
	}

	scraper.sleepTime = siteSleepTime

	// Cached versions do not have a proper next link, so we have to generate one.
	scraper.nextURL = func(scraper *sbScraper, doc *goquery.Selection) string {
		scraper.curPage++

		return scraper.url(fmt.Sprintf("index.php?p=banlist&page=%d", scraper.curPage))
	}

	return scraper, nil
}

func newLazyPurpleScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lazypurple", "https://www.lazypurple.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newFirePoweredScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "firepowered", "https://firepoweredgaming.com/sourcebanspp/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newHarpoonScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "harpoongaming", "https://bans.harpoongaming.com/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newPandaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "panda", "https://bans.panda-community.com/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newNeonHeightsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "neonheights", "https://neonheights.xyz/bans/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newPancakesScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "pancakes", "https://pancakes.tf/", "",
		parseDefault, nextURLLast, parsePancakesTime, nil)
}

func newLOOSScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "loos", "https://looscommunity.com/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newPubsTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "pubstf", "https://bans.pubs.tf/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newServiliveClScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "servilivecl", "https://sourcebans.servilive.cl/", "",
		parseFluent, nextURLFluent, parseDefaultTimeMonthFirst, nil)
}

func newCutiePieScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "cutiepie", "https://bans.cutiepie.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSGGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sggaming", "https://sg-gaming.net/bans/", "",
		parseDefault, nextURLLast, parseSGGamingTime, nil)
}

func newApeModeScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "apemode", "https://sourcebans.apemode.tf/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newMaxDBScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "maxdb", "https://bans.maxdb.net/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSvdosBrothersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "svdosbrothers", "https://bans.svdosbrothers.com/", "",
		parseFluent, nextURLFluent, parseSVDos, nil)
}

func newElectricScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "electric", "http://168.181.184.179/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newGlobalParadiseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "globalparadise", "https://bans.theglobalparadise.org/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSavageServidoresScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "savageservidores", "https://bans.savageservidores.com/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newCSIServersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "csiservers", "https://bans.csiservers.com/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newLBGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lbgaming", "https://bans.lbgaming.co/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newFluxTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "fluxtf", "https://bans.flux.tf/", "",
		parseDefault, nextURLLast, parseFluxTime, nil)
}

func newDarkPyroScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "darkpyro", "https://bans.darkpyrogaming.com/", "",
		parseDefault, nextURLLast, parseDarkPyroTime, nil)
}

func newOpstOnlineScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "opstonline", "https://www.opstonline.com/bans/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newBouncyBallScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bouncyball", "https://www.bouncyball.eu/bans2/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newFurryPoundScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "furrypound", "http://sourcebans.thefurrypound.org/", "",
		parseDefault, nextURLLast, parseFurryPoundTime, nil)
}

func newRetroServersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "retroservers", "https://bans.retroservers.net/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSwapShopScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "swapshop", "http://tf2swapshop.com/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newECJScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "ecj", "https://ecj.tf/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newJumpAcademyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "jumpacademy", "https://bans.jumpacademy.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newTF2ROScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	// Not enough values to page yet...
	return newScraper(logger, cacheDir, "tf2ro", "https://bans.tf2ro.com/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSameTeemScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sameteem", "https://sameteem.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newPowerFPSScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "powerfps", "https://bans.powerfps.com/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func new7MauScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "7mau", "https://7-mau.com/server/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newGhostCapScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "ghostcap", "https://sourcebans.ghostcap.com/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSpectreScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "spectre", "https://spectre.gg/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newDreamFireScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "dreamfire", "https://sourcebans.dreamfire.fr/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newSettiScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "setti", "https://pong.setti.info/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newGunServerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gunserver", "https://gunserver.ru/sourcebans/", "",
		parseDefault, nextURLFirst, parseGunServer, nil)
}

func newHellClanScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "hellclan", "https://hellclan.co.uk/sourcebans/", "",
		parseDefault, nextURLLast, parseHellClanTime, nil)
}

func newSneaksScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sneaks", "https://bans.snksrv.com/", "",
		parseDefault, nextURLLast, parseSneakTime, nil)
}

func newNideScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "nide", "https://bans.nide.gg/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

func newAstraManiaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "astramania", "https://astramania.ro/sban2/", "",
		parseDefault, nextURLLast, parseTrailYear, nil)
}

func newTF2MapsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "tf2maps", "https://bans.tf2maps.net/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newPetrolTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "petroltf", "https://petrol.tf/sb/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newVaticanCityScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "vaticancity", "https://www.the-vaticancity.com/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newLazyNeerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lazyneer", "https://www.lazyneer.com/SourceBans/", "",
		parseDefault, nextURLLast, parseSkialAltTime, nil)
}

func newTheVilleScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "theville", "https://www.theville.org/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newOreonScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "oreon", "https://www.tf2-oreon.fr/sourceban/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newTriggerHappyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "triggerhappy", "https://triggerhappygamers.com/sourcebans/", "",
		parseDefault, nextURLLast, parseTriggerHappyTime, nil)
}

func newDefuseRoScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "defusero", "https://bans.defusero.org/", "",
		parseFluent, nextURLFluent, parseDefaultTime, nil)
}

// Has cloudflare.
// func newTawernaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
//	return newScraper(logger, cacheDir, "tawerna", "https://sb.tawerna.tf/", "",
//		parseDefault, nextURLLast, parseSkialTime)
// }

func newTitanScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "titan", "https://bans.titan.tf/", "",
		parseDefault, nextURLLast, parseTitanTime, nil)
}

func newDiscFFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "discff", "http://disc-ff.site.nfoservers.com/sourcebanstf2/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

// New theme
// func NewOtakuScraper( ) (*sbScraper, error){
//	return newScraper(logger, cacheDir, "otaku", "https://bans.otaku.tf/bans", "",
//		parseDefault, nextURLLast, parseOtakuTime)
//}

func newAMSGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "amsgaming", "https://bans.amsgaming.in/", "",
		parseStar, nextURLLast, parseAMSGamingTime, nil)
}

func newBaitedCommunityScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "baitedcommunity", "https://bans.baitedcommunity.com/", "",
		parseStar, nextURLLast, parseBaitedTime, nil)
}

func newCedaPugScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "cedapug", "https://cedapug.com/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newGameSitesScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gamesites", "https://banlist.gamesites.cz/tf2/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newBachuruServasScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bachuruservas", "https://bachuruservas.lt/sb/", "",
		parseStar, nextURLLast, parseBachuruServasTime, nil)
}

func newBierwieseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bierwiese", "http://94.249.194.218/sb/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newAceKillScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "acekill", "https://sourcebans.acekill.pl/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newMagyarhnsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "magyarhns", "https://magyarhns.hu/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newGamesTownScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gamestown", "https://banlist.games-town.eu/", "",
		parseStar, nextURLLast, parseTrailYear, nil)
}

func newProGamesZetScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "progameszet", "https://bans.progameszet.ru/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime, nil)
}

func newG44Scraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "g44", "http://bans.allmaps.g44.rocks/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime, nil)
}

func newCuteProjectScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 4

	scraper, errScraper := newScraper(logger, cacheDir, "cuteproject", "https://bans.cute-project.net/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime, nil)
	if errScraper != nil {
		return nil, errScraper
	}

	scraper.sleepTime = siteSleepTime

	return scraper, nil
}

func newPhoenixSourceScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "phoenixsource", "https://phoenix-source.ru/sb/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime, nil)
}

func newSlavonServerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "slavonserver", "http://slavonserver.ru/ma/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime, nil)
}

func newGetSomeScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "getsome", "https://bans.getsome.co.nz/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newRushyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "rushy", "https://sourcebans.rushyservers.com/", "",
		parseDefault, nextURLLast, parseRushyTime, nil)
}

func newMoevsMachineScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "moevsmachine", "https://moevsmachine.tf/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newPRWHScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "prwh", "https://sourcebans.prwh.de/", "",
		parseDefault, nextURLLast, parsePRWHTime, nil)
}

func newVortexScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "vortex", "http://vortex.oyunboss.net/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newCasualFunScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "casualfun", "https://tf2-casual-fun.de/sourcebans/", "",
		parseDefault, nextURLLast, parsePRWHTime, nil)
}

func newRandomTF2Scraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "randomtf2", "https://bans.randomtf2.com/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newPlayesROScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "playesro", "https://www.playes.ro/csgobans/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newEOTLGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "eotlgaming", "https://tf2.endofthelinegaming.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime, nil)
}

func newBioCraftingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "biocrafting", "https://sourcebans.biocrafting.net/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newBigBangGamersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bigbanggamers", "http://208.71.172.9/", "",
		parseDefault, nextURLLast, parseSkialTime, nil)
}

func newEpicZoneScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "epiczone", "https://sourcebans.epiczone.sk/", "",
		parseStar, nextURLLast, parseGunServer, nil)
}

func newZubatScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "zubat", "https://sb.zubat.ru/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}

func newLunarioScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lunario", "https://sb.lunario.ro/", "",
		parseStar, nextURLLast, parseSkialTime, nil)
}
