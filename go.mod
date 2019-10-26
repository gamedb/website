module github.com/gamedb/gamedb

go 1.13

require (
	cloud.google.com/go v0.47.0 // indirect
	cloud.google.com/go/logging v1.0.0
	cloud.google.com/go/pubsub v1.0.1
	github.com/Jleagle/go-durationfmt v0.0.0-20190307132420-e57bfad84057
	github.com/Jleagle/influxql v0.0.0-20190502115937-4ac053a1ed8e
	github.com/Jleagle/memcache-go v0.0.0-20191017205741-83516ce5e61a
	github.com/Jleagle/patreon-go v0.0.0-20190513114123-359f6ccef16d
	github.com/Jleagle/recaptcha-go v0.0.0-20190220085232-0e548dc7cc83
	github.com/Jleagle/session-go v0.0.0-20190515070633-3c8712426233
	github.com/Jleagle/sitemap-go v0.0.0-20190405195207-2bdddbb3bd50
	github.com/Jleagle/steam-go v0.0.0-20191015112215-cb5cd23f33ef
	github.com/Jleagle/unmarshal-go v0.0.0-20190815120521-15f0bd3950ff
	github.com/Jleagle/valve-data-format-go v0.0.0-20191018170405-419e8ff7cf85
	github.com/Philipp15b/go-steam v1.0.1-0.20190816133340-b04c5a83c1c0
	github.com/PuerkitoBio/goquery v1.5.0 // indirect
	github.com/ahmdrz/goinsta/v2 v2.4.4
	github.com/andybalholm/cascadia v1.1.0 // indirect
	github.com/antchfx/htmlquery v1.1.0 // indirect
	github.com/antchfx/xmlquery v1.1.0 // indirect
	github.com/antchfx/xpath v1.1.0 // indirect
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/beefsack/go-rate v0.0.0-20180408011153-efa7637bb9b6 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b // indirect
	github.com/buger/jsonparser v0.0.0-20191004114745-ee4c978eae7e // indirect
	github.com/bwmarrin/discordgo v0.20.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/derekstavis/go-qs v0.0.0-20180720192143-9eef69e6c4e7
	github.com/dghubble/go-twitter v0.0.0-20190719072343-39e5462e111f
	github.com/dghubble/oauth1 v0.6.0
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/didip/tollbooth/v5 v5.1.0
	github.com/djherbis/fscache v0.9.0
	github.com/dustin/go-humanize v1.0.0
	github.com/frustra/bbcode v0.0.0-20180807171629-48be21ce690c
	github.com/getkin/kin-openapi v0.2.0
	github.com/getsentry/sentry-go v0.3.0
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/cors v1.0.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gocolly/colly v1.2.0
	github.com/golang/groupcache v0.0.0-20191025150517-4a4ac3fbac33 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.1
	github.com/google/go-github/v27 v27.0.6
	github.com/gorilla/sessions v1.2.0
	github.com/gorilla/websocket v1.4.1
	github.com/gosimple/slug v1.9.0
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jinzhu/gorm v1.9.11
	github.com/jinzhu/now v1.1.1
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/justinas/nosurf v0.0.0-20190416172904-05988550ea18
	github.com/jzelinskie/geddit v0.0.0-20190913104144-95ef6806b073
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/logrusorgru/aurora v0.0.0-20191017060258-dc85c304c434
	github.com/mattn/go-runewidth v0.0.5 // indirect
	github.com/microcosm-cc/bluemonday v1.0.2
	github.com/mxpv/patreon-go v0.0.0-20190917022727-646111f1d983
	github.com/nicklaw5/helix v0.5.4
	github.com/nlopes/slack v0.6.0
	github.com/olekukonko/tablewriter v0.0.1 // indirect
	github.com/oschwald/maxminddb-golang v1.5.0
	github.com/pariz/gountries v0.0.0-20171019111738-adb00f6513a3
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/powerslacker/ratelimit v0.0.0-20190505003410-df2fcffc8e0d
	github.com/robfig/cron/v3 v3.0.0
	github.com/rollbar/rollbar-go v1.2.0
	github.com/russross/blackfriday v2.0.0+incompatible
	github.com/saintfish/chardet v0.0.0-20120816061221-3af4cd4741ca // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sendgrid/rest v2.4.1+incompatible
	github.com/sendgrid/sendgrid-go v3.5.0+incompatible
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/streadway/amqp v0.0.0-20190827072141-edfb9018d271
	github.com/tdewolff/minify/v2 v2.5.2
	github.com/temoto/robotstxt v1.1.1 // indirect
	github.com/tidwall/pretty v1.0.0 // indirect
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	github.com/yohcop/openid-go v1.0.0
	go.mongodb.org/mongo-driver v1.1.2
	go.opencensus.io v0.22.1 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/exp v0.0.0-20191024150812-c286b889502e // indirect
	golang.org/x/net v0.0.0-20191021144547-ec77196f6094 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20191026034945-b2104f82a97d // indirect
	google.golang.org/api v0.11.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/grpc v1.24.0
	gopkg.in/djherbis/atime.v1 v1.0.0 // indirect
	gopkg.in/djherbis/stream.v1 v1.2.0 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
	jaytaylor.com/html2text v0.0.0-20190408195923-01ec452cbe43
)
