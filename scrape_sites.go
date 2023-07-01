package main

import (
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

func newSkialScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "skial", "https://www.skial.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLFirst, parseSkialTime,
	)
}

func newGFLScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gfl", "https://sourcebans.gflclan.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpaceShipScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "spaceship", "https://sappho.io/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newUGCScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "ugc", "https://sb.ugc-gaming.net/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newSirPleaseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sirplease", "https://sirplease.gg/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newVidyaGaemsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "vidyagaems", "https://www.vidyagaems.net/sourcebans/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseTrailYear)
}

func newOwlTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "owl", "https://kingpandagamer.xyz/sb/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newZMBrasilScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "zmbrasil", "http://bans.zmbrasil.com.br/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newDixiGameScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "dixigame", "https://dixigame.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newScrapTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "scraptf", "https://bans.scrap.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newWonderlandTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "wonderland", "https://bans.wonderland.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseWonderlandTime)
}

// Uses google cache since cloudflare will restrict access.
func newWonderlandTFGOOGScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 10

	scraper, errScraper := newScraper(logger, cacheDir,
		"wonderland_goog",
		"https://webcache.googleusercontent.com/search?q=cache:https://bans.wonderland.tf/",
		"index.php?p=banlist",
		parseDefault,
		nextURLLast,
		parseWonderlandTime)
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
	return newScraper(logger, cacheDir, "lazypurple", "https://www.lazypurple.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newFirePoweredScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "firepowered", "https://firepoweredgaming.com/sourcebanspp/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newHarpoonScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "harpoongaming", "https://bans.harpoongaming.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPandaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "panda", "https://bans.panda-community.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newNeonHeightsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "neonheights", "https://neonheights.xyz/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPancakesScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "pancakes", "https://pancakes.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePancakesTime)
}

func newLOOSScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "loos", "https://looscommunity.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPubsTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "pubstf", "https://bans.pubs.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newServiliveClScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "servilivecl", "https://sourcebans.servilive.cl/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTimeMonthFirst)
}

func newCutiePieScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "cutiepie", "https://bans.cutiepie.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSGGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sggaming", "https://sg-gaming.net/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSGGamingTime)
}

func newApeModeScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "apemode", "https://sourcebans.apemode.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newMaxDBScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "maxdb", "https://bans.maxdb.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSvdosBrothersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "svdosbrothers", "https://bans.svdosbrothers.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseSVDos)
}

func newElectricScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "electric", "http://168.181.184.179/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGlobalParadiseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "globalparadise", "https://bans.theglobalparadise.org/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSavageServidoresScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "savageservidores", "https://bans.savageservidores.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newCSIServersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "csiservers", "https://bans.csiservers.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newLBGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lbgaming", "https://bans.lbgaming.co/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFluxTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "fluxtf", "https://bans.flux.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseFluxTime)
}

func newDarkPyroScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "darkpyro", "https://bans.darkpyrogaming.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDarkPyroTime)
}

func newOpstOnlineScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "opstonline", "https://www.opstonline.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBouncyBallScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bouncyball", "https://www.bouncyball.eu/bans2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFurryPoundScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "furrypound", "http://sourcebans.thefurrypound.org/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseFurryPoundTime)
}

func newRetroServersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "retroservers", "https://bans.retroservers.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSwapShopScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "swapshop", "http://tf2swapshop.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newECJScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "ecj", "https://ecj.tf/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newJumpAcademyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "jumpacademy", "https://bans.jumpacademy.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newTF2ROScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	// Not enough values to page yet...
	return newScraper(logger, cacheDir, "tf2ro", "https://bans.tf2ro.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSameTeemScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sameteem", "https://sameteem.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPowerFPSScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "powerfps", "https://bans.powerfps.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func new7MauScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "7mau", "https://7-mau.com/server/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGhostCapScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "ghostcap", "https://sourcebans.ghostcap.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpectreScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "spectre", "https://spectre.gg/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newDreamFireScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "dreamfire", "https://sourcebans.dreamfire.fr/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSettiScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "setti", "https://pong.setti.info/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newGunServerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gunserver", "https://gunserver.ru/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLFirst, parseGunServer)
}

func newHellClanScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "hellclan", "https://hellclan.co.uk/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseHellClanTime)
}

func newSneaksScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "sneaks", "https://bans.snksrv.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSneakTime)
}

func newNideScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "nide", "https://bans.nide.gg/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newAstraManiaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "astramania", "https://astramania.ro/sban2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTrailYear)
}

func newTF2MapsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "tf2maps", "https://bans.tf2maps.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPetrolTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "petroltf", "https://petrol.tf/sb/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newVaticanCityScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "vaticancity", "https://www.the-vaticancity.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newLazyNeerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lazyneer", "https://www.lazyneer.com/SourceBans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialAltTime)
}

func newTheVilleScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "theville", "https://www.theville.org/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newOreonScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "oreon", "https://www.tf2-oreon.fr/sourceban/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTriggerHappyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "triggerhappy", "https://triggerhappygamers.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTriggerHappyTime)
}

func newDefuseRoScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "defusero", "https://bans.defusero.org/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

// FIXME overcome cloudflare?
func newTawernaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "tawerna", "https://sb.tawerna.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTitanScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "titan", "https://bans.titan.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTitanTime)
}

func newDiscFFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "discff", "http://disc-ff.site.nfoservers.com/sourcebanstf2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

// TODO Has unique theme...
// func NewOtakuScraper( ) (*sbScraper, error){
//	return newScraper(logger, cacheDir, "otaku", "https://bans.otaku.tf/bans", "",
//		parseDefault, nextURLLast, parseOtakuTime)
//}

func newAMSGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "amsgaming", "https://bans.amsgaming.in/", "index.php?p=banlist",
		parseStar, nextURLLast, parseAMSGamingTime)
}

func newBaitedCommunityScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "baitedcommunity", "https://bans.baitedcommunity.com/", "index.php?p=banlist",
		parseStar, nextURLLast, parseBaitedTime)
}

func newCedaPugScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "cedapug", "https://cedapug.com/sourcebans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newGameSitesScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gamesites", "https://banlist.gamesites.cz/tf2/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newBachuruServasScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bachuruservas", "https://bachuruservas.lt/sb/", "index.php?p=banlist",
		parseStar, nextURLLast, parseBachuruServasTime)
}

func newBierwieseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bierwiese", "http://94.249.194.218/sb/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newAceKillScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "acekill", "https://sourcebans.acekill.pl/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newMagyarhnsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "magyarhns", "https://magyarhns.hu/sourcebans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newGamesTownScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "gamestown", "https://banlist.games-town.eu/", "index.php?p=banlist",
		parseStar, nextURLLast, parseTrailYear)
}

func newProGamesZetScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "progameszet", "https://bans.progameszet.ru/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newG44Scraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "g44", "http://bans.allmaps.g44.rocks/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newCuteProjectScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 4

	scraper, errScraper := newScraper(logger, cacheDir, "cuteproject", "https://bans.cute-project.net/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
	if errScraper != nil {
		return nil, errScraper
	}
	scraper.sleepTime = siteSleepTime

	return scraper, nil
}

func newPhoenixSourceScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "phoenixsource", "https://phoenix-source.ru/sb/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newSlavonServerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "slavonserver", "http://slavonserver.ru/ma/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newGetSomeScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "getsome", "https://bans.getsome.co.nz/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newRushyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "rushy", "https://sourcebans.rushyservers.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseRushyTime)
}

func newMoevsMachineScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "moevsmachine", "https://moevsmachine.tf/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPRWHScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "prwh", "https://sourcebans.prwh.de/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newVortexScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "vortex", "http://vortex.oyunboss.net/sourcebans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newCasualFunScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "casualfun", "https://tf2-casual-fun.de/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newRandomTF2Scraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "randomtf2", "https://bans.randomtf2.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPlayesROScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "playesro", "https://www.playes.ro/csgobans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newEOTLGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "eotlgaming", "https://tf2.endofthelinegaming.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newBioCraftingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "biocrafting", "https://sourcebans.biocrafting.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBigBangGamersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "bigbanggamers", "http://208.71.172.9/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newEpicZoneScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "epiczone", "https://sourcebans.epiczone.sk/", "index.php?p=banlist",
		parseStar, nextURLLast, parseGunServer)
}

func newZubatScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "zubat", "https://sb.zubat.ru/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newLunarioScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, "lunario", "https://sb.lunario.ro/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}
