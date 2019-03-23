package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/datastore"
	"github.com/Jleagle/influxql"
	"github.com/dustin/go-humanize"
	"github.com/gamedb/website/db"
	"github.com/gamedb/website/helpers"
	"github.com/gamedb/website/log"
	"github.com/gamedb/website/mongo"
	"github.com/gamedb/website/queue"
	"github.com/gamedb/website/session"
	"github.com/go-chi/chi"
)

func playerHandler(w http.ResponseWriter, r *http.Request) {

	t := playerTemplate{}

	id := chi.URLParam(r, "id")
	var toasts []Toast

	idx, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		returnErrorTemplate(w, r, errorTemplate{Code: 404, Message: "Invalid Player ID: " + id, Error: err})
		return
	}

	if !db.IsValidPlayerID(idx) {
		returnErrorTemplate(w, r, errorTemplate{Code: 404, Message: "Invalid Player ID: " + id})
		return
	}

	// Find the player row
	player, err := db.GetPlayer(idx)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {

			err = queue.ProducePlayer(idx)
			log.Err(err, r)

			data := errorTemplate{Code: 404, Message: "We haven't scanned this player yet, but we are looking now.", DataID: idx}
			data.addToast(Toast{Title: "Update", Message: "Player has been queued for an update"})

			returnErrorTemplate(w, r, data)
		} else {
			returnErrorTemplate(w, r, errorTemplate{Code: 500, Message: "There was an issue retrieving the player.", Error: err})
		}
		return
	}

	// Queue profile for a refresh
	if player.ShouldUpdate(r.UserAgent(), db.PlayerUpdateAuto) {
		err = queue.ProducePlayer(player.PlayerID)
		log.Err(err, r)
		t.addToast(Toast{Title: "Update", Message: "Player has been queued for an update"})
	}

	var wg sync.WaitGroup

	// Get friends
	var friends []db.ProfileFriend
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		// var err error
		friends, _ = player.GetFriends()
		// if err != nil {
		// 	log.Err(err, r)
		// 	return
		// }

		// Queue friends to be scanned
		// if !player.ShouldUpdate(r.UserAgent(), db.PlayerUpdateFriends) {
		// 	return
		// }

		// for _, friend := range friends {
		// 	err = queue.QueuePlayer(friend.SteamID)
		// 	log.Err(err, r)
		// }
		//
		// player.FriendsAddedAt = time.Now()
		//
		// err = player.Save() // switch to update query so not to overwrite other player changes
		// log.Err(err, r)

	}(player)

	// Get ranks
	var ranks db.PlayerRank
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		var err error
		ranks, err = db.GetRank(player.PlayerID)
		if err != nil && err != datastore.ErrNoSuchEntity {
			log.Err(err, r)
		}

	}(player)

	// Number of players
	var players int64
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		var err error
		players, err = mongo.CountPlayers()
		log.Err(err, r)

	}(player)

	// Get badges
	var badges []db.ProfileBadge
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		// var err error
		badges, _ = player.GetBadges()
		// log.Err(err, r) // log.Err(err, r) // Too many logs

	}(player)

	// Get recent games
	var recentGames []RecentlyPlayedGame
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		response, err := player.GetRecentGames()
		if err != nil {

			log.Err(err, r)
			return
		}

		for _, v := range response {

			game := RecentlyPlayedGame{}
			game.AppID = v.AppID
			game.Name = v.Name
			game.Weeks = v.PlayTime2Weeks
			game.WeeksNice = helpers.GetTimeShort(v.PlayTime2Weeks, 2)
			game.AllTime = v.PlayTimeForever
			game.AllTimeNice = helpers.GetTimeShort(v.PlayTimeForever, 2)

			if v.ImgIconURL == "" {
				game.Icon = db.DefaultAppIcon
			} else {
				game.Icon = "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/" + strconv.Itoa(v.AppID) + "/" + v.ImgIconURL + ".jpg"
			}

			recentGames = append(recentGames, game)
		}

	}(player)

	// Get bans
	var bans db.PlayerBans
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		var err error
		bans, err = player.GetBans()
		log.Err(err, r)

		if err == nil {
			bans.EconomyBan = strings.Title(bans.EconomyBan)
		}

	}(player)

	// Get badge stats
	var badgeStats db.ProfileBadgeStats
	wg.Add(1)
	go func(player db.Player) {

		defer wg.Done()

		var err error
		badgeStats, err = player.GetBadgeStats()
		log.Err(err, r)

	}(player)

	// Wait
	wg.Wait()

	player.VanintyURL = helpers.TruncateString(player.VanintyURL, 14)

	// Game stats
	gameStats, err := player.GetGameStats(session.GetCountryCode(r))
	// log.Err(err, r) // Disable for now, too many logs

	// Make banners
	banners := make(map[string][]string)
	var primary []string

	if player.PlayerID == 76561197960287930 {
		primary = append(primary, "This profile belongs to Gabe Newell, the Co-founder of Valve")
	}

	if len(primary) > 0 {
		banners["primary"] = primary
	}

	t.Banners = banners

	// Template
	t.fill(w, r, player.PersonaName, "")
	t.addAssetHighCharts()
	t.Player = player
	t.Friends = friends
	t.Apps = []db.PlayerApp{}
	t.Ranks = playerRanksTemplate{ranks, players}
	t.Badges = badges
	t.BadgeStats = badgeStats
	t.RecentGames = recentGames
	t.GameStats = gameStats
	t.Bans = bans
	t.toasts = toasts

	err = returnTemplate(w, r, "player", t)
	log.Err(err, r)
}

type playerTemplate struct {
	GlobalTemplate
	Apps        []db.PlayerApp
	Badges      []db.ProfileBadge
	BadgeStats  db.ProfileBadgeStats
	Banners     map[string][]string
	Bans        db.PlayerBans
	Friends     []db.ProfileFriend
	GameStats   db.PlayerAppStatsTemplate
	Player      db.Player
	Ranks       playerRanksTemplate
	RecentGames []RecentlyPlayedGame
}

// RecentlyPlayedGame
type RecentlyPlayedGame struct {
	AllTime     int
	AllTimeNice string
	AppID       int
	Icon        string
	Name        string
	Weeks       int
	WeeksNice   string
}

// playerRanksTemplate
type playerRanksTemplate struct {
	Ranks   db.PlayerRank
	Players int64
}

func (p playerRanksTemplate) format(rank int) string {

	ord := humanize.Ordinal(rank)
	if ord == "0th" {
		return "-"
	}
	return ord
}

func (p playerRanksTemplate) GetLevel() string {
	return p.format(p.Ranks.LevelRank)
}

func (p playerRanksTemplate) GetGames() string {
	return p.format(p.Ranks.GamesRank)
}

func (p playerRanksTemplate) GetBadges() string {
	return p.format(p.Ranks.BadgesRank)
}

func (p playerRanksTemplate) GetTime() string {
	return p.format(p.Ranks.PlayTimeRank)
}

func (p playerRanksTemplate) GetFriends() string {
	return p.format(p.Ranks.FriendsRank)
}

func (p playerRanksTemplate) formatPercent(rank int) string {

	if rank == 0 {
		return ""
	}

	precision := 0
	if rank <= 10 {
		precision = 3
	} else if rank <= 100 {
		precision = 2
	} else if rank <= 1000 {
		precision = 1
	}

	percent := (float64(rank) / float64(p.Players)) * 100
	return helpers.FloatToString(percent, precision) + "%"

}

func (p playerRanksTemplate) GetLevelPercent() string {
	return p.formatPercent(p.Ranks.LevelRank)
}

func (p playerRanksTemplate) GetGamesPercent() string {
	return p.formatPercent(p.Ranks.GamesRank)
}

func (p playerRanksTemplate) GetBadgesPercent() string {
	return p.formatPercent(p.Ranks.BadgesRank)
}

func (p playerRanksTemplate) GetTimePercent() string {
	return p.formatPercent(p.Ranks.PlayTimeRank)
}

func (p playerRanksTemplate) GetFriendsPercent() string {
	return p.formatPercent(p.Ranks.FriendsRank)
}

func playerGamesAjaxHandler(w http.ResponseWriter, r *http.Request) {

	playerID := chi.URLParam(r, "id")

	playerIDInt, err := strconv.ParseInt(playerID, 10, 64)
	if err != nil {
		log.Err(err, r)
		return
	}

	query := DataTablesQuery{}
	err = query.fillFromURL(r.URL.Query())
	log.Err(err, r)

	//
	var wg sync.WaitGroup

	// Get apps
	var playerApps []db.PlayerApp

	wg.Add(1)
	go func() {

		defer wg.Done()

		client, ctx, err := db.GetDSClient()
		if err != nil {

			log.Err(err, r)
			return
		}

		columns := map[string]string{
			"0": "app_name",
			"1": "app_price",
			"2": "app_time",
			"3": "app_price_hour",
		}

		q := datastore.NewQuery(db.KindPlayerApp).Filter("player_id =", playerIDInt).Limit(100)
		q, err = query.setOrderOffsetDS(q, columns)
		if err != nil {

			log.Err(err, r)
			return
		}

		_, err = client.GetAll(ctx, q, &playerApps)
		log.Err(err, r)

	}()

	// Get total
	var total int
	wg.Add(1)
	go func() {

		defer wg.Done()

		player, err := db.GetPlayer(playerIDInt)
		if err != nil {
			log.Err(err, r)
			return
		}

		total = player.GamesCount

	}()

	// Wait
	wg.Wait()

	response := DataTablesAjaxResponse{}
	response.RecordsTotal = strconv.Itoa(total)
	response.RecordsFiltered = strconv.Itoa(total)
	response.Draw = query.Draw

	code := session.GetCountryCode(r)

	for _, v := range playerApps {
		response.AddRow(v.OutputForJSON(code))
	}

	response.output(w, r)

}

func playersUpdateAjaxHandler(w http.ResponseWriter, r *http.Request) {

	message, err, success := func(r *http.Request) (string, error, bool) {

		if helpers.IsBot(r.UserAgent()) {
			return "Bots can't update players", nil, false
		}

		idx, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			return "Invalid Player ID", err, false
		}

		if !db.IsValidPlayerID(idx) {
			return "Invalid Player ID", err, false
		}

		var message string

		player, err := db.GetPlayer(idx)
		if err == nil {
			message = "Updating player!"
		} else if err == datastore.ErrNoSuchEntity {
			message = "Looking for new player!"
		} else {
			log.Err(err, r)
			return "Error looking for player", err, false
		}

		updateType := db.PlayerUpdateManual
		if isAdmin(r) {
			message = "Admin update!"
			updateType = db.PlayerUpdateAdmin
		}

		if !player.ShouldUpdate(r.UserAgent(), updateType) {
			return "Player can't be updated yet", nil, false
		}

		err = queue.ProducePlayer(player.PlayerID)
		if err != nil {
			log.Err(err, r)
			return "Something has gone wrong", err, false
		}

		return message, err, true
	}(r)

	var response = PlayersUpdateResponse{
		Success: success,
		Toast:   message,
		Log:     err,
	}

	bytes, err := json.Marshal(response)
	log.Err(err, r)
	if err == nil {
		err = returnJSON(w, r, bytes)
		log.Err(err, r)
	}
}

func playersHistoryAjaxHandler(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	if id == "" {
		log.Err("invalid id", r)
		return
	}

	builder := influxql.NewBuilder()
	builder.AddSelect(`mean("level")`, "mean_level")
	builder.AddSelect(`mean("games")`, "mean_games")
	builder.AddSelect(`mean("badges")`, "mean_badges")
	builder.AddSelect(`mean("playtime")`, "mean_playtime")
	builder.AddSelect(`mean("friends")`, "mean_friends")
	builder.AddSelect(`mean("level_rank")`, "mean_level_rank")
	builder.AddSelect(`mean("games_rank")`, "mean_games_rank")
	builder.AddSelect(`mean("badges_rank")`, "mean_badges_rank")
	builder.AddSelect(`mean("playtime_rank")`, "mean_playtime_rank")
	builder.AddSelect(`mean("friends_rank")`, "mean_friends_rank")
	builder.SetFrom("GameDB", "alltime", "players")
	builder.AddWhere("player_id", "=", id)
	builder.AddWhere("time", ">", "now()-365d")
	builder.AddGroupByTime("1d")
	builder.SetFillNone()

	resp, err := db.InfluxQuery(builder.String())
	if err != nil {
		log.Err(err, r, builder.String())
		return
	}

	var hc db.HighChartsJson

	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {

		hc = db.InfluxResponseToHighCharts(resp.Results[0].Series[0])
	}

	b, err := json.Marshal(hc)
	if err != nil {
		log.Err(err, r)
		return
	}

	err = returnJSON(w, r, b)
	if err != nil {
		log.Err(err, r)
		return
	}
}

type PlayersUpdateResponse struct {
	Success bool   `json:"success"` // Red or green
	Toast   string `json:"toast"`   // Browser notification
	Log     error  `json:"log"`     // Console log
}
