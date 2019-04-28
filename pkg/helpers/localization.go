package helpers

import (
	"sort"
	"strings"

	"github.com/Jleagle/steam-go/steam"
	"github.com/gamedb/gamedb/pkg/log"
	"github.com/leekchan/accounting"
	"github.com/pariz/gountries"
)

var gountriesInstance = gountries.New()
var byCurrency = map[steam.CurrencyCode]Locale{}
var byCountry = map[steam.CountryCode]Locale{}
var accountingFormat = accounting.Accounting{
	Precision:      2,
	Format:         "%s %v",
	FormatNegative: "%s -%v",
}
var locales = []Locale{
	{CountryCode: steam.CountryAE, CurrencyCode: steam.CurrencyAED, CurrencySymbol: "AED", CountryName: "United Arab Emirates"},
	{CountryCode: steam.CountryAR, CurrencyCode: steam.CurrencyARS, CurrencySymbol: "$", CountryName: "Argentina"},
	{CountryCode: steam.CountryAU, CurrencyCode: steam.CurrencyAUD, CurrencySymbol: "$", CountryName: "Australia"},
	{CountryCode: steam.CountryBR, CurrencyCode: steam.CurrencyBRL, CurrencySymbol: "R$", CountryName: "Brazil"},
	{CountryCode: steam.CountryCA, CurrencyCode: steam.CurrencyCAD, CurrencySymbol: "$", CountryName: "Canada"},
	{CountryCode: steam.CountryCH, CurrencyCode: steam.CurrencyCHF, CurrencySymbol: "CHF", CountryName: "Switzerland"},
	{CountryCode: steam.CountryCL, CurrencyCode: steam.CurrencyCLP, CurrencySymbol: "$", CountryName: "Chile"},
	{CountryCode: steam.CountryCN, CurrencyCode: steam.CurrencyCNY, CurrencySymbol: "¥", CountryName: "China", Enabled: true},
	{CountryCode: steam.CountryCO, CurrencyCode: steam.CurrencyCOP, CurrencySymbol: "$", CountryName: "Colombia"},
	{CountryCode: steam.CountryCR, CurrencyCode: steam.CurrencyCRC, CurrencySymbol: "₡", CountryName: "Costa Rica"},
	{CountryCode: steam.CountryDE, CurrencyCode: steam.CurrencyEUR, CurrencySymbol: "€", CountryName: "Germany", Enabled: true},
	{CountryCode: steam.CountryGB, CurrencyCode: steam.CurrencyGBP, CurrencySymbol: "£", CountryName: "United Kingdom", Enabled: true},
	{CountryCode: steam.CountryHK, CurrencyCode: steam.CurrencyHKD, CurrencySymbol: "$", CountryName: "Hong Kong"},
	{CountryCode: steam.CountryIL, CurrencyCode: steam.CurrencyILS, CurrencySymbol: "₪", CountryName: "Israel"},
	{CountryCode: steam.CountryID, CurrencyCode: steam.CurrencyIDR, CurrencySymbol: "Rp", CountryName: "Indonesia"},
	{CountryCode: steam.CountryIN, CurrencyCode: steam.CurrencyINR, CurrencySymbol: "₹", CountryName: "India"},
	{CountryCode: steam.CountryJP, CurrencyCode: steam.CurrencyJPY, CurrencySymbol: "¥", CountryName: "Japan"},
	{CountryCode: steam.CountryKR, CurrencyCode: steam.CurrencyKRW, CurrencySymbol: "₩", CountryName: "South Korea"},
	{CountryCode: steam.CountryKW, CurrencyCode: steam.CurrencyKWD, CurrencySymbol: "KWD", CountryName: "Kuwait"},
	{CountryCode: steam.CountryKZ, CurrencyCode: steam.CurrencyKZT, CurrencySymbol: "₸", CountryName: "Kazakhstan"},
	{CountryCode: steam.CountryMX, CurrencyCode: steam.CurrencyMXN, CurrencySymbol: "$", CountryName: "Mexico"},
	{CountryCode: steam.CountryMY, CurrencyCode: steam.CurrencyMYR, CurrencySymbol: "RM", CountryName: "Malaysia"},
	{CountryCode: steam.CountryNO, CurrencyCode: steam.CurrencyNOK, CurrencySymbol: "kr", CountryName: "Norway"},
	{CountryCode: steam.CountryNZ, CurrencyCode: steam.CurrencyNZD, CurrencySymbol: "$", CountryName: "New Zealand"},
	{CountryCode: steam.CountryPE, CurrencyCode: steam.CurrencyPEN, CurrencySymbol: "PEN", CountryName: "Peru"},
	{CountryCode: steam.CountryPH, CurrencyCode: steam.CurrencyPHP, CurrencySymbol: "₱", CountryName: "Philippines"},
	{CountryCode: steam.CountryPL, CurrencyCode: steam.CurrencyPLN, CurrencySymbol: "zł", CountryName: "Poland"},
	{CountryCode: steam.CountryQA, CurrencyCode: steam.CurrencyQAR, CurrencySymbol: "QAR", CountryName: "Qatar"},
	{CountryCode: steam.CountryRU, CurrencyCode: steam.CurrencyRUB, CurrencySymbol: "₽", CountryName: "Russia", Enabled: true},
	{CountryCode: steam.CountrySA, CurrencyCode: steam.CurrencySAR, CurrencySymbol: "SAR", CountryName: "Saudi Arabia"},
	{CountryCode: steam.CountrySG, CurrencyCode: steam.CurrencySGD, CurrencySymbol: "$", CountryName: "Singapore"},
	{CountryCode: steam.CountryTH, CurrencyCode: steam.CurrencyTHB, CurrencySymbol: "฿", CountryName: "Thailand"},
	{CountryCode: steam.CountryTR, CurrencyCode: steam.CurrencyTRY, CurrencySymbol: "₺", CountryName: "Turkey"},
	{CountryCode: steam.CountryTW, CurrencyCode: steam.CurrencyTWD, CurrencySymbol: "$", CountryName: "Taiwan"},
	{CountryCode: steam.CountryUA, CurrencyCode: steam.CurrencyUAH, CurrencySymbol: "₴", CountryName: "Ukraine"},
	{CountryCode: steam.CountryUS, CurrencyCode: steam.CurrencyUSD, CurrencySymbol: "$", CountryName: "United States", Enabled: true},
	{CountryCode: steam.CountryUY, CurrencyCode: steam.CurrencyUYU, CurrencySymbol: "$", CountryName: "Uruguay"},
	{CountryCode: steam.CountryVN, CurrencyCode: steam.CurrencyVND, CurrencySymbol: "₫", CountryName: "Vietnam"},
	{CountryCode: steam.CountryZA, CurrencyCode: steam.CurrencyZAR, CurrencySymbol: "R", CountryName: "South Africa"},
}

func init() {
	for _, v := range locales {
		byCurrency[v.CurrencyCode] = v
		byCountry[v.CountryCode] = v
	}
}

func GetLocaleFromCurrency(code steam.CurrencyCode) (loc Locale, err error) {

	if val, ok := byCurrency[code]; ok {
		return val, err
	}
	return loc, err
}

func GetLocaleFromCountry(code steam.CountryCode) (loc Locale, err error) {

	if val, ok := byCountry[code]; ok {
		return val, err
	}
	return loc, err
}

func GetActiveCountries() (ret []steam.CountryCode) {
	for _, v := range locales {
		if v.Enabled {
			ret = append(ret, v.CountryCode)
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return steam.Countries[ret[i]] < steam.Countries[ret[j]]
	})

	return ret
}

type Locale struct {
	CountryCode    steam.CountryCode
	CountryName    string
	CurrencyCode   steam.CurrencyCode
	CurrencySymbol string
	Enabled        bool
}

func (l Locale) Format(cents int) string {

	f := accountingFormat
	f.Symbol = l.CurrencySymbol

	return strings.TrimSpace(f.FormatMoney(float64(cents) / 100))
}

func (l Locale) FormatFloat(amount float64) string {

	f := accountingFormat
	f.Symbol = l.CurrencySymbol

	return strings.TrimSpace(f.FormatMoney(amount))
}

// For player countries
func CountryCodeToName(code string) string {

	if code == "" {
		return code
	} else if code == "BQ" {
		return "Bonaire, Sint Eustatius and Saba"
	} else if code == "SH" {
		return "Saint Helena"
	} else if code == "XK" {
		return "Kosovo"
	}

	country, err := gountriesInstance.FindCountryByAlpha(code)
	if err != nil {
		log.Err(err)
		return code
	}

	return country.Name.Common
}
