package pages

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/Jleagle/influxql"
	"github.com/gamedb/gamedb/pkg/helpers"
	"github.com/gamedb/gamedb/pkg/log"
	"github.com/gamedb/gamedb/pkg/mongo"
	"github.com/gamedb/gamedb/pkg/queue"
	"github.com/gamedb/gamedb/pkg/sql"
	"github.com/go-chi/chi"
)

func GroupRouter() http.Handler {

	r := chi.NewRouter()
	r.Get("/", groupHandler)
	r.Get("/time.json", groupAjaxHandler)
	r.Get("/{slug}", groupHandler)
	return r
}

func groupHandler(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	if id == "" {
		returnErrorTemplate(w, r, errorTemplate{Code: 400, Message: "Invalid group ID"})
		return
	}

	// if !db.IsValidAppID(idx) {
	// 	returnErrorTemplate(w, r, errorTemplate{Code: 400, Message: "Invalid bundle ID: " + id})
	// 	return
	// }

	// Get group
	group, err := mongo.GetGroup(id)
	if err != nil {

		if err == sql.ErrRecordNotFound {
			returnErrorTemplate(w, r, errorTemplate{Code: 400, Message: "Sorry but we can not find this group"})
			return
		}

		returnErrorTemplate(w, r, errorTemplate{Code: 500, Message: "There was an issue retrieving the group", Error: err})
		return
	}

	t := groupTemplate{}
	t.fill(w, r, group.GetName(), "")
	t.addAssetHighCharts()
	t.Canonical = group.GetPath()

	// Update group
	func() {

		if helpers.IsBot(r.UserAgent()) {
			return
		}

		if group.UpdatedAt.Unix() > time.Now().Add(time.Hour * -1).Unix() {
			return
		}

		err = queue.ProduceGroup([]string{group.ID64})
		if err != nil {
			log.Err(err, r)
		} else {
			t.addToast(Toast{Title: "Update", Message: "Group has been queued for an update"})
		}
	}()

	t.Group = group
	t.Summary = helpers.RenderHTMLAndBBCode(group.Summary)
	t.Group.Error = strings.Replace(t.Group.Error, "Click here for information on how to report groups on Steam.", "", 1)

	err = returnTemplate(w, r, "group", t)
	log.Err(err, r)
}

type groupTemplate struct {
	GlobalTemplate
	Group   mongo.Group
	Summary template.HTML
}

func groupAjaxHandler(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	if id == "" {
		log.Err("invalid id: "+id, r)
		return
	}

	if len(id) != 18 {
		log.Err("invalid id: "+id, r)
		return
	}

	if !helpers.IsValidGroupID(id) {
		log.Err("invalid id: "+id, r)
		return
	}

	builder := influxql.NewBuilder()
	builder.AddSelect(`max("members_count")`, "max_members_count")
	// builder.AddSelect(`max("members_in_chat")`, "max_members_in_chat")
	// builder.AddSelect(`max("members_in_game")`, "max_members_in_game")
	// builder.AddSelect(`max("members_online")`, "max_members_online")
	builder.SetFrom(helpers.InfluxGameDB, helpers.InfluxRetentionPolicyAllTime.String(), helpers.InfluxMeasurementGroups.String())
	builder.AddWhere("group_id", "=", id)
	// builder.AddWhere("time", ">", "now()-365d")
	builder.AddGroupByTime("1h")
	builder.SetFillLinear()

	resp, err := helpers.InfluxQuery(builder.String())
	if err != nil {
		log.Err(err, r, builder.String())
		return
	}

	var hc helpers.HighChartsJson

	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {

		hc = helpers.InfluxResponseToHighCharts(resp.Results[0].Series[0])
	}

	err = returnJSON(w, r, hc)
	log.Err(err, r)
}
