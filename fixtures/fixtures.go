// Package fixtures holds the canonical synthetic dataset for the fraud
// investigation swarm demo. All data is fabricated; no real personal or
// financial data. The dataset is embedded from dataset.json so the servers
// and the alert driver share one source of truth.
package fixtures

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed dataset.json
var datasetJSON []byte

type Customer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AccountID  string `json:"accountId"`
	RiskRating string `json:"riskRating"`
	KYCStatus  string `json:"kycStatus"`
	Country    string `json:"country"`
}

type Account struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customerId"`
	Type       string  `json:"type"`
	OpenedDate string  `json:"openedDate"`
	Balance    float64 `json:"balance"`
	Currency   string  `json:"currency"`
}

type Transaction struct {
	ID                  string  `json:"id"`
	AccountID           string  `json:"accountId"`
	Timestamp           string  `json:"timestamp"`
	Amount              float64 `json:"amount"`
	Currency            string  `json:"currency"`
	Direction           string  `json:"direction"`
	CounterpartyName    string  `json:"counterpartyName"`
	CounterpartyAccount string  `json:"counterpartyAccount"`
	Channel             string  `json:"channel"`
	Country             string  `json:"country"`
}

type WatchlistEntry struct {
	Name      string `json:"name"`
	List      string `json:"list"`
	MatchType string `json:"matchType"`
	Program   string `json:"program"`
}

type DeviceSignal struct {
	CustomerID          string  `json:"customerId"`
	DeviceID            string  `json:"deviceId"`
	IP                  string  `json:"ip"`
	Geo                 string  `json:"geo"`
	IsNewDevice         bool    `json:"isNewDevice"`
	IPRiskScore         int     `json:"ipRiskScore"`
	LocationVelocityKmh float64 `json:"locationVelocityKmh"`
}

type IPGeo struct {
	IP        string `json:"ip"`
	Geo       string `json:"geo"`
	RiskScore int    `json:"riskScore"`
}

type Typology struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Indicators  []string `json:"indicators"`
}

type Alert struct {
	ID                   string   `json:"id"`
	CustomerID           string   `json:"customerId"`
	AccountID            string   `json:"accountId"`
	Type                 string   `json:"type"`
	RaisedAt             string   `json:"raisedAt"`
	TriggerReason        string   `json:"triggerReason"`
	TransactionIDs       []string `json:"transactionIds"`
	SuspectedTypologyIDs []string `json:"suspectedTypologyIds"`
	HeroCase             bool     `json:"heroCase"`
}

type dataset struct {
	Customers     []Customer       `json:"customers"`
	Accounts      []Account        `json:"accounts"`
	Transactions  []Transaction    `json:"transactions"`
	Watchlist     []WatchlistEntry `json:"watchlist"`
	PEP           []WatchlistEntry `json:"pep"`
	DeviceSignals []DeviceSignal   `json:"deviceSignals"`
	IPGeo         []IPGeo          `json:"ipGeo"`
	Typologies    []Typology       `json:"typologies"`
	Alerts        []Alert          `json:"alerts"`
}

var data dataset

func init() {
	if err := json.Unmarshal(datasetJSON, &data); err != nil {
		panic("fixtures: cannot parse dataset.json: " + err.Error())
	}
}

func CustomerByID(id string) (Customer, bool) {
	for _, c := range data.Customers {
		if c.ID == id {
			return c, true
		}
	}
	return Customer{}, false
}

func AccountByID(id string) (Account, bool) {
	for _, a := range data.Accounts {
		if a.ID == id {
			return a, true
		}
	}
	return Account{}, false
}

func TransactionsByAccount(accountID string) []Transaction {
	out := []Transaction{}
	for _, t := range data.Transactions {
		if t.AccountID == accountID {
			out = append(out, t)
		}
	}
	return out
}

func screen(entries []WatchlistEntry, name string) []WatchlistEntry {
	out := []WatchlistEntry{}
	q := strings.ToLower(strings.TrimSpace(name))
	for _, e := range entries {
		if strings.ToLower(e.Name) == q {
			out = append(out, e)
		}
	}
	return out
}

func ScreenSanctions(name string) []WatchlistEntry { return screen(data.Watchlist, name) }

func ScreenPEP(name string) []WatchlistEntry { return screen(data.PEP, name) }

func DeviceSignalsByCustomer(customerID string) []DeviceSignal {
	out := []DeviceSignal{}
	for _, d := range data.DeviceSignals {
		if d.CustomerID == customerID {
			out = append(out, d)
		}
	}
	return out
}

func IPGeoByIP(ip string) (IPGeo, bool) {
	for _, g := range data.IPGeo {
		if g.IP == ip {
			return g, true
		}
	}
	return IPGeo{}, false
}

func TypologiesByIDs(ids []string) []Typology {
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := []Typology{}
	for _, ty := range data.Typologies {
		if want[ty.ID] {
			out = append(out, ty)
		}
	}
	return out
}

// MatchTypologies returns every typology that shares at least one indicator
// with the supplied indicator list.
func MatchTypologies(indicators []string) []Typology {
	have := map[string]bool{}
	for _, i := range indicators {
		have[i] = true
	}
	out := []Typology{}
	for _, ty := range data.Typologies {
		for _, ind := range ty.Indicators {
			if have[ind] {
				out = append(out, ty)
				break
			}
		}
	}
	return out
}

func AllAlerts() []Alert { return data.Alerts }

func AlertByID(id string) (Alert, bool) {
	for _, a := range data.Alerts {
		if a.ID == id {
			return a, true
		}
	}
	return Alert{}, false
}
