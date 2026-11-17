package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Quote struct {
	Rate   float64
	Time   time.Time
	Source string
}

type Provider interface {
	Name() string
	Fetch(pairs []CurrencyPair) (map[string]Quote, error)
}

var providers = []Provider{
	swissquoteProvider{},
	coinbaseProvider{},
	frankfurterProvider{},
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func getJSON(url string, v any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "FXTray/"+version)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func fetchQuotes(pairs []CurrencyPair) (map[string]Quote, error) {
	result := map[string]Quote{}
	missing := pairs
	var errs []string

	for _, p := range providers {
		if len(missing) == 0 {
			break
		}
		got, err := p.Fetch(missing)
		if err != nil {
			errs = append(errs, p.Name()+": "+err.Error())
		}
		var still []CurrencyPair
		for _, cp := range missing {
			key := pairKey(cp.From, cp.To)
			if q, ok := got[key]; ok && q.Rate > 0 {
				result[key] = q
			} else {
				still = append(still, cp)
			}
		}
		missing = still
	}

	if len(result) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
	}
	return result, nil
}

type swissquoteProvider struct{}

func (swissquoteProvider) Name() string { return "Swissquote" }

type sqQuote struct {
	SpreadProfilePrices []struct {
		Bid float64 `json:"bid"`
		Ask float64 `json:"ask"`
	} `json:"spreadProfilePrices"`
	Ts int64 `json:"ts"`
}

func sqMid(url string) (float64, time.Time, error) {
	var rows []sqQuote
	if err := getJSON(url, &rows); err != nil {
		return 0, time.Time{}, err
	}
	best, mid, ts := -1.0, 0.0, int64(0)
	for _, r := range rows {
		for _, sp := range r.SpreadProfilePrices {
			if sp.Bid <= 0 || sp.Ask <= 0 {
				continue
			}
			if s := sp.Ask - sp.Bid; best < 0 || s < best {
				best, mid, ts = s, (sp.Bid+sp.Ask)/2, r.Ts
			}
		}
	}
	if mid == 0 {
		return 0, time.Time{}, fmt.Errorf("no prices")
	}
	return mid, time.UnixMilli(ts), nil
}

func (swissquoteProvider) Fetch(pairs []CurrencyPair) (map[string]Quote, error) {
	const base = "https://forex-data-feed.swissquote.com/public-quotes/bboquotes/instrument/"
	out := map[string]Quote{}
	var lastErr error

	for _, p := range pairs {
		from, to := strings.ToUpper(p.From), strings.ToUpper(p.To)
		key := pairKey(from, to)

		if mid, ts, err := sqMid(base + from + "/" + to); err == nil {
			out[key] = Quote{mid, ts, "Swissquote"}
			continue
		}
		mid, ts, err := sqMid(base + to + "/" + from)
		if err != nil {
			lastErr = err
			continue
		}
		out[key] = Quote{1 / mid, ts, "Swissquote"}
	}
	if len(out) == 0 {
		return out, lastErr
	}
	return out, nil
}

type coinbaseProvider struct{}

func (coinbaseProvider) Name() string { return "Coinbase" }

func (coinbaseProvider) Fetch(pairs []CurrencyPair) (map[string]Quote, error) {
	out := map[string]Quote{}
	var lastErr error

	for _, base := range uniqueBases(pairs) {
		var resp struct {
			Data struct {
				Rates map[string]string `json:"rates"`
			} `json:"data"`
		}
		if err := getJSON("https://api.coinbase.com/v2/exchange-rates?currency="+base, &resp); err != nil {
			lastErr = err
			continue
		}
		now := time.Now()
		for _, p := range pairs {
			if strings.ToUpper(p.From) != base {
				continue
			}
			s, ok := resp.Data.Rates[strings.ToUpper(p.To)]
			if !ok {
				continue
			}
			if r, err := strconv.ParseFloat(s, 64); err == nil {
				out[pairKey(p.From, p.To)] = Quote{r, now, "Coinbase"}
			}
		}
	}
	if len(out) == 0 {
		return out, lastErr
	}
	return out, nil
}

type frankfurterRate struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

type frankfurterProvider struct{}

func (frankfurterProvider) Name() string { return "Frankfurter" }

func (frankfurterProvider) Fetch(pairs []CurrencyPair) (map[string]Quote, error) {
	out := map[string]Quote{}
	var lastErr error

	for _, base := range uniqueBases(pairs) {
		var rows []frankfurterRate
		if err := getJSON("https://api.frankfurter.dev/v2/rates?base="+base, &rows); err != nil {
			lastErr = err
			continue
		}
		byQuote := map[string]frankfurterRate{}
		for _, r := range rows {
			byQuote[strings.ToUpper(r.Quote)] = r
		}
		for _, p := range pairs {
			if strings.ToUpper(p.From) != base {
				continue
			}
			if r, ok := byQuote[strings.ToUpper(p.To)]; ok {
				d, _ := time.Parse("2006-01-02", r.Date)
				out[pairKey(p.From, p.To)] = Quote{r.Rate, d, "Frankfurter"}
			}
		}
	}
	if len(out) == 0 {
		return out, lastErr
	}
	return out, nil
}

func uniqueBases(pairs []CurrencyPair) []string {
	set := map[string]struct{}{}
	for _, p := range pairs {
		set[strings.ToUpper(p.From)] = struct{}{}
	}
	bases := make([]string, 0, len(set))
	for b := range set {
		bases = append(bases, b)
	}
	sort.Strings(bases)
	return bases
}
