package mongo

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gamedb/gamedb/pkg/log"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AppItem struct {
	AppID            int        `bson:"app_id"`
	Bundle           string     `bson:"bundle"`
	Commodity        bool       `bson:"commodity"`
	DateCreated      time.Time  `bson:"date_created"`
	Description      string     `bson:"description"`
	DisplayType      string     `bson:"display_type"`
	DropInterval     int        `bson:"drop_interval"`
	DropMaxPerWindow int        `bson:"drop_max_per_window"`
	Exchange         []string   `bson:"exchange"`
	Hash             string     `bson:"hash"`
	IconURL          string     `bson:"icon_url"`
	IconURLLarge     string     `bson:"icon_url_large"`
	ItemDefID        int        `bson:"item_def_id"`
	ItemQuality      string     `bson:"item_quality"`
	Marketable       bool       `bson:"marketable"`
	Modified         time.Time  `bson:"modified"`
	Name             string     `bson:"name"`
	Price            string     `bson:"price"`
	Promo            string     `bson:"promo"`
	Quantity         int        `bson:"quantity"`
	Tags             [][]string `bson:"tags"`
	Timestamp        time.Time  `bson:"timestamp"`
	Tradable         bool       `bson:"tradable"`
	Type             string     `bson:"type"`
	WorkshopID       int64      `bson:"workshop_id"`
}

func (a AppItem) BSON() (ret interface{}) {

	return M{
		"_id":                 a.getKey(),
		"app_id":              a.AppID,
		"bundle":              a.Bundle,
		"commodity":           a.Commodity,
		"date_created":        a.DateCreated,
		"description":         a.Description,
		"display_type":        a.DisplayType,
		"drop_interval":       a.DropInterval,
		"drop_max_per_window": a.DropMaxPerWindow,
		"exchange":            a.Exchange,
		"hash":                a.Hash,
		"icon_url":            a.IconURL,
		"icon_url_large":      a.IconURLLarge,
		"item_def_id":         a.ItemDefID,
		"item_quality":        a.ItemQuality,
		"marketable":          a.Marketable,
		"modified":            a.Modified,
		"name":                a.Name,
		"price":               a.Price,
		"promo":               a.Promo,
		"quantity":            a.Quantity,
		"tags":                a.Tags,
		"timestamp":           a.Timestamp,
		"tradable":            a.Tradable,
		"type":                a.Type,
		"workshop_id":         a.WorkshopID,
	}
}

func (a AppItem) getKey() string {
	return strconv.Itoa(a.AppID) + "-" + strconv.Itoa(a.ItemDefID)
}

func (a *AppItem) SetTags(tags string) {
	if tags != "" {
		split := strings.Split(tags, ";")
		for _, v := range split {
			split2 := strings.Split(v, ":")
			a.Tags = append(a.Tags, []string{split2[0], split2[1]})
		}
	}
}

func (a *AppItem) SetExchange(exchange string) {

	a.Exchange = []string{}

	split := strings.Split(exchange, ";")
	for _, v := range split {
		a.Exchange = append(a.Exchange, v)
	}
}

func (a *AppItem) Link() string {
	return "https://steamcommunity.com/market/listings/" + strconv.Itoa(a.AppID) + "/" + url.QueryEscape(a.Name)
}

func (a *AppItem) Image() string {

	params := url.Values{}
	params.Set("width", "256")
	params.Set("height", "256")
	params.Set("bg", "00FFFFFF")

	return "https://images.weserv.nl?" + params.Encode()
}

func GetAppItems(appID int, offset int64, limit int64, projection M) (items []AppItem, err error) {

	filter := M{
		"app_id": appID,
	}

	client, ctx, err := getMongo()
	if err != nil {
		return items, err
	}

	c := client.Database(MongoDatabase, options.Database()).Collection(CollectionAppItems.String())

	ops := options.Find().SetSort(D{{"workshop_id", 1}})
	if limit > 0 {
		ops.SetLimit(limit)
	}
	if offset > 0 {
		ops.SetSkip(offset)
	}
	if projection != nil {
		ops.SetProjection(projection)
	}

	cur, err := c.Find(ctx, filter, ops)
	if err != nil {
		return items, err
	}

	defer func() {
		err = cur.Close(ctx)
		log.Err(err)
	}()

	for cur.Next(ctx) {

		var item AppItem
		err := cur.Decode(&item)
		if err != nil {
			log.Err(err)
		}
		items = append(items, item)
	}

	return items, cur.Err()
}
