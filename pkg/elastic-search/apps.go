package elastic_search

import (
	"encoding/json"
	"strconv"

	"github.com/Jleagle/steam-go/steamapi"
	"github.com/gamedb/gamedb/pkg/helpers"
	"github.com/gamedb/gamedb/pkg/log"
	"github.com/olivere/elastic/v7"
)

type App struct {
	ID          int                   `json:"id"`
	Name        string                `json:"name"`
	NameMarked  string                `json:"name_marked"`
	Players     int                   `json:"players"`
	Aliases     []string              `json:"aliases"`
	Icon        string                `json:"icon"`
	Followers   int                   `json:"followers"`
	ReviewScore float64               `json:"score"`
	Prices      helpers.ProductPrices `json:"prices"`
	Tags        []int                 `json:"tags"`
	Genres      []int                 `json:"genres"`
	Categories  []int                 `json:"categories"`
	Publishers  []int                 `json:"publishers"`
	Developers  []int                 `json:"developers"`
	Type        string                `json:"type"`
	Platforms   []string              `json:"platforms"`
	Score       float64               `json:"-"`
}

func (app App) GetName() string {
	return helpers.GetAppName(app.ID, app.Name)
}

func (app App) GetIcon() string {
	return helpers.GetAppIcon(app.ID, app.Icon)
}

func (app App) GetPath() string {
	return helpers.GetAppPath(app.ID, app.Name)
}

func (app App) GetCommunityLink() string {
	return helpers.GetAppCommunityLink(app.ID)
}

func SearchApps(limit int, offset int, search string, totals bool, highlights bool, aggregation bool) (apps []App, aggregations map[string]map[string]int64, total int64, err error) {

	client, ctx, err := GetElastic()
	if err != nil {
		return apps, aggregations, 0, err
	}

	searchService := client.Search().
		Index(IndexApps).
		From(offset).
		Size(limit)

	if search != "" {

		var search2 = helpers.RegexNonAlphaNumeric.ReplaceAllString(search, "")

		searchService.Query(elastic.NewBoolQuery().
			Must(
				elastic.NewBoolQuery().MinimumNumberShouldMatch(1).Should(
					elastic.NewTermQuery("id", search2).Boost(5),
					elastic.NewMatchQuery("name", search).Boost(1),
					elastic.NewTermQuery("aliases", search2).Boost(1),
					elastic.NewPrefixQuery("name", search).Boost(0.2),
				),
			).
			Should(
				elastic.NewFunctionScoreQuery().
					AddScoreFunc(elastic.NewFieldValueFactorFunction().Modifier("sqrt").Field("players").Factor(0.005)),
			),
		)
	}

	if aggregation {
		searchService.Aggregation("type", elastic.NewTermsAggregation().Field("type").Size(10).OrderByCountDesc())
	}

	if totals {
		searchService.TrackTotalHits(true)
	}

	if highlights {
		searchService.Highlight(elastic.NewHighlight().Field("name").PreTags("<mark>").PostTags("</mark>"))
	}

	searchResult, err := searchService.Do(ctx)
	if err != nil {
		return apps, aggregations, 0, err
	}

	if aggregation {
		aggregations = make(map[string]map[string]int64, len(searchResult.Aggregations))
		for k := range searchResult.Aggregations {
			a, ok := searchResult.Aggregations.Terms(k)
			if ok {
				aggregations[k] = make(map[string]int64, len(a.Buckets))
				for _, v := range a.Buckets {
					aggregations[k][*v.KeyAsString] = v.DocCount
				}
			}
		}
	}

	for _, hit := range searchResult.Hits.Hits {

		var app = App{}

		err := json.Unmarshal(hit.Source, &app)
		if err != nil {
			log.Err(err)
			continue
		}

		if hit.Score != nil {
			app.Score = *hit.Score
		}

		if highlights {

			app.NameMarked = app.Name
			if val, ok := hit.Highlight["name"]; ok {
				if len(val) > 0 {
					app.NameMarked = val[0]
				}
			}
		}

		apps = append(apps, app)
	}

	return apps, aggregations, searchResult.TotalHits(), err
}

func IndexApp(a App) error {

	err := IndexGlobalItem(Global{ID: strconv.Itoa(a.ID), Name: a.Name, Icon: a.Icon, Type: GlobalTypeApp})
	log.Err(err)

	return indexDocument(IndexApps, strconv.Itoa(a.ID), a)
}

//noinspection GoUnusedExportedFunction
func DeleteAndRebuildAppsIndex() {

	var priceProperties = map[string]interface{}{}
	for _, v := range steamapi.ProductCCs {
		priceProperties[string(v)] = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"currency":         fieldTypeKeyword,
				"discount_percent": fieldTypeInteger,
				"final":            fieldTypeInteger,
				"individual":       fieldTypeInteger,
				"initial":          fieldTypeInteger,
			},
		}
	}

	var mapping = map[string]interface{}{
		"settings": settings,
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":         fieldTypeKeyword,
				"name":       fieldTypeText,
				"aliases":    fieldTypeText,
				"players":    fieldTypeInteger,
				"icon":       fieldTypeDisabled,
				"followers":  fieldTypeInteger,
				"score":      fieldTypeHalfFloat,
				"prices":     map[string]interface{}{"type": "object", "properties": priceProperties},
				"tags":       fieldTypeKeyword,
				"genres":     fieldTypeKeyword,
				"categories": fieldTypeKeyword,
				"publishers": fieldTypeKeyword,
				"developers": fieldTypeKeyword,
				"type":       fieldTypeKeyword,
				"platforms":  fieldTypeKeyword,
			},
		},
	}

	rebuildIndex(IndexApps, mapping)
}