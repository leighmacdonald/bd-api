package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"time"
)

func newSkialScraper() *sbScraper {
	return newScraper("skial", "https://www.skial.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLFirst, parseSkialTime,
	)
}

func newGFLScraper() *sbScraper {
	return newScraper("gfl", "https://sourcebans.gflclan.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpaceShipScraper() *sbScraper {
	return newScraper("spaceship", "https://sappho.io/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newUGCScraper() *sbScraper {
	return newScraper("ugc", "https://sb.ugc-gaming.net/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newSirPleaseScraper() *sbScraper {
	return newScraper("sirplease", "https://sirplease.gg/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newVidyaGaemsScraper() *sbScraper {
	return newScraper("vidyagaems", "https://www.vidyagaems.net/sourcebans/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseTrailYear)
}

func newOwlTFScraper() *sbScraper {
	return newScraper("owl", "https://kingpandagamer.xyz/sb/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newZMBrasilScraper() *sbScraper {
	return newScraper("zmbrasil", "http://bans.zmbrasil.com.br/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newDixiGameScraper() *sbScraper {
	return newScraper("dixigame", "https://dixigame.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newScrapTFScraper() *sbScraper {
	return newScraper("scraptf", "https://bans.scrap.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newWonderlandTFScraper() *sbScraper {
	return newScraper("wonderland", "https://bans.wonderland.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseWonderlandTime)
}

// Uses google cache since cloudflare will restrict access
func newWonderlandTFGOOGScraper() *sbScraper {
	s := newScraper("wonderland_goog", "https://webcache.googleusercontent.com/search?q=cache:https://bans.wonderland.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseWonderlandTime)
	s.sleepTime = time.Second * 10
	// Cached versions do not have a proper next link, so we have to generate one.
	s.nextURL = func(scraper *sbScraper, doc *goquery.Selection) string {
		s.curPage++
		return s.url(fmt.Sprintf("index.php?p=banlist&page=%d", s.curPage))
	}
	return s
}

func newLazyPurpleScraper() *sbScraper {
	return newScraper("lazypurple", "https://www.lazypurple.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newFirePoweredScraper() *sbScraper {
	return newScraper("firepowered", "https://firepoweredgaming.com/sourcebanspp/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newHarpoonScraper() *sbScraper {
	return newScraper("harpoongaming", "https://bans.harpoongaming.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPandaScraper() *sbScraper {
	return newScraper("panda", "https://bans.panda-community.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newNeonHeightsScraper() *sbScraper {
	return newScraper("neonheights", "https://neonheights.xyz/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPancakesScraper() *sbScraper {
	return newScraper("pancakes", "https://pancakes.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePancakesTime)
}

func newLOOSScraper() *sbScraper {
	return newScraper("loos", "https://looscommunity.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPubsTFScraper() *sbScraper {
	return newScraper("pubstf", "https://bans.pubs.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newServiliveClScraper() *sbScraper {
	return newScraper("servilivecl", "https://sourcebans.servilive.cl/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTimeMonthFirst)
}

func newCutiePieScraper() *sbScraper {
	return newScraper("cutiepie", "https://bans.cutiepie.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}
func newSGGamingScraper() *sbScraper {
	return newScraper("sggaming", "https://sg-gaming.net/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSGGamingTime)
}

func newApeModeScraper() *sbScraper {
	return newScraper("apemode", "https://sourcebans.apemode.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newMaxDBScraper() *sbScraper {
	return newScraper("maxdb", "https://bans.maxdb.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSvdosBrothersScraper() *sbScraper {
	return newScraper("svdosbrothers", "https://bans.svdosbrothers.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseSVDos)
}

func newElectricScraper() *sbScraper {
	return newScraper("electric", "http://168.181.184.179/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGlobalParadiseScraper() *sbScraper {
	return newScraper("globalparadise", "https://bans.theglobalparadise.org/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSavageServidoresScraper() *sbScraper {
	return newScraper("savageservidores", "https://bans.savageservidores.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newCSIServersScraper() *sbScraper {
	return newScraper("csiservers", "https://bans.csiservers.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newLBGamingScraper() *sbScraper {
	return newScraper("lbgaming", "https://bans.lbgaming.co/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFluxTFScraper() *sbScraper {
	return newScraper("fluxtf", "https://bans.flux.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseFluxTime)
}

func newDarkPyroScraper() *sbScraper {
	return newScraper("darkpyro", "https://bans.darkpyrogaming.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDarkPyroTime)
}

func newOpstOnlineScraper() *sbScraper {
	return newScraper("opstonline", "https://www.opstonline.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBouncyBallScraper() *sbScraper {
	return newScraper("bouncyball", "https://www.bouncyball.eu/bans2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFurryPoundScraper() *sbScraper {
	return newScraper("furrypound", "http://sourcebans.thefurrypound.org/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseFurryPoundTime)
}

func newRetroServersScraper() *sbScraper {
	return newScraper("retroservers", "https://bans.retroservers.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSwapShopScraper() *sbScraper {
	return newScraper("swapshop", "http://tf2swapshop.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newECJScraper() *sbScraper {
	return newScraper("ecj", "https://ecj.tf/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newJumpAcademyScraper() *sbScraper {
	return newScraper("jumpacademy", "https://bans.jumpacademy.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newTF2ROScraper() *sbScraper {
	// Not enough values to page yet...
	return newScraper("tf2ro", "https://bans.tf2ro.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSameTeemScraper() *sbScraper {
	return newScraper("sameteem", "https://sameteem.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPowerFPSScraper() *sbScraper {
	return newScraper("powerfps", "https://bans.powerfps.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func new7MauScraper() *sbScraper {
	return newScraper("7mau", "https://7-mau.com/server/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGhostCapScraper() *sbScraper {
	return newScraper("ghostcap", "https://sourcebans.ghostcap.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpectreScraper() *sbScraper {
	return newScraper("spectre", "https://spectre.gg/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newDreamFireScraper() *sbScraper {
	return newScraper("dreamfire", "https://sourcebans.dreamfire.fr/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSettiScraper() *sbScraper {
	return newScraper("setti", "https://pong.setti.info/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newGunServerScraper() *sbScraper {
	return newScraper("gunserver", "https://gunserver.ru/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLFirst, parseGunServer)
}

func newHellClanScraper() *sbScraper {
	return newScraper("hellclan", "https://hellclan.co.uk/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseHellClanTime)
}

func newSneaksScraper() *sbScraper {
	return newScraper("sneaks", "https://bans.snksrv.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSneakTime)
}

func newNideScraper() *sbScraper {
	return newScraper("nide", "https://bans.nide.gg/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newAstraManiaScraper() *sbScraper {
	return newScraper("astramania", "https://astramania.ro/sban2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTrailYear)
}

func newTF2MapsScraper() *sbScraper {
	return newScraper("tf2maps", "https://bans.tf2maps.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPetrolTFScraper() *sbScraper {
	return newScraper("petroltf", "https://petrol.tf/sb/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newVaticanCityScraper() *sbScraper {
	return newScraper("vaticancity", "https://www.the-vaticancity.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newLazyNeerScraper() *sbScraper {
	return newScraper("lazyneer", "https://www.lazyneer.com/SourceBans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialAltTime)
}

func newTheVilleScraper() *sbScraper {
	return newScraper("theville", "https://www.theville.org/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newOreonScraper() *sbScraper {
	return newScraper("oreon", "https://www.tf2-oreon.fr/sourceban/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTriggerHappyScraper() *sbScraper {
	return newScraper("triggerhappy", "https://triggerhappygamers.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTriggerHappyTime)
}

func newDefuseRoScraper() *sbScraper {
	return newScraper("defusero", "https://bans.defusero.org/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

// FIXME overcome cloudflare?
func newTawernaScraper() *sbScraper {
	return newScraper("tawerna", "https://sb.tawerna.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTitanScraper() *sbScraper {
	return newScraper("titan", "https://bans.titan.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTitanTime)
}

func newDiscFFScraper() *sbScraper {
	return newScraper("discff", "http://disc-ff.site.nfoservers.com/sourcebanstf2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

// TODO Has unique theme...
//func NewOtakuScraper() *sbScraper {
//	return newScraper("otaku", "https://bans.otaku.tf/bans", "",
//		parseDefault, nextURLLast, parseOtakuTime)
//}

func newAMSGamingScraper() *sbScraper {
	return newScraper("amsgaming", "https://bans.amsgaming.in/", "index.php?p=banlist",
		parseStar, nextURLLast, parseAMSGamingTime)
}

func newBaitedCommunityScraper() *sbScraper {
	return newScraper("baitedcommunity", "https://bans.baitedcommunity.com/", "index.php?p=banlist",
		parseStar, nextURLLast, parseBaitedTime)
}

func newCedaPugScraper() *sbScraper {
	return newScraper("cedapug", "https://cedapug.com/sourcebans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newGameSitesScraper() *sbScraper {
	return newScraper("gamesites", "https://banlist.gamesites.cz/tf2/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newBachuruServasScraper() *sbScraper {
	return newScraper("bachuruservas", "https://bachuruservas.lt/sb/", "index.php?p=banlist",
		parseStar, nextURLLast, parseBachuruServasTime)
}

func newBierwieseScraper() *sbScraper {
	return newScraper("bierwiese", "http://94.249.194.218/sb/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newAceKillScraper() *sbScraper {
	return newScraper("acekill", "https://sourcebans.acekill.pl/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newMagyarhnsScraper() *sbScraper {
	return newScraper("magyarhns", "https://magyarhns.hu/sourcebans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newGamesTownScraper() *sbScraper {
	return newScraper("gamestown", "https://banlist.games-town.eu/", "index.php?p=banlist",
		parseStar, nextURLLast, parseTrailYear)
}

func newProGamesZetScraper() *sbScraper {
	return newScraper("progameszet", "https://bans.progameszet.ru/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newG44Scraper() *sbScraper {
	return newScraper("g44", "http://bans.allmaps.g44.rocks/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newCuteProjectScraper() *sbScraper {
	s := newScraper("cuteproject", "https://bans.cute-project.net/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
	s.sleepTime = time.Second * 4
	return s
}

func newPhoenixSourceScraper() *sbScraper {
	return newScraper("phoenixsource", "https://phoenix-source.ru/sb/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newSlavonServerScraper() *sbScraper {
	return newScraper("slavonserver", "http://slavonserver.ru/ma/", "index.php?p=banlist",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newGetSomeScraper() *sbScraper {
	return newScraper("getsome", "https://bans.getsome.co.nz/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newRushyScraper() *sbScraper {
	return newScraper("rushy", "https://sourcebans.rushyservers.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseRushyTime)
}

func newMoevsMachineScraper() *sbScraper {
	return newScraper("moevsmachine", "https://moevsmachine.tf/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPRWHScraper() *sbScraper {
	return newScraper("prwh", "https://sourcebans.prwh.de/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newVortexScraper() *sbScraper {
	return newScraper("vortex", "http://vortex.oyunboss.net/sourcebans/", "index.php?p=banlist",
		parseStar, nextURLLast, parseSkialTime)
}

func newCasualFunScraper() *sbScraper {
	return newScraper("casualfun", "https://tf2-casual-fun.de/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newRandomTF2Scraper() *sbScraper {
	return newScraper("randomtf2", "https://bans.randomtf2.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}
