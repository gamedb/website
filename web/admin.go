package web

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/99designs/basicauth-go"
	"github.com/Jleagle/steam-go/steam"
	"github.com/gamedb/website/config"
	"github.com/gamedb/website/db"
	"github.com/gamedb/website/helpers"
	"github.com/gamedb/website/log"
	"github.com/gamedb/website/queue"
	"github.com/gamedb/website/websockets"
	"github.com/go-chi/chi"
)

func adminRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(basicauth.New("Steam", map[string][]string{
		config.Config.AdminUsername: {config.Config.AdminPassword},
	}))
	r.Get("/", adminHandler)
	r.Get("/{option}", adminHandler)
	r.Post("/{option}", adminHandler)
	return r
}

func adminHandler(w http.ResponseWriter, r *http.Request) {

	option := chi.URLParam(r, "option")

	switch option {
	case "refresh-all-apps":
		go adminQueueEveryApp()
	case "refresh-all-packages":
		go adminQueueEveryPackage()
	case "refresh-genres":
		go CronGenres()
	case "refresh-tags":
		go CronTags()
	case "refresh-developers":
		go CronDevelopers()
	case "refresh-publishers":
		go CronPublishers()
	case "refresh-donations":
		go CronDonations()
	case "refresh-ranks":
		go CronRanks()
	case "wipe-memcache":
		go adminMemcache()
	case "delete-bin-logs":
		go adminDeleteBinLogs(r)
	case "disable-consumers":
		go adminDisableConsumers()
	case "run-dev-code":
		go adminDev()
	case "queues":
		err := r.ParseForm()
		log.Err(err, r)
		go adminQueues(r)
	}

	// Redirect away after action
	if option != "" {
		http.Redirect(w, r, "/admin?"+option, 302)
		return
	}

	// Get configs for times
	configs, err := db.GetConfigs([]string{
		db.ConfTagsUpdated,
		db.ConfGenresUpdated,
		db.ConfGenresUpdated,
		db.ConfDonationsUpdated,
		db.ConfRanksUpdated,
		db.ConfAddedAllApps,
		db.ConfDevelopersUpdated,
		db.ConfPublishersUpdated,
		db.ConfWipeMemcache + "-" + config.Config.Environment.Get(),
		db.ConfRunDevCode,
		db.ConfGarbageCollection,
		db.ConfFixBrokenPlayers,
	})
	log.Err(err, r)

	// Template
	t := adminTemplate{}
	t.fill(w, r, "Admin", "")
	t.Configs = configs
	t.Goroutines = runtime.NumGoroutine()

	//
	gorm, err := db.GetMySQLClient()
	if err != nil {
		returnErrorTemplate(w, r, errorTemplate{Code: 500, Message: "Can't connect to mysql", Error: err})
		return
	}

	gorm.Raw("show binary logs").Scan(&t.BinLogs)

	var total uint64
	for k, v := range t.BinLogs {
		total = total + v.Bytes
		t.BinLogs[k].Total = total
	}

	gorm.Raw("SELECT * FROM information_schema.processlist where command != 'sleep'").Scan(&t.Queries)

	err = returnTemplate(w, r, "admin", t)
	log.Err(err, r)
}

type adminTemplate struct {
	GlobalTemplate
	Errors     []string
	Configs    map[string]db.Config
	Goroutines int
	Queries    []adminQuery
	BinLogs    []adminBinLog
}

type adminQuery struct {
	ID       int    `gorm:"column:ID"`
	User     string `gorm:"column:USER"`
	Host     string `gorm:"column:HOST"`
	Database string `gorm:"column:DB"`
	Command  string `gorm:"column:COMMAND"`
	Seconds  int64  `gorm:"column:TIME"`
	State    string `gorm:"column:STATE"`
	Info     string `gorm:"column:INFO"`
}

type adminBinLog struct {
	Name      string `gorm:"column:Log_name"`
	Bytes     uint64 `gorm:"column:File_size"`
	Encrypted string `gorm:"column:Encrypted"`
	Total     uint64
}

func (at adminTemplate) GetMCConfigKey() string {
	return "wipe-memcache" + "-" + config.Config.Environment.Get()
}

func adminDisableConsumers() {

}

func adminQueueEveryApp() {

	var last = 0
	var keepGoing = true
	var apps steam.AppList
	var err error
	var count int

	for keepGoing {

		apps, _, err = helpers.GetSteam().GetAppList(1000, last, 0, "")
		if err != nil {
			log.Err(err)
			return
		}

		count = count + len(apps.Apps)

		for _, v := range apps.Apps {
			err = queue.ProduceApp(v.AppID)
			if err != nil {
				log.Err(err, strconv.Itoa(v.AppID))
				continue
			}
			last = v.AppID
		}

		keepGoing = apps.HaveMoreResults
	}

	log.Info("Found " + strconv.Itoa(count) + " apps")

	//
	err = db.SetConfig(db.ConfAddedAllApps, strconv.FormatInt(time.Now().Unix(), 10))
	log.Err(err)

	page, err := websockets.GetPage(websockets.PageAdmin)
	log.Err(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfAddedAllApps + " complete"})
	}

	log.Info(strconv.Itoa(len(apps.Apps)) + " apps added to rabbit")
}

func adminQueueEveryPackage() {

	apps, err := db.GetAppsWithColumnDepth("packages", 2, []string{"packages"})
	if err != nil {
		log.Err(err)
		return
	}

	packageIDs := map[int]bool{}
	for _, v := range apps {

		packages, err := v.GetPackages()
		if err != nil {
			log.Err(err)
			return
		}

		for _, vv := range packages {
			packageIDs[vv] = true
		}
	}

	for k := range packageIDs {

		err = queue.ProducePackage(k)
		if err != nil {
			log.Err(err)
			return
		}
	}

	//
	err = db.SetConfig(db.ConfAddedAllPackages, strconv.FormatInt(time.Now().Unix(), 10))
	log.Err(err)

	page, err := websockets.GetPage(websockets.PageAdmin)
	log.Err(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfAddedAllPackages + " complete"})
	}

	log.Info(strconv.Itoa(len(packageIDs)) + " packages added to rabbit")
}

func CronDonations() {

	donations, err := db.GetDonations(0, 0)
	if err != nil {
		cronLogErr(err)
		return
	}

	// map[player]total
	counts := make(map[int64]int)

	for _, v := range donations {

		if _, ok := counts[v.PlayerID]; ok {
			counts[v.PlayerID] = counts[v.PlayerID] + v.AmountUSD
		} else {
			counts[v.PlayerID] = v.AmountUSD
		}
	}

	for k, v := range counts {
		player, err := db.GetPlayer(k)
		if err != nil {
			cronLogErr(err)
			continue
		}

		player.Donated = v
		err = db.SaveKind(player.GetKey(), player)
		cronLogErr(err)
	}

	//
	err = db.SetConfig(db.ConfDonationsUpdated, strconv.FormatInt(time.Now().Unix(), 10))
	cronLogErr(err)

	page, err := websockets.GetPage(websockets.PageAdmin)
	log.Err(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfDonationsUpdated + " complete"})
	}

	cronLogInfo("Updated " + strconv.Itoa(len(counts)) + " player donation counts")
}

func adminQueues(r *http.Request) {

	if val := r.PostForm.Get("player-id"); val != "" {

		playerID, err := strconv.ParseInt(val, 10, 64)
		log.Err(err, r)
		if err == nil {

			err = queue.ProducePlayer(playerID)
			log.Err(err, r)
		}
	}

	if val := r.PostForm.Get("app-id"); val != "" {

		appID, err := strconv.Atoi(val)
		log.Err(err, r)
		if err == nil {

			err = queue.ProduceApp(appID)
			log.Err(err, r)
		}
	}

	if val := r.PostForm.Get("package-id"); val != "" {

		packageID, err := strconv.Atoi(val)
		log.Err(err, r)
		if err == nil {

			err = queue.ProducePackage(packageID)
			log.Err(err, r)
		}
	}

	if val := r.PostForm.Get("bundle-id"); val != "" {

		bundleID, err := strconv.Atoi(val)
		log.Err(err, r)
		if err == nil {

			err = queue.ProduceBundle(bundleID, 0)
			log.Err(err, r)
		}
	}

	if val := r.PostForm.Get("apps-ts"); val != "" {

		log.Info("Queueing apps")

		ts, err := strconv.ParseInt(val, 10, 64)
		log.Err(err, r)
		if err == nil {

			apps, _, err := helpers.GetSteam().GetAppList(100000, 0, ts, "")
			log.Err(err, r)
			if err == nil {

				log.Info("Found " + strconv.Itoa(len(apps.Apps)) + " apps")

				for _, v := range apps.Apps {
					err = queue.ProduceApp(v.AppID)
					log.Err(err, r)
				}
			}
		}
	}
}

func CronGenres() {

	cronLogInfo(log.ServiceLocal, "Genres updating")

	// Get current genres, to delete old ones
	currentGenres, err := db.GetAllGenres()
	if err != nil {
		cronLogErr(err)
		return
	}

	genresToDelete := map[int]bool{}
	for _, v := range currentGenres {
		genresToDelete[v.ID] = true
	}

	var genreNameMap = map[int]string{}
	for _, v := range currentGenres {
		genreNameMap[v.ID] = strings.TrimSpace(v.GetName())
	}

	// Get apps from mysql
	appsWithGenres, err := db.GetAppsWithColumnDepth("genres", 3, []string{"genres", "prices", "reviews_score"})
	cronLogErr(err)

	cronLogInfo("Found " + strconv.Itoa(len(appsWithGenres)) + " apps with genres")

	newGenres := make(map[int]*statsRow)
	for _, app := range appsWithGenres {

		appGenreIDs, err := app.GetGenreIDs()
		if err != nil {
			cronLogErr(err)
			continue
		}

		if len(appGenreIDs) == 0 {
			// appGenreIDs = []db.AppGenre{{ID: 0, Name: ""}}
		}

		// For each genre in an app
		for _, genreID := range appGenreIDs {

			delete(genresToDelete, genreID)

			var genreName string
			if val, ok := genreNameMap[genreID]; ok {
				genreName = val
			} else {
				// genreName = "Unknown"
				continue
			}

			if _, ok := newGenres[genreID]; ok {
				newGenres[genreID].count++
				newGenres[genreID].totalScore += app.ReviewsScore
			} else {
				newGenres[genreID] = &statsRow{
					name:       genreName,
					count:      1,
					totalScore: app.ReviewsScore,
					totalPrice: map[steam.CountryCode]int{},
				}
			}

			for code := range steam.Countries {
				price, err := app.GetPrice(code)
				if err != nil {
					// cronLogErr(err, r)
					continue
				}
				newGenres[genreID].totalPrice[code] += price.Final
			}
		}
	}

	var limit int
	var wg sync.WaitGroup

	// Delete old genres
	limit++
	wg.Add(1)
	go func() {

		defer func() {
			limit--
			wg.Done()
		}()

		var genresToDeleteSlice []int
		for genreID := range genresToDelete {
			genresToDeleteSlice = append(genresToDeleteSlice, genreID)
		}

		err := db.DeleteGenres(genresToDeleteSlice)
		cronLogErr(err)

	}()

	wg.Wait()

	gorm, err := db.GetMySQLClient()
	if err != nil {
		cronLogErr(err)
		return
	}

	// Update current genres
	var count = 1
	for k, v := range newGenres {

		if limit >= 2 {
			wg.Wait()
		}

		adminStatsLogger("genre", count, len(newGenres), v.name)

		limit++
		wg.Add(1)
		go func(genreID int, v *statsRow) {

			defer func() {
				limit--
				wg.Done()
			}()

			var genre db.Genre

			gorm = gorm.Unscoped().FirstOrInit(&genre, db.Genre{ID: genreID})
			cronLogErr(gorm.Error)

			genre.Name = v.name
			genre.Apps = v.count
			genre.MeanPrice = v.getMeanPrice()
			genre.MeanScore = v.getMeanScore()
			genre.DeletedAt = nil

			gorm = gorm.Unscoped().Save(&genre)
			cronLogErr(gorm.Error)

		}(k, v)

		count++
	}
	wg.Wait()

	//
	err = db.SetConfig(db.ConfGenresUpdated, strconv.FormatInt(time.Now().Unix(), 10))
	cronLogErr(err)

	//
	page, err := websockets.GetPage(websockets.PageAdmin)
	cronLogErr(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfGenresUpdated + " complete"})
	}

	//
	err = helpers.GetMemcache().Delete(helpers.MemcacheGenreKeyNames.Key)
	cronLogErr(err)

	//
	cronLogInfo("Genres updated")
}

func CronPublishers() {

	cronLogInfo(log.ServiceLocal, "Publishers updating")

	// Get current publishers, to delete old ones
	allPublishers, err := db.GetAllPublishers()
	if err != nil {
		cronLogErr(err)
		return
	}

	publishersToDelete := map[int]bool{}
	for _, publisherRow := range allPublishers {
		publishersToDelete[publisherRow.ID] = true
	}

	var publisherNameMap = map[int]string{}
	for _, v := range allPublishers {
		publisherNameMap[v.ID] = strings.TrimSpace(v.GetName())
	}

	// Get apps from mysql
	appsWithPublishers, err := db.GetAppsWithColumnDepth("publishers", 2, []string{"publishers", "prices", "reviews_score"})
	cronLogErr(err)

	cronLogInfo("Found " + strconv.Itoa(len(appsWithPublishers)) + " apps with publishers")

	newPublishers := make(map[int]*statsRow)
	for _, app := range appsWithPublishers {

		appPublishers, err := app.GetPublisherIDs()
		if err != nil {
			cronLogErr(err)
			continue
		}

		if len(appPublishers) == 0 {
			// appPublishers = []string{""}
		}

		// For each publisher in an app
		for _, appPublisherID := range appPublishers {

			delete(publishersToDelete, appPublisherID)

			var publisherName string
			if val, ok := publisherNameMap[appPublisherID]; ok {
				publisherName = val
			} else {
				// publisherName = "Unknown"
				continue
			}

			if _, ok := newPublishers[appPublisherID]; ok {
				newPublishers[appPublisherID].count++
				newPublishers[appPublisherID].totalScore += app.ReviewsScore
			} else {
				newPublishers[appPublisherID] = &statsRow{
					name:       publisherName,
					count:      1,
					totalPrice: map[steam.CountryCode]int{},
					totalScore: app.ReviewsScore,
				}
			}

			for code := range steam.Countries {
				price, err := app.GetPrice(code)
				if err != nil {
					continue
				}
				newPublishers[appPublisherID].totalPrice[code] += price.Final
			}
		}
	}

	var limit int
	var wg sync.WaitGroup

	// Delete old publishers
	limit++
	wg.Add(1)
	go func() {

		defer func() {
			limit--
			wg.Done()
		}()

		var pubsToDeleteSlice []int
		for publisherID := range publishersToDelete {
			pubsToDeleteSlice = append(pubsToDeleteSlice, publisherID)
		}

		err := db.DeletePublishers(pubsToDeleteSlice)
		cronLogErr(err)

	}()

	wg.Wait()

	gorm, err := db.GetMySQLClient()
	if err != nil {
		cronLogErr(err)
		return
	}

	// Update current publishers
	var count = 1
	for k, v := range newPublishers {

		if limit >= 2 {
			wg.Wait()
		}

		adminStatsLogger("publisher", count, len(newPublishers), v.name)

		limit++
		wg.Add(1)
		go func(publisherID int, v *statsRow) {

			defer func() {
				limit--
				wg.Done()
			}()

			var publisher db.Publisher

			gorm = gorm.Unscoped().FirstOrInit(&publisher, db.Publisher{ID: publisherID})
			cronLogErr(gorm.Error)

			publisher.Name = v.name
			publisher.Apps = v.count
			publisher.MeanPrice = v.getMeanPrice()
			publisher.MeanScore = v.getMeanScore()
			publisher.DeletedAt = nil

			gorm = gorm.Unscoped().Save(&publisher)
			cronLogErr(gorm.Error)

		}(k, v)

		count++
	}

	wg.Wait()

	//
	err = db.SetConfig(db.ConfPublishersUpdated, strconv.FormatInt(time.Now().Unix(), 10))
	cronLogErr(err)

	//
	page, err := websockets.GetPage(websockets.PageAdmin)
	cronLogErr(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfPublishersUpdated + " complete"})
	}

	//
	err = helpers.GetMemcache().Delete(helpers.MemcachePublisherKeyNames.Key)
	cronLogErr(err)

	//
	cronLogInfo("Publishers updated")
}

func CronDevelopers() {

	cronLogInfo(log.ServiceLocal, "Developers updating")

	// Get current developers, to delete old ones
	allDevelopers, err := db.GetAllDevelopers([]string{"id", "name"})
	if err != nil {
		cronLogErr(err)
		return
	}

	developersToDelete := map[int]bool{}
	for _, v := range allDevelopers {
		developersToDelete[v.ID] = true
	}

	var developersNameMap = map[int]string{}
	for _, v := range allDevelopers {
		developersNameMap[v.ID] = strings.TrimSpace(v.GetName())
	}

	// Get apps from mysql
	appsWithDevelopers, err := db.GetAppsWithColumnDepth("developers", 2, []string{"developers", "prices", "reviews_score"})
	cronLogErr(err)

	cronLogInfo("Found " + strconv.Itoa(len(appsWithDevelopers)) + " apps with developers")

	newDevelopers := make(map[int]*statsRow)
	for _, app := range appsWithDevelopers {

		appDevelopers, err := app.GetDeveloperIDs()
		if err != nil {
			cronLogErr(err)
			continue
		}

		if len(appDevelopers) == 0 {
			// appDevelopers = []string{""}
		}

		// For each developer in an app
		for _, appDeveloperID := range appDevelopers {

			delete(developersToDelete, appDeveloperID)

			var developersName string
			if val, ok := developersNameMap[appDeveloperID]; ok {
				developersName = val
			} else {
				continue
			}

			if _, ok := newDevelopers[appDeveloperID]; ok {
				newDevelopers[appDeveloperID].count++
				newDevelopers[appDeveloperID].totalScore += app.ReviewsScore
			} else {
				newDevelopers[appDeveloperID] = &statsRow{
					name:       developersName,
					count:      1,
					totalPrice: map[steam.CountryCode]int{},
					totalScore: app.ReviewsScore,
				}
			}

			for code := range steam.Countries {
				price, err := app.GetPrice(code)
				if err != nil {
					// cronLogErr(err, r)
					continue
				}
				newDevelopers[appDeveloperID].totalPrice[code] += price.Final
			}
		}
	}

	var limit int
	var wg sync.WaitGroup

	// Delete old developers
	limit++
	wg.Add(1)
	go func() {

		defer func() {
			limit--
			wg.Done()
		}()

		var devsToDeleteSlice []int
		for k := range developersToDelete {
			devsToDeleteSlice = append(devsToDeleteSlice, k)
		}

		err := db.DeleteDevelopers(devsToDeleteSlice)
		cronLogErr(err)

	}()

	wg.Wait()

	gorm, err := db.GetMySQLClient()
	if err != nil {
		cronLogErr(err)
		return
	}

	// Update current developers
	var count = 1
	for k, v := range newDevelopers {

		if limit >= 2 {
			wg.Wait()
		}

		adminStatsLogger("developer", count, len(newDevelopers), v.name)

		limit++
		wg.Add(1)
		go func(developerInt int, v *statsRow) {

			defer func() {
				limit--
				wg.Done()
			}()

			var developer db.Developer

			gorm = gorm.Unscoped().FirstOrInit(&developer, db.Developer{ID: developerInt})
			cronLogErr(gorm.Error)

			developer.Name = v.name
			developer.Apps = v.count
			developer.MeanPrice = v.getMeanPrice()
			developer.MeanScore = v.getMeanScore()
			developer.DeletedAt = nil

			gorm = gorm.Unscoped().Save(&developer)
			cronLogErr(gorm.Error)

		}(k, v)

		count++
	}
	wg.Wait()

	//
	err = db.SetConfig(db.ConfDevelopersUpdated, strconv.FormatInt(time.Now().Unix(), 10))
	cronLogErr(err)

	//
	page, err := websockets.GetPage(websockets.PageAdmin)
	cronLogErr(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfDevelopersUpdated + " complete"})
	}

	//
	err = helpers.GetMemcache().Delete(helpers.MemcacheDeveloperKeyNames.Key)
	cronLogErr(err)

	//
	cronLogInfo("Developers updated")
}

func CronTags() {

	// Get current tags, to delete old ones
	tags, err := db.GetAllTags()
	if err != nil {
		cronLogErr(err)
		return
	}

	tagsToDelete := map[int]int{}
	for _, tag := range tags {
		tagsToDelete[tag.ID] = tag.ID
	}

	// Get tags from Steam
	tagsResp, _, err := helpers.GetSteam().GetTags()
	if err != nil {
		cronLogErr(err)
		return
	}

	steamTagMap := tagsResp.GetMap()

	appsWithTags, err := db.GetAppsWithColumnDepth("tags", 2, []string{"tags", "prices", "reviews_score"})
	cronLogErr(err)

	cronLogInfo("Found " + strconv.Itoa(len(appsWithTags)) + " apps with tags")

	newTags := make(map[int]*statsRow)
	for _, app := range appsWithTags {

		appTags, err := app.GetTagIDs()
		if err != nil {
			cronLogErr(err)
			continue
		}

		if len(appTags) == 0 {
			// appTags = []int{}
		}

		// For each tag in an app
		for _, tagID := range appTags {

			delete(tagsToDelete, tagID)

			if _, ok := newTags[tagID]; ok {
				newTags[tagID].count++
				newTags[tagID].totalScore += app.ReviewsScore
			} else {
				newTags[tagID] = &statsRow{
					name:       strings.TrimSpace(steamTagMap[tagID]),
					count:      1,
					totalPrice: map[steam.CountryCode]int{},
					totalScore: app.ReviewsScore,
				}
			}

			for code := range steam.Countries {
				price, err := app.GetPrice(code)
				if err != nil {
					// cronLogErr(err, r)
					continue
				}
				newTags[tagID].totalPrice[code] += price.Final
			}
		}
	}

	var limit int
	var wg sync.WaitGroup

	// Delete old tags
	limit++
	wg.Add(1)
	go func() {

		defer func() {
			limit--
			wg.Done()
		}()

		var tagsToDeleteSlice []int
		for _, v := range tagsToDelete {
			tagsToDeleteSlice = append(tagsToDeleteSlice, v)
		}

		err := db.DeleteTags(tagsToDeleteSlice)
		cronLogErr(err)

	}()

	wg.Wait()

	gorm, err := db.GetMySQLClient()
	if err != nil {
		cronLogErr(err)
		return
	}

	// Update current tags
	var count = 1
	for k, v := range newTags {

		if limit >= 2 {
			wg.Wait()
		}

		adminStatsLogger("tag", count, len(newTags), v.name)

		limit++
		wg.Add(1)
		go func(tagID int, v *statsRow) {

			defer func() {
				limit--
				wg.Done()
			}()

			var tag db.Tag

			gorm = gorm.Unscoped().FirstOrInit(&tag, db.Tag{ID: tagID})
			cronLogErr(gorm.Error)

			tag.Name = v.name
			tag.Apps = v.count
			tag.MeanPrice = v.getMeanPrice()
			tag.MeanScore = v.getMeanScore()
			tag.DeletedAt = nil

			gorm = gorm.Unscoped().Save(&tag)
			cronLogErr(gorm.Error)

		}(k, v)

		count++
	}
	wg.Wait()

	//
	err = db.SetConfig(db.ConfTagsUpdated, strconv.FormatInt(time.Now().Unix(), 10))
	cronLogErr(err)

	//
	page, err := websockets.GetPage(websockets.PageAdmin)
	cronLogErr(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfTagsUpdated + " complete"})
	}

	//
	err = helpers.GetMemcache().Delete(helpers.MemcacheTagKeyNames.Key)
	cronLogErr(err)

	//
	cronLogInfo("Tags updated")
}

func adminStatsLogger(tableName string, count int, total int, rowName string) {

	log.Info("Updating " + tableName + " - " + strconv.Itoa(count) + " / " + strconv.Itoa(total) + ": " + rowName)
}

func CronRanks() {

	cronLogInfo("Ranks updated started")

	timeStart := time.Now().Unix()

	oldKeys, err := db.GetRankKeys()
	if err != nil {
		cronLogErr(err)
		return
	}

	newRanks := make(map[int64]*db.PlayerRank)
	var players []db.Player

	var wg sync.WaitGroup

	for _, v := range []string{"-level", "-games_count", "-badges_count", "-play_time", "-friends_count"} {

		wg.Add(1)
		go func(column string) {

			defer wg.Done()

			players, _, err = db.GetAllPlayers(column, db.PlayersToRank, false)
			if err != nil {
				cronLogErr(err)
				return
			}

			for _, player := range players {
				newRanks[player.PlayerID] = db.NewRankFromPlayer(player)
				delete(oldKeys, player.PlayerID)
			}

		}(v)

	}
	wg.Wait()

	// Convert new ranks to slice
	var ranks []*db.PlayerRank
	for _, v := range newRanks {
		ranks = append(ranks, v)
	}

	// Make ranks
	var prev int
	var rank = 0

	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Level > ranks[j].Level
	})
	for _, v := range ranks {
		if v.Level != prev {
			rank++
		}
		v.UpdatedAt = time.Now()
		v.LevelRank = rank
		prev = v.Level
	}

	rank = 0
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Games > ranks[j].Games
	})
	for _, v := range ranks {
		if v.Games != prev {
			rank++
		}
		v.UpdatedAt = time.Now()
		v.GamesRank = rank
		prev = v.Games
	}

	rank = 0
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Badges > ranks[j].Badges
	})
	for _, v := range ranks {
		if v.Badges != prev {
			rank++
		}
		v.UpdatedAt = time.Now()
		v.BadgesRank = rank
		prev = v.Badges
	}

	rank = 0
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].PlayTime > ranks[j].PlayTime
	})
	for _, v := range ranks {
		if v.PlayTime != prev {
			rank++
		}
		v.UpdatedAt = time.Now()
		v.PlayTimeRank = rank
		prev = v.PlayTime
	}

	rank = 0
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Friends > ranks[j].Friends
	})
	for _, v := range ranks {
		if v.Friends != prev {
			rank++
		}
		v.UpdatedAt = time.Now()
		v.FriendsRank = rank
		prev = v.Friends
	}

	// Make kinds
	var kinds []db.Kind
	for _, v := range ranks {
		kinds = append(kinds, *v)
	}

	// Update ranks
	err = db.BulkSaveKinds(kinds, db.KindPlayerRank, false)
	if err != nil {
		cronLogErr(err)
		return
	}

	// Remove old ranks
	var keysToDelete []*datastore.Key
	for _, v := range oldKeys {
		keysToDelete = append(keysToDelete, v)
	}

	err = db.BulkDeleteKinds(keysToDelete, false)
	if err != nil {
		cronLogErr(err)
		return
	}

	// Update config
	err = db.SetConfig(db.ConfRanksUpdated, strconv.FormatInt(time.Now().Unix(), 10))
	cronLogErr(err)

	page, err := websockets.GetPage(websockets.PageAdmin)

	if err == nil {
		page.Send(adminWebsocket{db.ConfRanksUpdated + " complete"})
	}

	//
	err = helpers.GetMemcache().Delete(helpers.MemcacheRanksCount.Key)
	cronLogErr(err)

	//
	cronLogInfo("Ranks updated in " + strconv.FormatInt(time.Now().Unix()-timeStart, 10) + " seconds")
}

func adminMemcache() {

	err := helpers.GetMemcache().DeleteAll()
	log.Err(err)

	err = db.SetConfig(db.ConfWipeMemcache+"-"+config.Config.Environment.Get(), strconv.FormatInt(time.Now().Unix(), 10))
	log.Err(err)

	page, err := websockets.GetPage(websockets.PageAdmin)
	log.Err(err)

	if err == nil {
		page.Send(adminWebsocket{db.ConfWipeMemcache + "-" + config.Config.Environment.Get() + " complete"})
	}

	log.Info("Memcache wiped")
}

func adminDeleteBinLogs(r *http.Request) {

	name := r.URL.Query().Get("name")
	if name != "" {

		gorm, err := db.GetMySQLClient(true)
		if err != nil {
			log.Err(err)
			return
		}

		gorm.Exec("PURGE BINARY LOGS TO '" + name + "'")
	}
}

func adminDev() {

	log.Info("Started dev code")

	// players, _, err := db.GetAllPlayers("player_id", 0, false)
	// if err != nil {
	// 	log.Err(err)
	// 	return
	// }
	//
	// chunks := db.ChunkPlayers(players)
	//
	// for k, chunk := range chunks {
	//
	// 	log.Info("Chunk " + strconv.Itoa(k))
	//
	// 	var keys []*datastore.Key
	// 	var docs []mongo.MongoDocument
	//
	// 	for _, vv := range chunk {
	//
	// 		keys = append(keys, vv.GetKey())
	//
	// 		docs = append(docs, mongo.Player{
	// 			ID: vv.PlayerID,
	// 		})
	// 	}
	//
	// 	_, err := mongo.InsertDocuments(mongo.CollectionPlayers, docs)
	// 	if err != nil {
	// 		log.Err(err)
	// 	} else {
	// 		err = db.BulkDeleteKinds(keys, true)
	// 		log.Err(err)
	// 	}
	// }

	err := db.SetConfig(db.ConfRunDevCode, strconv.FormatInt(time.Now().Unix(), 10))
	log.Err(err)

	page, err := websockets.GetPage(websockets.PageAdmin)
	log.Err(err)
	if err == nil {
		page.Send(adminWebsocket{db.ConfRunDevCode + " complete"})
	}

	log.Info("Dev code run")
}

type statsRow struct {
	name       string
	count      int
	totalPrice map[steam.CountryCode]int
	totalScore float64
}

func (t statsRow) getMeanPrice() string {

	means := map[steam.CountryCode]float64{}

	for code, total := range t.totalPrice {
		means[code] = float64(total) / float64(t.count)
	}

	bytes, err := json.Marshal(means)
	log.Err(err)

	return string(bytes)
}

func (t statsRow) getMeanScore() float64 {
	return t.totalScore / float64(t.count)
}

type adminWebsocket struct {
	Message string `json:"message"`
}

func cronLogErr(interfaces ...interface{}) {
	log.Err(append(interfaces, log.LogNameCron, log.LogNameGameDB)...)
}

func cronLogInfo(interfaces ...interface{}) {
	log.Info(append(interfaces, log.LogNameCron, log.LogNameGameDB)...)
}
